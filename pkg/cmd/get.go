package cmd

import (
	"fmt"

	"github.com/fntlnz/kubectl-trace/pkg/factory"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	// "k8s.io/kubernetes/pkg/kubectl/util/templates"
)

var (
	getCommand = "get"
	getShort   = `Get the running traces` // Wrap with i18n.T()
	getLong    = getShort + `
	
...`

	getExamples = `
  # Get all traces in a namespace
  %[1]s trace get -n myns

  # Get only a specific trace
  %[1]s trace get 656ee75a-ee3c-11e8-9e7a-8c164500a77e

  # Get only a specific trace in a specific namespace
  %[1]s trace get 656ee75a-ee3c-11e8-9e7a-8c164500a77e -n myns

  # Get all traces in all namespaces
  %[1]s trace get --all-namespaces`

	argumentsErr     = fmt.Sprintf("at most one argument for %s command", getCommand)
	missingTargetErr = fmt.Sprintf("specify either a TRACE_ID or a namespace or all namespaces")
)

// GetOptions ...
type GetOptions struct {
	genericclioptions.IOStreams
	ResourceBuilderFlags *genericclioptions.ResourceBuilderFlags

	namespace         string
	explicitNamespace bool

	// Local to this command
	allNamespaces bool
	traceArg      string
}

// NewGetOptions provides an instance of GetOptions with default values.
func NewGetOptions(streams genericclioptions.IOStreams) *GetOptions {
	rbFlags := &genericclioptions.ResourceBuilderFlags{}
	rbFlags.WithAllNamespaces(false)

	return &GetOptions{
		ResourceBuilderFlags: rbFlags,
		IOStreams:            streams,
	}
}

// NewGetCommand provides the get command wrapping GetOptions.
func NewGetCommand(factory factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewGetOptions(streams)

	cmd := &cobra.Command{
		Use:          fmt.Sprintf("%s [TRACE_ID]", getCommand),
		Short:        getShort,
		Long:         getLong,                             // wrap with templates.LongDesc()
		Example:      fmt.Sprintf(getExamples, "kubectl"), // wrap with templates.Examples()
		SilenceUsage: true,
		PreRunE: func(c *cobra.Command, args []string) error {
			return o.Validate(c, args)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(factory, c, args); err != nil {
				return err
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

	return cmd
}

// Validate validates the arguments and flags populating GetOptions accordingly.
func (o *GetOptions) Validate(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 0:
		break
	case 1:
		o.traceArg = args[0]
		break
	default:
		return fmt.Errorf(argumentsErr)
	}

	return nil
}

// Complete completes the setup of the command.
func (o *GetOptions) Complete(factory factory.Factory, cmd *cobra.Command, args []string) error {
	o.namespace, o.explicitNamespace, _ = factory.ToRawKubeConfigLoader().Namespace()

	if cmd.Flag("all-namespaces").Changed {
		o.allNamespaces = *o.ResourceBuilderFlags.AllNamespaces
		o.explicitNamespace = false
		o.namespace = ""
	}
	if o.traceArg == "" && !o.allNamespaces && !o.explicitNamespace {
		return fmt.Errorf(missingTargetErr)
	}

	return nil
}
