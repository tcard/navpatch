package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/tcard/navpatch/internal"
	"github.com/tcard/navpatch/navpatch"
	"github.com/tcard/navpatch/navpatch/repositories"
)

func main() {
	listenAddr, baseDir, rawPatch := processArgs()

	r := repositories.NewFSRepository(baseDir)
	nav, err := navpatch.NewNavigator(r, rawPatch)
	if err != nil {
		internal.ErrorExit(err)
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		internal.ErrorExit("starting server:", err)
	}

	fmt.Println("Serving at " + listener.Addr().String())
	log.Fatal(http.Serve(listener, nav))
}

func processArgs() (string, string, []byte) {
	args := os.Args
	if len(args) < 2 || len(args) > 4 {
		badArgs()
	}

	if args[1] == "-h" {
		usage()
		os.Exit(0)
	} else if len(args) < 3 {
		badArgs()
	}

	var rawPatch []byte
	var err error
	if len(args) == 3 {
		rawPatch, err = ioutil.ReadAll(os.Stdin)
	} else {
		rawPatch, err = ioutil.ReadFile(args[3])
		if err != nil {
			resp, getErr := http.Get(args[3])
			if getErr == nil {
				rawPatch, err = ioutil.ReadAll(resp.Body)
			}
		}
	}
	if err != nil {
		internal.ErrorExit(err)
	}

	return args[1], args[2], rawPatch
}

func badArgs() {
	fmt.Fprintln(os.Stderr, "missing, exceeding or malformed arguments.\n")
	usage()
	os.Exit(1)
}

func usage() {
	fmt.Println(`usage: navpatch [-h] <listenAddr> <baseDir> [<patchFile>]

Visualize a patch file through a file navigator

Patch files should be formatted as understood by golang.org/x/codereview/patch.

Options:
  -h         : show this help message.
  listenAddr : the HTTP address in which to serve the web interface.
               ':0' serves at an arbitrary port.
  baseDir    : path to the directory to which the patch is applied.
  patchFile  : path or URL to the patch file to be applied.
               If ommitted, reads from stdin.`)
}
