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
	folderPath := filepath.Join(gc.cloneDir, repoDir)

	_, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("opening repo folder: %v", err)
	}

	err = updateRepo(folderPath)
	if err != nil {
		return "", err
	}

	tmpPath, err := ioutil.TempDir("", repoDir)
	if err != nil {
		return "", fmt.Errorf("creating temp folder: %v", err)
	}

	feedback("Copying repo directory...")
	out, err := exec.Command("cp", "-r", folderPath, tmpPath+"/").CombinedOutput()
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
	cmd = exec.Command("git", "checkout", oldCommit)
	cmd.Dir = repoPath
	_, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("checking out %s: %v; git output: %v", oldCommit, err, string(rawPatch))
	}

	feedback("Generating patch visualization...")
	return navpatch.NewNavigator(repoPath, rawPatch)
}

func md5hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

var repoLocks = struct {
	sync.RWMutex
	m map[string]*sync.Mutex
}{m: map[string]*sync.Mutex{}}

func updateRepo(path string) error {
	repoLocks.RLock()
	lock := repoLocks.m[path]
	repoLocks.RUnlock()

	if lock == nil {
		lock = &sync.Mutex{}
		repoLocks.Lock()
		repoLocks.m[path] = lock
		repoLocks.Unlock()
	}

	lock.Lock()
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
