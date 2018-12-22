package main

import (
	"os"

	"github.com/fntlnz/kubectl-trace/pkg/cmd"
	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("trace-runner", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewTraceRunnerCommand()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
