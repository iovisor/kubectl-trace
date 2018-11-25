package cmd

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/fntlnz/kubectl-trace/pkg/factory"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	deleteShort = `Delete a bpftrace program execution` // Wrap with i18n.T()
	deleteLong  = `
...`

	deleteExamples = `
  # Delete a specific bpftrace program by ID
  %[1]s trace delete k656ee75a-ee3c-11e8-9e7a-8c164500a77e

  # Delete a specific bpftrace program by name
  %[1]s trace delete kubectl-trace-1bb3ae39-efe8-11e8-9f29-8c164500a77e

  # Delete all bpftrace programs in a specific namespace
  %[1]s trace delete -n myns"`
)

// DeleteOptions ...
type DeleteOptions struct {
	genericclioptions.IOStreams
	traceID   string
	traceName string
	namespace string
}

// NewDeleteOptions provides an instance of DeleteOptions with default values.
func NewDeleteOptions(streams genericclioptions.IOStreams) *DeleteOptions {
	return &DeleteOptions{
		IOStreams: streams,
	}
}

func (o *DeleteOptions) Validate(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 1:
		o.traceID = args[0]
		break
	default:
		return fmt.Errorf(requiredArgErrString)
	}

	return nil
}

func (o *DeleteOptions) Complete(factory factory.Factory, cmd *cobra.Command, args []string) error {
	// Prepare namespace
	var err error
	o.namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	//// Prepare client
	//clientConfig, err := factory.ToRESTConfig()
	//if err != nil {
	//return err
	//}
	//o.client, err = batchv1client.NewForConfig(clientConfig)
	//if err != nil {
	//return err
	//}

	return nil
}

// NewDeleteCommand provides the delete command wrapping DeleteOptions.
func NewDeleteCommand(factory factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDeleteOptions(streams)

	cmd := &cobra.Command{
		Use:     "delete TRACE_ID",
		Short:   deleteShort,
		Long:    deleteLong,                             // Wrap with templates.LongDesc()
		Example: fmt.Sprintf(deleteExamples, "kubectl"), // Wrap with templates.Examples()
		PreRunE: func(c *cobra.Command, args []string) error {
			return o.Validate(c, args)
		},
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("delete")
			spew.Dump(o)
		},
	}

	return cmd
}
