package main

import (
	"os"

	"github.com/iovisor/kubectl-trace/pkg/cmd"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// Initialize all k8s client auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-trace", pflag.ExitOnError)
	pflag.CommandLine = flags

	streams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	root := cmd.NewTraceCommand(streams)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
