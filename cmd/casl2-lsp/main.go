package main

import (
	"os"
	
	"github.com/f0reachARR/casljs/lsp"
)

func main() {
	server := lsp.NewServer()
	if err := server.Start(os.Stdin, os.Stdout); err != nil {
		os.Exit(1)
	}
}
