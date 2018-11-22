package cmd

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	// "k8s.io/kubernetes/pkg/kubectl/util/templates"
)

var (
	attachShort = `` // Wrap with i18n.T()
	attachLong  = attachShort + `

...`

	attachExamples = `
  # ...
  %[1]s trace attach -h

  # ...
  %[1]s trace (...) attach`
)

// AttachOptions ...
type AttachOptions struct {
	genericclioptions.IOStreams
}

// NewAttachOptions provides an instance of AttachOptions with default values.
func NewAttachOptions(streams genericclioptions.IOStreams) *AttachOptions {
	return &AttachOptions{
		IOStreams: streams,
	}
}

// NewAttachCommand provides the attach command wrapping AttachOptions.
func NewAttachCommand(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewAttachOptions(streams)

	cmd := &cobra.Command{
		Use:                   "attach TRACE_ID",
		DisableFlagsInUseLine: true,
		Short:                 attachShort,
		Long:                  attachLong,                             // Wrap with templates.LongDesc()
		Example:               fmt.Sprintf(attachExamples, "kubectl"), // Wrap with templates.Examples()
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("attach")
			spew.Dump(o)
		},
	}

	return cmd
}
