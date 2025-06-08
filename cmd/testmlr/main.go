// Main
package main

import (
	"os"

	"github.com/johnkerl/miller/v6/pkg/entrypoint"
)

func main() {
	os.Args = []string{"", "help", "topics"}
	// reader := bytes.NewBufferString("hello")
	// os.Stdin = reader
	entrypoint.Main()
}
