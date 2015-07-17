package repositories

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/tcard/navpatch/navpatch"
)

type FSRepository struct {
	baseDir string
}

func NewFSRepository(baseDir string) *FSRepository {
	return &FSRepository{baseDir}
}

func (r *FSRepository) Tree() (navpatch.TreeEntry, error) {
	return dirPathToTree(r.baseDir)
}

func dirPathToTree(path string) (navpatch.TreeEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening directory: %s", err)
	}
	return dirFileToTree(f, path)
}

func dirFileToTree(f *os.File, path string) (navpatch.TreeEntry, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading stats for %s: %s", path, err)
	}

	if fi.IsDir() {
		names, err := f.Readdirnames(0)
		if err != nil {
			return nil, fmt.Errorf("reading files at %s: %s", path, err)
		}

		ret := navpatch.NewTreeFolder(fi.Name())
		for _, name := range names {
			entryPath := path + "/" + name
			f, err := os.Open(entryPath)
			if err != nil {
				// Ignore errors here; best effort.
				continue
			}
			defer f.Close()
			entry, err := dirFileToTree(f, entryPath)
			if err != nil {
				return nil, err
			}
			ret.Entries = append(ret.Entries, entry)
		}

		sort.Sort(byName(ret.Entries))

		return ret, nil
	} else {
		return navpatch.NewTreeFile(fi.Name(), func() (string, error) {
			bs, err := ioutil.ReadFile(path)
			return string(bs), err
		}), nil
	}
}

type byName []navpatch.TreeEntry

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return n[i].Name() < n[j].Name() }
