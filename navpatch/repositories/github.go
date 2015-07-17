package repositories

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
	"github.com/tcard/navpatch/navpatch"
	"gopkg.in/sourcegraph/go-vcsurl.v1"
)

type GithubRepository struct {
	vcs     *vcsurl.RepoInfo
	client  *github.GitService
	folders map[string]*navpatch.TreeFolder
}

func NewGithubRepository(url string) (*GithubRepository, error) {
	vcs, err := vcsurl.Parse(url)
	if err != nil {
		return nil, err
	}

	return &GithubRepository{
		vcs:     vcs,
		client:  github.NewClient(nil).Git,
		folders: make(map[string]*navpatch.TreeFolder, 0),
	}, nil
}

func (r *GithubRepository) GetTree() (navpatch.TreeEntry, error) {
	t, _, err := r.client.GetTree(r.vcs.Username, r.vcs.Name, r.vcs.Rev, true)
	if err != nil {
		return nil, err
	}

	return r.transformTree(t), nil
}

func (r *GithubRepository) transformTree(o *github.Tree) navpatch.TreeEntry {
	e := navpatch.NewTreeFolder(".")
	for _, entry := range o.Entries {
		t := r.transformTreeEntry(&entry)
		if t == nil {
			continue
		}

		e.Entries = append(e.Entries, t)
	}

	return e
}

func (r *GithubRepository) transformTreeEntry(o *github.TreeEntry) navpatch.TreeEntry {
	parent := filepath.Dir(*o.Path)
	base := filepath.Base(*o.Path)

	var entry navpatch.TreeEntry
	switch *o.Type {
	case "blob":
		sha := *o.SHA
		entry = navpatch.NewTreeFile(base, func() (string, error) {
			b, _, err := r.client.GetBlob(r.vcs.Username, r.vcs.Name, sha)
			if err != nil {
				return "", err
			}

			data, err := base64.StdEncoding.DecodeString(*b.Content)
			if err != nil {
				return "", err
			}

			return string(data), nil
		})
	case "tree":
		entry = navpatch.NewTreeFolder(base)
		r.folders[*o.Path] = entry.(*navpatch.TreeFolder)
	default:
		fmt.Println(*o.Type)

	}

	if inRootPath(*o.Path) {
		return entry
	} else {
		f := r.folders[parent]
		f.Entries = append(f.Entries, entry)
	}

	return nil
}

func inRootPath(path string) bool {
	return strings.Count(path, "/") == 0
}
