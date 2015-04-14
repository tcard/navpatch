package navpatchserve

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/tcard/errorer"
	"github.com/tcard/navpatch/navpatch"
)

type Handler struct {
	cloneDir string
	gitLib   gitLib
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.handleRoot(w, req)
}

func NewHandler(cloneDir string, gitLib string) *Handler {
	return &Handler{cloneDir, gitCommandUnix{cloneDir}}
}

func (h *Handler) handleRoot(w http.ResponseWriter, req *http.Request) {
	isNil := errorer.Numbered(func(err error, n int) {
		log.Println("ERROR", "rootHandler", n, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err, "\n")
	})

	h.rewriteReq(req)

	urlValues := req.URL.Query()
	oldArg := urlValues.Get("old")
	newArg := urlValues.Get("new")
	if oldArg == "" || newArg == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "You must provide 'old' and 'new' commits as query parameters, like: ?old=286c420&new=a29b7cd")
		return
	}
	pathArg := req.URL.Query().Get("path")
	if pathArg == "" {
		pathArg = "/"
	}

	gitURL := path2git(req.URL.Path)

	nav := getCachedNav(gitURL, oldArg, newArg)
	if nav == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, "<html><body>")

		var err error
		nav, err = h.gitLib.patchNavigator(gitURL, oldArg, newArg, func(feedback string) {
			fmt.Fprintln(w, "<p>"+feedback+"</p>")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		})

		if os.IsNotExist(err) {
			fmt.Fprintln(w, "<p>Cloning...</p>")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			if err := h.gitLib.clone(gitURL); !isNil(err) {
				fmt.Fprintln(w, "<p>Couldn't clone that repo. Sorry :(</p></body></html>")
				return
			}
		} else if !isNil(err) {
			return
		} else {
			setCachedNav(nav, gitURL, oldArg, newArg)
		}

		fmt.Fprintln(w, "<script>location.reload();</script></body></html>")
		return
	}

	linksURL := req.URL
	q := linksURL.Query()
	q.Del("path")
	linksURL.RawQuery = q.Encode() + "&path="
	nav.HandleRoot(w, req, pathArg, linksURL.String())
}

func path2git(path string) string {
	return path[1:]
}

var cachedNavs = struct {
	sync.RWMutex
	m map[navArgs]*cachedNavsEntry
}{m: map[navArgs]*cachedNavsEntry{}}

type navArgs struct {
	gitURL, oldArg, newArg string
}

var cachedNavTTL = 30 * time.Minute

type cachedNavsEntry struct {
	sync.RWMutex
	nav      *navpatch.Navigator
	lastUsed time.Time
}

func getCachedNav(gitURL, oldArg, newArg string) *navpatch.Navigator {
	cachedNavs.RLock()
	defer cachedNavs.RUnlock()

	ret := cachedNavs.m[navArgs{gitURL, oldArg, newArg}]
	if ret == nil {
		return nil
	}

	ret.Lock()
	ret.lastUsed = time.Now()
	ret.Unlock()

	return ret.nav
}

func setCachedNav(nav *navpatch.Navigator, gitURL, oldArg, newArg string) {
	cachedNavs.Lock()
	defer cachedNavs.Unlock()

	cachedNavs.m[navArgs{gitURL, oldArg, newArg}] = &cachedNavsEntry{
		nav:      nav,
		lastUsed: time.Now(),
	}

	var evict func(time.Duration)
	evict = func(sleepTime time.Duration) {
		time.Sleep(sleepTime)

		cachedNavs.RLock()
		e := cachedNavs.m[navArgs{gitURL, oldArg, newArg}]
		cachedNavs.RUnlock()

		e.RLock()
		now := time.Now()
		diff := now.Sub(e.lastUsed)
		e.RUnlock()

		if diff < cachedNavTTL {
			go evict(cachedNavTTL - diff)
		} else {
			cachedNavs.Lock()
			defer cachedNavs.Unlock()
			delete(cachedNavs.m, navArgs{gitURL, oldArg, newArg})
		}
	}
	go evict(cachedNavTTL)
}

func (h *Handler) rewriteReq(req *http.Request) error {
	return h.rewriteGithubPR(req)
}

var githubPRRegexp = regexp.MustCompile(`(github.com/[^/]+/[^/]+)/pull/([0-9]+)`)

func (h *Handler) rewriteGithubPR(req *http.Request) error {
	githubLib, ok := h.gitLib.(githubLib)
	if !ok {
		return nil
	}

	gitURL := path2git(req.URL.Path)
	m := githubPRRegexp.FindStringSubmatch(gitURL)
	if len(m) < 3 {
		return nil
	}

	oldCommit, newCommit, err := githubLib.commitsForPR(m[1], m[2])

	if err != nil {
		oldCommit = "FIXME"
		newCommit = "FIXME"
	}

	req.URL.Path = "/" + m[1]
	newQuery := req.URL.Query()
	newQuery.Del("old")
	newQuery.Add("old", oldCommit)
	newQuery.Del("new")
	newQuery.Add("new", newCommit)
	req.URL.RawQuery = newQuery.Encode()

	return nil
}
