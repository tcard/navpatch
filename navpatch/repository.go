package navpatch

import "strings"

type Repository interface {
	Tree() (TreeEntry, error)
}

type TreeEntry interface {
	Name() string
	isTreeEntry()
}

type TreeFolder struct {
	name    string
	Entries []TreeEntry
}

func NewTreeFolder(name string) *TreeFolder {
	return &TreeFolder{
		name: name,
	}
}

func (f *TreeFolder) isTreeEntry() {}

func (f *TreeFolder) String() string {
	return DirTreeString(f)
}

func (f *TreeFolder) Name() string {
	return f.name
}

type TreeFile struct {
	name           string
	contents       ContentRetriever
	cached         bool
	cachedContents string
}

type ContentRetriever func() (string, error)

func NewTreeFile(name string, f ContentRetriever) *TreeFile {
	return &TreeFile{name: name, contents: f}
}

func (f *TreeFile) isTreeEntry() {}

func (f *TreeFile) String() string {
	return DirTreeString(f)
}

func (f *TreeFile) Name() string {
	return f.name
}

func (f *TreeFile) Contents() (string, error) {
	if f.cached {
		return f.cachedContents, nil
	}

	ret, err := f.contents()
	if err == nil {
		f.cached = true
		f.cachedContents = ret
	}

	return ret, err
}

func DirTreeString(entry TreeEntry) string {
	return dirTreeString2(entry, 0)
}

func dirTreeString2(entry TreeEntry, level int) string {
	ret := strings.Repeat("-- ", level) + entry.Name() + "\n"

	switch v := entry.(type) {
	case *TreeFolder:
		for _, e := range v.Entries {
			ret += dirTreeString2(e, level+1)
		}
	}

	return ret
}
