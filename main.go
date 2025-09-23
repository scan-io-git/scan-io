package main

import (
	"os"

	"github.com/scan-io-git/scan-io/cmd"
)

func main() {
	code := cmd.Execute()
	os.Exit(code)
}
