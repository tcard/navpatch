package navpatch

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

func ApplyChangesToTree(patchSet *patch.Set, tree DirTree) map[string]*DiffStats {
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
			changes[pf.Dst] = stats
			addFileToTree(strings.Split(pf.Dst, "/"), tree, diff)
		case patch.Edit:
			stats = statsFromDiff(diff)
			changes[pf.Dst] = stats
			editFileInTree(strings.Split(pf.Dst, "/"), tree, diff)
		case patch.Delete:
			stats = statsFromDiff(diff)
			stats.Removed = true
			changes[pf.Src] = stats
			editFileInTree(strings.Split(pf.Src, "/"), tree, diff)
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

func addFileToTree(path []string, tree DirTree, diff patch.TextDiff) {
	changeFileInTree(path, tree, func(folder *DirFolder) {
		ret, err := applyPatch(diff, "")
		entry := &DirFile{
			name: path[len(path)-1],
			contents: func() (string, error) {
				return ret, err
			},
		}
		folder.Entries = append(folder.Entries, entry)
	})
}

func editFileInTree(path []string, tree DirTree, diff patch.TextDiff) {
	changeFileInTree(path, tree, func(folder *DirFolder) {
		for _, entry := range folder.Entries {
			if entry.Name() == path[len(path)-1] {
				entryFile, ok := entry.(*DirFile)
				if !ok {
					// TODO: This happen e. g. with submodules.
					return
				}
				prevContents := entryFile.contents
				entryFile.contents = func() (string, error) {
					prev, err := prevContents()
					if err != nil {
						return "", err
					}
					return applyPatch(diff, prev)
				}
				break
			}
		}
	})
}

func changeFileInTree(path []string, tree DirTree, changeCallback func(*DirFolder)) {
	if len(path) == 0 {
		return
	}
	name := path[0]
	isDir := len(path) > 1

	switch t := tree.(type) {
	case *DirFolder:
		if isDir {
			var entry DirTree
			for _, oldEntry := range t.Entries {
				if oldEntry.Name() == name {
					entry = oldEntry
					break
				}
			}
			if entry == nil {
				entry = &DirFolder{
					name:    name,
					Entries: []DirTree{},
				}
				t.Entries = append(t.Entries, entry)
			}
			changeFileInTree(path[1:], entry, changeCallback)
		} else {
			changeCallback(t)
		}
	case *DirFile:
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
