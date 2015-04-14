package navpatchserve

import "github.com/tcard/navpatch/navpatch"

type gitLib interface {
	clone(repoURL string) error
	patchNavigator(repoURL, oldCommit, newCommit string, feedback func(string)) (*navpatch.Navigator, error)
}

type githubLib interface {
	commitsForPR(repoURL string, pr string) (oldCommit string, newCommit string, err error)
}
