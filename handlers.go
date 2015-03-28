package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"log"

	"golang.org/x/codereview/patch"
)

type handlers struct {
	baseDir dirTree
	changes map[string]*DiffStats
}

func newHandlers(baseDir *os.File, basePath string, patchSet *patch.Set) (*handlers, error) {
	tree, err := dirToTree(baseDir, basePath)
	if err != nil {
		return nil, err
	}

	changes := applyChangesToTree(patchSet, tree)

	return &handlers{
		baseDir: tree,
		changes: changes,
	}, nil
}

func (h *handlers) root(w http.ResponseWriter, req *http.Request) {
	path := strings.Split(req.URL.Path, "/")[1:]
	for i := 1; i < len(path); i++ {
		if path[i] == "" {
			path = append(path[:i], path[i+1:]...)
			i--
		}
	}
	if req.URL.Path != "/" {
		path = append(path, "")
	}
	var levels []treeDataLevels
	pathString := ""
	tree := h.baseDir
	for _, part := range path {
		var (
			entries []treeDataLevelsEntries
			body    string
			err     error
		)

		switch t := tree.(type) {
		case *dirFolder:
			tree = nil
			dir := t
			for _, entry := range dir.Entries {
				_, isDir := entry.(*dirFolder)
				isOpen := entry.Name() == part
				diffStats := h.changes[(pathString + "/" + entry.Name())[1:]]
				if diffStats == nil {
					diffStats = &DiffStats{}
				}
				entries = append(entries, treeDataLevelsEntries{
					Name:      entry.Name(),
					IsDir:     isDir,
					IsOpen:    isOpen,
					DiffStats: *diffStats,
				})
				if isOpen {
					tree = entry
				}
			}
		case *dirFile:
			tree = nil
			body, err = t.Contents()
			if err != nil {
				fmt.Println(err)
				return
			}
			_, ok := h.changes[pathString[1:]]
			if !ok {
				padded := ""
				for _, line := range strings.Split(body, "\n") {
					padded += "  " + line + "\n"
				}
				body = padded
			}
		case nil:
			http.NotFound(w, req)
			return
		}

		levels = append(levels, treeDataLevels{
			Path:    pathString,
			Entries: entries,
			Body:    body,
		})

		pathString += "/" + part
	}

	err := templates.ExecuteTemplate(w, "full", &fullData{
		Title:    "navpatch - " + pathString,
		TreeData: treeData{Levels: levels},
	})
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
