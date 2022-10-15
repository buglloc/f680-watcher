package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/buglloc/f680-watcher/internal/commands"
)

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

func main() {
	runtime.GOMAXPROCS(1)
	rand.Seed(time.Now().UnixNano())

	if err := commands.Execute(); err != nil {
		fatal(err)
	}
}
