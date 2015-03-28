package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/tcard/navpatch/internal"
	"github.com/tcard/navpatch/navpatch.serve/navpatchserve"
)

var listenAddr = flag.String("http", ":6177", "HTTP address to listen on.")
var cloneDir = flag.String("cloneDir", ".", "Clone GitHub repos at this directory.")

func main() {
	flag.Parse()
	h := navpatchserve.NewHandler(*listenAddr, *cloneDir, "git_command_unix")

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		internal.ErrorExit("starting server:", err)
	}

	fmt.Println("Serving at " + listener.Addr().String())
	log.Fatal(http.Serve(listener, h))
}
