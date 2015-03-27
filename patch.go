package main

import (
	"strings"

	"github.com/aryann/difflib"
	"golang.org/x/codereview/patch"
)

type DiffStats struct {
	Additions int
	Deletions int
	Added     bool
	Removed   bool
	OldMode   int
	NewMode   int
	Chunks    patch.TextDiff
}

func applyChangesToTree(patchSet *patch.Set, tree dirTree) map[string]*DiffStats {
	changes := map[string]*DiffStats{}

	for _, pf := range patchSet.File {
		var stats *DiffStats
		diff, ok := pf.Diff.(patch.TextDiff)
		if !ok {
			// TODO: Git binary diffs.
			continue
		}
		switch pf.Verb {
		case patch.Add:
			stats = statsFromDiff(diff)
			stats.Added = true
			stats.Chunks = diff
			changes[pf.Dst] = stats
			addFileToTree(strings.Split(pf.Dst, "/"), tree, diff)
		case patch.Edit:
			stats = statsFromDiff(diff)
			changes[pf.Dst] = stats
			stats.Chunks = diff
			editFileInTree(strings.Split(pf.Dst, "/"), tree, diff)
		}
	}

	addFoldersToChanges(changes)

	return changes
}

func addFoldersToChanges(changes map[string]*DiffStats) {
	ks := []string{}
	for k, _ := range changes {
		ks = append(ks, k)
	}
	for _, k := range ks {
		diff := changes[k]
		parts := strings.Split(k, "/")
		for i := len(parts) - 2; i >= 0; i-- {
			folder := strings.Join(parts[:i+1], "/")
			prevDiff, ok := changes[folder]
			if !ok {
				prevDiff = &DiffStats{}
				changes[folder] = prevDiff
			}

			prevDiff.Additions += diff.Additions
			prevDiff.Deletions += diff.Deletions
		}
	}
}

func addFileToTree(path []string, tree dirTree, diff patch.TextDiff) {
	changeFileInTree(path, tree, func(folder *dirFolder) {
		ret, err := applyPatch(diff, "")
		entry := &dirFile{
			name: path[len(path)-1],
			Contents: func() (string, error) {
				return ret, err
			},
		}
		folder.Entries = append(folder.Entries, entry)
	})
}

func editFileInTree(path []string, tree dirTree, diff patch.TextDiff) {
	changeFileInTree(path, tree, func(folder *dirFolder) {
		for _, entry := range folder.Entries {
			if entry.Name() == path[len(path)-1] {
				prevContents := entry.(*dirFile).Contents
				entry.(*dirFile).Contents = func() (string, error) {
					prev, err := prevContents()
					if err != nil {
						return "", err
					}
					ret, err := applyPatch(diff, prev)
					entry.(*dirFile).Contents = func() (string, error) {
						return ret, err
					}
					return ret, err
				}
				break
			}
		}
	})
}

func changeFileInTree(path []string, tree dirTree, changeCallback func(*dirFolder)) {
	if len(path) == 0 {
		return
	}
	name := path[0]
	isDir := len(path) > 1

	switch t := tree.(type) {
	case *dirFolder:
		if isDir {
			var entry dirTree
			for _, oldEntry := range t.Entries {
				if oldEntry.Name() == name {
					entry = oldEntry
					break
				}
			}
			if entry == nil {
				entry = &dirFolder{
					name:    name,
					Entries: []dirTree{},
				}
				t.Entries = append(t.Entries, entry)
			}
			changeFileInTree(path[1:], entry, changeCallback)
		} else {
			changeCallback(t)
		}
	case *dirFile:
		// TODO?
	}
}

func statsFromDiff(diff patch.TextDiff) *DiffStats {
	stats := DiffStats{Chunks: diff}

	for _, chunk := range diff {
		atoms := difflib.Diff(
			strings.Split(string(chunk.Old), "\n"),
			strings.Split(string(chunk.New), "\n"),
		)
		for _, a := range atoms {
			if a.Delta == difflib.LeftOnly {
				stats.Deletions++
			} else if a.Delta == difflib.RightOnly {
				stats.Additions++
			}
		}
	}

	return &stats
}

func applyPatch(diff patch.TextDiff, prev string) (string, error) {
	curr, err := diff.Apply([]byte(prev))
	if err != nil {
		return "", err
	}

	chunks := difflib.Diff(strings.Split(prev, "\n"), strings.Split(string(curr), "\n"))
	ret := ""

	for _, ch := range chunks {
		ret += ch.String() + "\n"
	}

	return ret, nil
}
