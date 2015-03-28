package internal

import (
	"fmt"
	"os"
)

func ErrorExit(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}
