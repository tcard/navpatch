package repositories

import (
	"testing"

	"github.com/tcard/navpatch/navpatch"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type GithubS struct{}

var _ = Suite(&GithubS{})

func (s *GithubS) TestGithubTree(c *C) {
	r, err := NewGithubRepository("https://github.com/tyba/git-fixture#6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
	c.Assert(err, IsNil)

	t, err := r.Tree()
	c.Assert(err, IsNil)

	c.Assert(t.(*navpatch.TreeFolder).String(), Equals, `.
-- .gitignore
-- CHANGELOG
-- LICENSE
-- binary.jpg
-- go
-- -- example.go
-- json
-- -- long.json
-- -- short.json
-- php
-- -- crappy.php
-- vendor
-- -- foo.go
`)
}

func (s *GithubS) TestGithubContent(c *C) {
	r, err := NewGithubRepository("https://github.com/tyba/git-fixture#6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
	c.Assert(err, IsNil)

	t, err := r.Tree()
	c.Assert(err, IsNil)

	file := t.(*navpatch.TreeFolder).Entries[0].(*navpatch.TreeFile)
	c.Assert(file.Name(), Equals, ".gitignore")

	content, err := file.Contents()
	c.Assert(err, IsNil)
	c.Assert(len(content), Equals, 189)
}
