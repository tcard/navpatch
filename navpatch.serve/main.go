package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/tcard/navpatch/internal"
	"github.com/tcard/navpatch/navpatch.serve/navpatchserve"
)

var listenAddr = flag.String("http", ":6177", "HTTP address to listen on.")
var cloneDir = flag.String("cloneDir", ".", "Clone GitHub repos at this directory.")
var sessionsLimit = flag.Int("sessionsLimit", -1, "If > 0, number of concurrent sessions allowed.")
var whitelistFlag = flag.String("whitelist", "", "A '|'-separated list of regexps. If not empty, only git repos matching any of them will be allowed.")
var timePerSession = flag.Duration("timePerSession", 10*time.Minute, "Time before a session is ended (ie. its cached data is removed and everything is slow).")

func main() {
	flag.Parse()

	whitelistStr := strings.Split(*whitelistFlag, "|")
	whitelist := make([]*regexp.Regexp, 0, len(whitelistStr))
	for _, s := range whitelistStr {
		rgx, err := regexp.Compile(s)
		if err != nil {
			internal.ErrorExit("compiling whitelist regexps:", err)
		}
		whitelist = append(whitelist, rgx)
	}

	h := navpatchserve.NewHandler(*cloneDir, "git_command_unix", *sessionsLimit, whitelist, *timePerSession)

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		internal.ErrorExit("starting server:", err)
	}

	fmt.Println("Serving at " + listener.Addr().String())
	log.Fatal(http.Serve(listener, h))
}
