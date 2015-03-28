package navpatchserve

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tcard/errorer"
	"github.com/tcard/navpatch/navpatch"
)

type Handler struct {
	listenAddr, cloneDir string
	gitLib               gitCommandUnix
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.handleRoot(w, req)
}

func NewHandler(listenAddr, cloneDir string, gitLib string) *Handler {
	return &Handler{listenAddr, cloneDir, gitCommandUnix{cloneDir}}
}

func (h *Handler) handleRoot(w http.ResponseWriter, req *http.Request) {
	// TODO: GitHub rewriting.
	isNil := errorer.Numbered(func(err error, n int) {
		log.Println("ERROR", "rootHandler", n, err)
		w.WriteHeader(http.StatusInternalServerError)
	})

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
