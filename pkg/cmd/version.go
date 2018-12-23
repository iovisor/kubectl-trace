package cmd

import (
	"fmt"

	"github.com/fntlnz/kubectl-trace/pkg/version"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewVersionCommand(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information for kubectl trace",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(streams.Out, version.String())
			return nil
		},
	}
	return cmd
}
