package cmd

import (
	"fmt"

	"github.com/fntlnz/kubectl-trace/factory"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	// cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	// "k8s.io/kubernetes/pkg/kubectl/util/i18n"
	// "k8s.io/kubectl/pkg/pluginutils"
)

// Possible resources include (case insensitive):
//   pod (po), node

var (
	traceLong = `Configure, execute, and manage bpftrace programs.

These commands help you trace existing application resources.
	`
	traceExamples = `
  # Execute a bpftrace program from file on a specific node
  %[1]s trace run kubernetes-node-emt8.c.myproject.internal -p read.bt

  # Get all bpftrace programs in all namespaces
  %[1]s trace get --all-namespaces

  # Delete all bpftrace programs in a specific namespace
  %[1]s trace delete -n myns
`
)

// TraceOptions ...
type TraceOptions struct {
	configFlags *genericclioptions.ConfigFlags

	genericclioptions.IOStreams
}

// NewTraceOptions provides an instance of TraceOptions with default values.
func NewTraceOptions(streams genericclioptions.IOStreams) *TraceOptions {
	return &TraceOptions{
		configFlags: genericclioptions.NewConfigFlags(),

		IOStreams: streams,
	}
}

// NewTraceCommand creates the trace command and its nested children.
func NewTraceCommand(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewTraceOptions(streams)

	cmd := &cobra.Command{
		Use:                   "trace",
		DisableFlagsInUseLine: true,
		Short:                 `Execute and manage bpftrace programs`, // Wrap with i18n.T()
		Long:                  traceLong,                              // Wrap with templates.LongDesc()
		Example:               fmt.Sprintf(traceExamples, "kubectl"),  // Wrap with templates.Examples()
		Run: func(c *cobra.Command, args []string) {
			c.SetOutput(streams.ErrOut)
			cobra.NoArgs(c, args)
			c.Help()
		},
	}

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	matchVersionFlags := factory.NewMatchVersionFlags(o.configFlags)
	matchVersionFlags.AddFlags(flags)

	// flags.AddGoFlagSet(flag.CommandLine) // todo(leodido) > evaluate whether we need this or not

	f := factory.NewFactory(matchVersionFlags)

	cmd.AddCommand(NewRunCommand(f, streams))
	cmd.AddCommand(NewGetCommand(f, streams))
	cmd.AddCommand(NewAttachCommand(streams))
	cmd.AddCommand(NewDeleteCommand(streams))

	return cmd
}
