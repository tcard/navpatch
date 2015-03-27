package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/codereview/patch"
)

func main() {
	listenAddr, baseDir, basePath, patchSet := processArgs()
	_, _ = baseDir, patchSet

	handlers, err := newHandlers(baseDir, basePath, patchSet)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", handlers.root)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening at " + listener.Addr().String())
	log.Fatal(http.Serve(listener, nil))
}

func processArgs() (string, *os.File, string, *patch.Set) {
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

	baseDir, err := os.Open(args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var rawPatch []byte
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	set, err := patch.Parse(rawPatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return args[1], baseDir, args[2], set
}

func badArgs() {
	usage()
	os.Exit(1)
}

func usage() {
	fmt.Println(`usage: navpatch [-h] listenAddr baseDir [patchFile]

If patchFile is not specified, it will read a patch file from the standard input.`)
}
