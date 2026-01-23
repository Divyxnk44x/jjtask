package main

import (
	"os"

	"jjtask/cmd/jjtask/cmd"
	"jjtask/internal/jj"
)

func main() {
	jj.SetupEnv()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
