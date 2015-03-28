package navpatch

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/codereview/patch"

	"log"
)

type Navigator struct {
	BasePath string
	RawPatch []byte
	BaseDir  DirTree
	Changes  map[string]*DiffStats
}

func NewNavigator(baseDir string, rawPatch []byte) (*Navigator, error) {
	patchSet, err := patch.Parse(rawPatch)
	if err != nil {
		return nil, fmt.Errorf("parsing patch: %s", err)
	}

	tree, err := DirPathToTree(baseDir)
	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %s", baseDir, err)
	}

	changes := ApplyChangesToTree(patchSet, tree)

	return &Navigator{
		BasePath: baseDir,
		RawPatch: rawPatch,
		BaseDir:  tree,
		Changes:  changes,
	}, nil
}

func (nav *Navigator) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	nav.HandleRoot(w, req)
}

func (nav *Navigator) HandleRoot(w http.ResponseWriter, req *http.Request) {
	levels, err := nav.makeTplLevels(req.URL.Path)
	if err == errBadPath {
		http.NotFound(w, req)
		return
	} else if err != nil && err != patch.ErrPatchFailure {
		log.Println(req.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	err = templates.ExecuteTemplate(w, "full", &tplFullData{
		Title:    "navpatch - " + req.URL.Path,
		TreeData: tplTreeData{Levels: levels, Nav: nav},
		Nav:      nav,
	})
	if err != nil {
		log.Println(req.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Splits a request path in parts. Empty parts are discarded.
// The last part is always the empty string.
func splitReqPath(path string) []string {
	parts := strings.Split(path, "/")[1:]
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			parts = append(parts[:i], parts[i+1:]...)
			i--
		}
	}
	if path != "/" {
		parts = append(parts, "")
	}
	return parts
}

func (nav *Navigator) makeTplLevels(path string) ([]tplTreeDataLevel, error) {
	pathParts := splitReqPath(path)

	var levels []tplTreeDataLevel

	lvlPath := ""
	tree := nav.BaseDir
	for _, part := range pathParts {
		level, nextTree, err := nav.makeTplLevel(lvlPath, part, tree)
		levels = append(levels, level)
		if err != nil {
			return levels, err
		}

		lvlPath += "/" + part
		tree = nextTree
	}

	return levels, nil
}

func (nav *Navigator) makeTplLevel(
	lvlPath string,
	part string,
	tree DirTree,
) (
	level tplTreeDataLevel,
	nextTree DirTree,
	err error,
) {
	level.Path = lvlPath

	if tree == nil {
		err = errBadPath
		return
	}

	nextTree = nil

	switch t := tree.(type) {
	case *DirFolder:
		dir := t
		for _, entry := range dir.Entries {
			_, isDir := entry.(*DirFolder)
			isOpen := entry.Name() == part
			diffStats := nav.Changes[(lvlPath + "/" + entry.Name())[1:]]
			if diffStats == nil {
				diffStats = &DiffStats{}
			}
			level.Entries = append(level.Entries, tplTreeDataLevelEntry{
				Name:      entry.Name(),
				IsDir:     isDir,
				IsOpen:    isOpen,
				DiffStats: *diffStats,
			})
			if isOpen {
				nextTree = entry
			}
		}
	case *DirFile:
		level.Body, err = t.Contents()
		if err != nil {
			level.Error = err
			break
		}
		_, ok := nav.Changes[lvlPath[1:]]
		if !ok {
			padded := ""
			for _, line := range strings.Split(level.Body, "\n") {
				padded += "  " + line + "\n"
			}
			level.Body = padded
		}
	}

	return
}

var errBadPath = errors.New("bad path.")