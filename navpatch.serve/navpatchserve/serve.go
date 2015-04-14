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

type requestContext struct {
	writingHTML bool
	h           *Handler
	w           http.ResponseWriter
	req         *http.Request
	isNil       errorer.NumberedErrorer
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctxt := &requestContext{h: h, w: w, req: req}
	ctxt.isNil = errorer.Numbered(func(err error, n int) {
		log.Println("ERROR", "rootHandler", n, err)
		if ctxt.writingHTML {
			ctxt.writeHTML("<pre>" + err.Error() + "</pre>")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(ctxt.w, err, "\n")
		}
	})
	ctxt.handleRoot()
}

func NewHandler(cloneDir string, gitLib string) *Handler {
	return &Handler{cloneDir, gitCommandUnix{cloneDir}}
}

func (ctxt *requestContext) handleRoot() {
	if ctxt.maybeRedirect() {
		return
	}

	urlValues := ctxt.req.URL.Query()
	oldArg := urlValues.Get("old")
	newArg := urlValues.Get("new")
	if oldArg == "" || newArg == "" {
		ctxt.w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(ctxt.w, "You must provide 'old' and 'new' commits as query parameters, like: ?old=286c420&new=a29b7cd")
		return
	}
	pathArg := ctxt.req.URL.Query().Get("path")
	if pathArg == "" {
		pathArg = "/"
	}

	gitURL := path2git(ctxt.req.URL.Path)

	nav := getCachedNav(gitURL, oldArg, newArg)
	if nav == nil {
		var err error
		nav, err = ctxt.h.gitLib.patchNavigator(gitURL, oldArg, newArg, func(feedback string) {
			ctxt.writeHTML("<p>" + feedback + "</p>")
		})

		if os.IsNotExist(err) {
			ctxt.cloneAndReload(gitURL)
			return
		} else if !ctxt.isNil(err) {
			return
		} else {
			setCachedNav(nav, gitURL, oldArg, newArg)
		}

		ctxt.htmlReload()

		return
	}

	linksURL := ctxt.req.URL
	q := linksURL.Query()
	q.Del("path")
	linksURL.RawQuery = q.Encode() + "&path="
	nav.HandleRoot(ctxt.w, ctxt.req, pathArg, linksURL.String())
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

func (ctxt *requestContext) maybeRedirect() bool {
	return ctxt.maybeRedirectGithubPR()
}

var githubPRRegexp = regexp.MustCompile(`(github.com/[^/]+/[^/]+)/pull/([0-9]+)`)

func (ctxt *requestContext) maybeRedirectGithubPR() bool {
	githubLib, ok := ctxt.h.gitLib.(githubLib)
	if !ok {
		return false
	}

	gitURL := path2git(ctxt.req.URL.Path)
	m := githubPRRegexp.FindStringSubmatch(gitURL)
	if len(m) < 3 {
		return false
	}

	oldCommit, newCommit, err := githubLib.commitsForPR(m[1], m[2])

	if os.IsNotExist(err) {
		ctxt.cloneAndReload(m[1])
		return true
	} else if !ctxt.isNil(err) {
		return true
	}

	newURL := ctxt.req.URL
	newURL.Path = m[1]
	newQuery := newURL.Query()
	newQuery.Del("old")
	newQuery.Add("old", oldCommit)
	newQuery.Del("new")
	newQuery.Add("new", newCommit)
	newURL.RawQuery = newQuery.Encode()

	ctxt.w.Header().Set("Location", "/"+newURL.String())
	ctxt.w.WriteHeader(http.StatusTemporaryRedirect)

	return true
}

func (ctxt *requestContext) writeHTML(s string) {
	if !ctxt.writingHTML {
		ctxt.writingHTML = true
		ctxt.w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(ctxt.w, "<html><body>")
	}

	fmt.Fprintln(ctxt.w, s)
	if f, ok := ctxt.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (ctxt *requestContext) htmlReload() {
	ctxt.writeHTML("<script>location.reload();</script></body></html>")
}

func (ctxt *requestContext) cloneAndReload(gitURL string) {
	ctxt.writeHTML("<p>Cloning...</p>")

	if err := ctxt.h.gitLib.clone(gitURL); !ctxt.isNil(err) {
		ctxt.writeHTML("<p>Couldn't clone that repo. Sorry :(</p></body></html>")
	}

	ctxt.htmlReload()
}
