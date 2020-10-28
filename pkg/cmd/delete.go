package cmd

import (
	"fmt"

	"github.com/iovisor/kubectl-trace/pkg/meta"
	"github.com/iovisor/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var (
	deleteShort = `Delete a bpftrace program execution` // Wrap with i18n.T()
	deleteLong  = deleteShort

	deleteExamples = `
  # Delete a specific bpftrace program by ID
  %[1]s trace delete k656ee75a-ee3c-11e8-9e7a-8c164500a77e

  # Delete a specific bpftrace program by name
  %[1]s trace delete kubectl-trace-1bb3ae39-efe8-11e8-9f29-8c164500a77e

  # Delete all bpftrace programs in a specific namespace
  %[1]s trace delete -n myns --all

  # Delete all bpftrace programs in all the namespaces
  %[1]s trace delete --all-namespaces`
)

// DeleteOptions ...
type DeleteOptions struct {
	genericclioptions.IOStreams
	ResourceBuilderFlags *genericclioptions.ResourceBuilderFlags
	traceID              *types.UID
	traceName            *string
	namespace            string
	clientConfig         *rest.Config
	all                  bool
	allNamespaces        bool
}

// NewDeleteOptions provides an instance of DeleteOptions with default values.
func NewDeleteOptions(streams genericclioptions.IOStreams) *DeleteOptions {
	rbFlags := &genericclioptions.ResourceBuilderFlags{}
	rbFlags.WithAllNamespaces(false)
	rbFlags.WithAll(false)

	return &DeleteOptions{
		ResourceBuilderFlags: rbFlags,
		IOStreams:            streams,
		all:                  false,
	}
}

// NewDeleteCommand provides the delete command wrapping DeleteOptions.
func NewDeleteCommand(factory cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDeleteOptions(streams)

	cmd := &cobra.Command{
		Use:     "delete (TRACE_ID | TRACE_NAME)",
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

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

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
	}

	return nil
}

// Complete completes the setup of the command.
func (o *DeleteOptions) Complete(factory cmdutil.Factory, cmd *cobra.Command, args []string) error {
	// Prepare namespace
	var err error
	o.namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	if cmd.Flag("all-namespaces").Changed {
		o.allNamespaces = *o.ResourceBuilderFlags.AllNamespaces
		o.namespace = ""
	}

	if cmd.Flag("all").Changed {
		o.all = *o.ResourceBuilderFlags.All
	}

	//// Prepare client
	o.clientConfig, err = factory.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.traceID == nil && o.traceName == nil && o.all == false {
		return fmt.Errorf("when no trace id or trace name are specified you must specify --all=true to delete all the traces")
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
