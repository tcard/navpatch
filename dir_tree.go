package main

import (
	"io/ioutil"
	"os"
	"strings"
)

type dirTree interface {
	Name() string
	isDirTree()
}

type dirFolder struct {
	name    string
	Entries []dirTree
}

func (d *dirFolder) isDirTree() {}

func (d *dirFolder) String() string {
	return dirTreeString(d)
}

func (d *dirFolder) Name() string {
	return d.name
}

type dirFile struct {
	name     string
	Contents func() (string, error)
}

func (d *dirFile) isDirTree() {}

func (d *dirFile) String() string {
	return dirTreeString(d)
}

func (d *dirFile) Name() string {
	return d.name
}

func dirToTree(f *os.File, path string) (dirTree, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		ret := &dirFolder{
			name:    fi.Name(),
			Entries: []dirTree{},
		}

		names, err := f.Readdirnames(0)
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			entryPath := path + "/" + name
			f, err := os.Open(entryPath)
			if err != nil {
				// Ignore errors here; best effort.
				continue
			}
			defer f.Close()
			entry, err := dirToTree(f, entryPath)
			if err != nil {
				return nil, err
			}
			ret.Entries = append(ret.Entries, entry)
		}

		return ret, nil
	} else {
		var df *dirFile
		df = &dirFile{
			name: fi.Name(),
			Contents: func() (string, error) {
				bs, err := ioutil.ReadFile(path)
				s := string(bs)
				df.Contents = func() (string, error) {
					return s, err
				}
				return s, err
			},
		}
		return df, nil
	}
}

func dirTreeString(d dirTree) string {
	return dirTreeString2(d, 0)
}

func dirTreeString2(d dirTree, level int) string {
	ret := strings.Repeat("-- ", level) + d.Name() + "\n"

	switch v := d.(type) {
	case *dirFolder:
		for _, e := range v.Entries {
			ret += dirTreeString2(e, level+1)
		}
	}

	return ret
}
