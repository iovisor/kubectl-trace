package cmd

import (
	"fmt"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	traceLong = `Configure, execute, and manage bpftrace programs.

These commands help you trace existing application resources.
	`
	traceExamples = `
  # Execute a bpftrace program from file on a specific node
  %[1]s trace run kubernetes-node-emt8.c.myproject.internal -f read.bt

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
		configFlags: genericclioptions.NewConfigFlags(false),

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
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.SetOutput(streams.ErrOut)
		},
		Run: func(c *cobra.Command, args []string) {
			cobra.NoArgs(c, args)
			c.Help()
		},
	}

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	matchVersionFlags := cmdutil.NewMatchVersionFlags(o.configFlags)
	matchVersionFlags.AddFlags(flags)

	f := cmdutil.NewFactory(matchVersionFlags)

	cmd.AddCommand(NewRunCommand(f, streams))
	cmd.AddCommand(NewGetCommand(f, streams))
	cmd.AddCommand(NewAttachCommand(f, streams))
	cmd.AddCommand(NewDeleteCommand(f, streams))
	cmd.AddCommand(NewVersionCommand(streams))
	cmd.AddCommand(NewLogCommand(f, streams))

	// Override help on all the commands tree
	walk(cmd, func(c *cobra.Command) {
		c.Flags().BoolP("help", "h", false, fmt.Sprintf("Help for the %s command", c.Name()))
	})

	return cmd
}

// walk calls f for c and all of its children.
func walk(c *cobra.Command, f func(*cobra.Command)) {
	f(c)
	for _, c := range c.Commands() {
		walk(c, f)
	}
}
