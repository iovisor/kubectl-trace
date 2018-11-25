package cmd

import (
	"fmt"

	"github.com/fntlnz/kubectl-trace/pkg/factory"
	"github.com/fntlnz/kubectl-trace/pkg/meta"
	"github.com/fntlnz/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
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
  %[1]s trace delete -n myns --all`
)

// DeleteOptions ...
type DeleteOptions struct {
	genericclioptions.IOStreams
	traceID      *types.UID
	traceName    *string
	namespace    string
	clientConfig *rest.Config
	all          bool
}

// NewDeleteOptions provides an instance of DeleteOptions with default values.
func NewDeleteOptions(streams genericclioptions.IOStreams) *DeleteOptions {
	return &DeleteOptions{
		IOStreams: streams,
		all:       false,
	}
}

// NewDeleteCommand provides the delete command wrapping DeleteOptions.
func NewDeleteCommand(factory factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDeleteOptions(streams)

	cmd := &cobra.Command{
		Use:     "delete [TRACE_ID] [--all]",
		Short:   deleteShort,
		Long:    deleteLong,                             // Wrap with templates.LongDesc()
		Example: fmt.Sprintf(deleteExamples, "kubectl"), // Wrap with templates.Examples()
		PreRunE: func(c *cobra.Command, args []string) error {
			return o.Validate(c, args)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(factory, c, args); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				fmt.Fprintln(o.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&o.all, "all", o.all, "Delete all trace jobs in the provided namespace")

	return cmd
}

func (o *DeleteOptions) Validate(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 1:
		if meta.IsObjectName(args[0]) {
			o.traceName = &args[0]
		} else {
			tid := types.UID(args[0])
			o.traceID = &tid
		}
		break
	default:
		if o.all == false {
			return fmt.Errorf("--all=true must be specified to delete all the trace programs in a namespace\n%s", requiredArgErrString)
		}
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
	o.clientConfig, err = factory.ToRESTConfig()
	if err != nil {
		return err
	}

	return nil
}

func (o *DeleteOptions) Run() error {
	jobsClient, err := batchv1client.NewForConfig(o.clientConfig)
	if err != nil {
		return err
	}

	coreClient, err := corev1client.NewForConfig(o.clientConfig)
	if err != nil {
		return err
	}

	tc := &tracejob.TraceJobClient{
		JobClient:    jobsClient.Jobs(o.namespace),
		ConfigClient: coreClient.ConfigMaps(o.namespace),
	}

	tc.WithOutStream(o.Out)

	tf := tracejob.TraceJobFilter{
		Name: o.traceName,
		ID:   o.traceID,
	}

	err = tc.DeleteJobs(tf)
	if err != nil {
		return err
	}

	return nil
}
