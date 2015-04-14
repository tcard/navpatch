package navpatchserve

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tcard/navpatch/navpatch"
)

type gitCommandUnix struct {
	cloneDir string
}

var cloneURLPrefixes = []string{
	"git://",
	"git+ssh://",
	"git+ssh://git@",
	"https://",
}

func (gc gitCommandUnix) clone(repoURL string) error {
	repoDir := md5hash(repoURL)
	cloneDir := filepath.Join(gc.cloneDir, repoDir)

	var err error
	var out []byte

	for _, pfx := range cloneURLPrefixes {
		url := pfx + repoURL
		cmd := exec.Command("git", "clone", url, cloneDir)
		out, err = cmd.CombinedOutput()
		if err == nil {
			break
		}
	}

	if strings.HasPrefix(repoURL, "github.com") {
		// Add PRs.
		bs, err := ioutil.ReadFile(filepath.Join(cloneDir, ".git/config"))
		if err != nil {
			return fmt.Errorf("opening .git/config: %v", err)
		}
		bs = bytes.Replace(bs, []byte(`[remote "origin"]`), []byte(`[remote "origin"]
	fetch = +refs/pull/*/head:refs/remotes/origin/pr/*`), -1)
		err = ioutil.WriteFile(filepath.Join(cloneDir, ".git/config"), bs, 0655)
		if err != nil {
			return fmt.Errorf("writing .git/config: %v", err)
		}
	}

	if err != nil {
		return fmt.Errorf("cloning: %v; git output: %v", err, string(out))
	}

	return nil
}

func (gc gitCommandUnix) copyRepo(repoURL string, feedback func(string)) (string, error) {
	repoDir := md5hash(repoURL)
	repoPath, err := gc.repoPath(repoURL)
	if err != nil {
		return "", err
	}

	err = updateRepo(repoPath)
	if err != nil {
		return "", err
	}

	tmpPath, err := ioutil.TempDir("", repoDir)
	if err != nil {
		return "", fmt.Errorf("creating temp folder: %v", err)
	}

	feedback("Copying repo directory...")
	out, err := exec.Command("cp", "-r", repoPath, tmpPath+"/").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("copying to temp folder: %v; cp output: %v", err, string(out))
	}

	return tmpPath + "/" + repoDir, nil
}

func (gc gitCommandUnix) patchNavigator(repoURL, oldCommit, newCommit string, feedback func(string)) (*navpatch.Navigator, error) {
	repoPath, err := gc.copyRepo(repoURL, feedback)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "diff", "--no-color", oldCommit, newCommit)
	cmd.Dir = repoPath
	rawPatch, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("diffing: %v; git output: %v", err, string(rawPatch))
	}

	feedback("Checking out base commit...")
	cmd = exec.Command("git", "stash")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "checkout", oldCommit)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("checking out %s: %v; git output: %v", oldCommit, err, string(out))
	}

	cmd = exec.Command("git", "stash", "pop")
	cmd.Dir = repoPath
	cmd.Run()

	feedback("Generating patch visualization...")
	return navpatch.NewNavigator(repoPath, rawPatch)
}

func (gc gitCommandUnix) commitsForPR(repoURL string, pr string) (oldCommit string, newCommit string, err error) {
	repoPath, err := gc.repoPath(repoURL)
	if err != nil {
		return "", "", err
	}

	err = updateRepo(repoPath)
	if err != nil {
		return "", "", err
	}

	lock := repoLocks.Lock(repoPath)
	defer lock.Unlock()

	cmd := exec.Command("git", "rev-parse", "--short", "origin/pr/"+pr)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git rev-parse --short %s: %v; git output: %v", "origin/pr/"+pr, err, string(out))
	}

	newCommit = string(out[:len(out)-1])

	cmd = exec.Command("git", "rev-parse", "--short", "--abrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err = cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git rev-parse --short --abrev-ref HEAD: %v; git output: %v", err, string(out))
	}

	baseBranch := string(out[:len(out)-1])

	cmd = exec.Command("git", "merge-base", "origin/pr/"+pr, baseBranch)
	cmd.Dir = repoPath
	out, err = cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git merge-base %v %v: %v; git output: %v", "origin/pr/"+pr, baseBranch, err, string(out))
	}

	oldCommit = string(out[:len(out)-1])

	return oldCommit, newCommit, nil
}

func (gc gitCommandUnix) repoPath(repoURL string) (string, error) {
	repoDir := md5hash(repoURL)
	repoPath := filepath.Join(gc.cloneDir, repoDir)

	_, err := os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("opening repo folder: %v", err)
	}

	return repoPath, nil
}

func md5hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

type repoLocksT struct {
	sync.RWMutex
	m map[string]*sync.Mutex
}

func (ls repoLocksT) Lock(path string) *sync.Mutex {
	ls.RLock()
	lock := ls.m[path]
	ls.RUnlock()

	if lock == nil {
		lock = &sync.Mutex{}
		ls.RWMutex.Lock()
		ls.m[path] = lock
		ls.RWMutex.Unlock()
	}

	lock.Lock()
	return lock
}

var repoLocks = repoLocksT{m: map[string]*sync.Mutex{}}

func updateRepo(path string) error {
	lock := repoLocks.Lock(path)
	defer lock.Unlock()

	commands := [][]string{
		[]string{"git", "stash"},
		[]string{"git", "fetch", "--all"},
		[]string{"git", "rebase", "origin/master"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %v; output: %v", strings.Join(cmdArgs, " "), err, string(out))
		}
	}

	cmd := exec.Command("git", "stash", "pop")
	cmd.Dir = path
	cmd.Run()

	out, err := exec.Command("find", path, "-type", "l", "-exec", "test", "!", "-e", "{}", ";", "-delete").CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing broken symlinks: %v; output: %v", err, string(out))
	}

	return nil
}
