package navpatch

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

type DirTree interface {
	Name() string
	isDirTree()
}

type DirFolder struct {
	name    string
	Entries []DirTree
}

func (d *DirFolder) isDirTree() {}

func (d *DirFolder) String() string {
	return DirTreeString(d)
}

func (d *DirFolder) Name() string {
	return d.name
}

type DirFile struct {
	name           string
	contents       func() (string, error)
	cached         bool
	cachedContents string
}

func (d *DirFile) isDirTree() {}

func (d *DirFile) String() string {
	return DirTreeString(d)
}

func (d *DirFile) Name() string {
	return d.name
}

func (d *DirFile) Contents() (string, error) {
	if d.cached {
		return d.cachedContents, nil
	}

	ret, err := d.contents()
	if err == nil {
		d.cached = true
		d.cachedContents = ret
	}

	return ret, err
}

func DirPathToTree(path string) (DirTree, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening directory: %s", err)
	}
	return DirFileToTree(f, path)
}

func DirFileToTree(f *os.File, path string) (DirTree, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading stats for %s: %s", path, err)
	}

	if fi.IsDir() {
		ret := &DirFolder{
			name:    fi.Name(),
			Entries: []DirTree{},
		}

		names, err := f.Readdirnames(0)
		if err != nil {
			return nil, fmt.Errorf("reading files at %s: %s", path, err)
		}

		for _, name := range names {
			entryPath := path + "/" + name
			f, err := os.Open(entryPath)
			if err != nil {
				// Ignore errors here; best effort.
				continue
			}
			defer f.Close()
			entry, err := DirFileToTree(f, entryPath)
			if err != nil {
				return nil, err
			}
			ret.Entries = append(ret.Entries, entry)
		}

		sort.Sort(byName(entries))

		return ret, nil
	} else {
		var df *DirFile
		df = &DirFile{
			name: fi.Name(),
			contents: func() (string, error) {
				bs, err := ioutil.ReadFile(path)
				return string(bs), err
			},
		}
		return df, nil
	}
}

func DirTreeString(d DirTree) string {
	return dirTreeString2(d, 0)
}

func dirTreeString2(d DirTree, level int) string {
	ret := strings.Repeat("-- ", level) + d.Name() + "\n"

	switch v := d.(type) {
	case *DirFolder:
		for _, e := range v.Entries {
			ret += dirTreeString2(e, level+1)
		}
	}

	return ret
}

type byName []DirTree

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return n[i].Name() < n[j].Name() }
