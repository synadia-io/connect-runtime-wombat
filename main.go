package main

import (
	"context"
	"fmt"
	"github.com/synadia-io/connect-runtime-wombat/runner"
	"github.com/synadia-io/connect/runtime"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("usage: wombat <config>")
		os.Exit(1)
	}

	rt, err := runtime.FromEnv()
	if err != nil {
		panic(err)
	}

	// launching a workload will start the workload and block
	if err := rt.Launch(context.Background(), runner.Run, args[0]); err != nil {
		panic(err)
	}
}
