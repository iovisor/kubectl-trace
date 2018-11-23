package cmd

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/fntlnz/kubectl-trace/pkg/factory"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	// "k8s.io/kubernetes/pkg/kubectl/util/templates"
)

var (
	deleteShort = `Delete a bpftrace program execution` // Wrap with i18n.T()
	deleteLong  = `
...`

	deleteExamples = `
  # Delete a specific bpftrace program
  %[1]s trace delete k656ee75a-ee3c-11e8-9e7a-8c164500a77e

  # Delete all bpftrace programs in a specific namespace
  %[1]s trace delete -n myns"`
)

// DeleteOptions ...
type DeleteOptions struct {
	genericclioptions.IOStreams
}

// NewDeleteOptions provides an instance of DeleteOptions with default values.
func NewDeleteOptions(streams genericclioptions.IOStreams) *DeleteOptions {
	return &DeleteOptions{
		IOStreams: streams,
	}
}

// NewDeleteCommand provides the delete command wrapping DeleteOptions.
func NewDeleteCommand(factory factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDeleteOptions(streams)

	cmd := &cobra.Command{
		Use:     "delete TRACE_ID",
		Short:   deleteShort,
		Long:    deleteLong,                             // Wrap with templates.LongDesc()
		Example: fmt.Sprintf(deleteExamples, "kubectl"), // Wrap with templates.Examples()
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("delete")
			spew.Dump(o)
		},
	}

	return cmd
}
