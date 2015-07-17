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
	BaseDir  TreeEntry
	Changes  map[string]*DiffStats
}

func NewNavigator(r Repository, rawPatch []byte) (*Navigator, error) {
	patchSet, err := patch.Parse(rawPatch)
	if err != nil {
		return nil, fmt.Errorf("parsing patch: %s", err)
	}

	tree, err := r.GetTree()
	if err != nil {
		return nil, err
	}

	changes := ApplyChangesToTree(patchSet, tree)

	return &Navigator{
		BasePath: "foo",
		RawPatch: rawPatch,
		BaseDir:  tree,
		Changes:  changes,
	}, nil
}

func (nav *Navigator) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	nav.HandleRoot(w, req, req.URL.Path, "")
}

func (nav *Navigator) HandleRoot(w http.ResponseWriter, req *http.Request, path string, linksPrefix string) {
	levels, err := nav.makeTplLevels(path)
	if err == errBadPath {
		http.NotFound(w, req)
		return
	} else if err != nil && err != patch.ErrPatchFailure {
		log.Println(path, err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	err = templates.ExecuteTemplate(w, "full", &tplFullData{
		Title: "navpatch - " + path,
		TreeData: tplTreeData{
			Levels:      levels,
			Nav:         nav,
			LinksPrefix: linksPrefix,
		},
		Nav: nav,
	})
	if err != nil {
		log.Println(path, err)
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
	tree TreeEntry,
) (
	level tplTreeDataLevel,
	nextTree TreeEntry,
	err error,
) {
	level.Path = lvlPath

	if tree == nil {
		err = errBadPath
		return
	}

	nextTree = nil

	switch t := tree.(type) {
	case *TreeFolder:
		dir := t
		for _, entry := range dir.Entries {
			_, isDir := entry.(*TreeFolder)
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
	case *TreeFile:
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
