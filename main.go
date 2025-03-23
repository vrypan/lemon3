package main

import (
	"os"

	"github.com/vrypan/lemon3/cmd"
)

func main() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "tui")
	}
	cmd.Execute()
}
