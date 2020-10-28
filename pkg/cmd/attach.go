package cmd

import (
	"context"
	"fmt"

	"github.com/iovisor/kubectl-trace/pkg/attacher"
	"github.com/iovisor/kubectl-trace/pkg/meta"
	"github.com/iovisor/kubectl-trace/pkg/signals"
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
	attachShort = `Attach to an existing trace` // Wrap with i18n.T()
	attachLong  = attachShort

	attachExamples = `
	# Attach to a trace using its name
	%[1]s trace attach kubectl-trace-d5842929-0b78-11e9-a9fa-40a3cc632df1

	# Attach to a trace using its id
	%[1]s trace attach 5594d7e1-0b78-11e9-b7f1-40a3cc632df1

	# Attach to a trace in a namespace using its name
	%[1]s trace attach kubectl-trace-d5842929-0b78-11e9-a9fa-40a3cc632df1 -n mynamespace
`
)

// AttachOptions ...
type AttachOptions struct {
	genericclioptions.IOStreams
	traceID      *types.UID
	traceName    *string
	namespace    string
	clientConfig *rest.Config
}

// NewAttachOptions provides an instance of AttachOptions with default values.
func NewAttachOptions(streams genericclioptions.IOStreams) *AttachOptions {
	return &AttachOptions{
		IOStreams: streams,
	}
}

// NewAttachCommand provides the attach command wrapping AttachOptions.
func NewAttachCommand(factory cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewAttachOptions(streams)

	cmd := &cobra.Command{
		Use:                   "attach (TRACE_ID | TRACE_NAME)",
		DisableFlagsInUseLine: true,
		Short:                 attachShort,
		Long:                  attachLong,                             // Wrap with templates.LongDesc()
		Example:               fmt.Sprintf(attachExamples, "kubectl"), // Wrap with templates.Examples()
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

	return cmd
}

func (o *AttachOptions) Validate(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("(TRACE_ID | TRACE_NAME) is a required argument for the attach command")
	}

	return nil
}

// Complete completes the setup of the command.
func (o *AttachOptions) Complete(factory cmdutil.Factory, cmd *cobra.Command, args []string) error {
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

func (o *AttachOptions) Run() error {
	jobsClient, err := batchv1client.NewForConfig(o.clientConfig)
	if err != nil {
		return err
	}

	coreClient, err := corev1client.NewForConfig(o.clientConfig)
	if err != nil {
		return err
	}

	tc := &tracejob.TraceJobClient{
		JobClient: jobsClient.Jobs(o.namespace),
	}

	tf := tracejob.TraceJobFilter{
		Name: o.traceName,
		ID:   o.traceID,
	}

	jobs, err := tc.GetJob(tf)

	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return fmt.Errorf("no trace found with the provided criterias")
	}

	job := jobs[0]

	ctx := context.Background()
	ctx = signals.WithStandardSignals(ctx)
	a := attacher.NewAttacher(coreClient, o.clientConfig, o.IOStreams)
	a.WithContext(ctx)
	a.AttachJob(job.ID, job.Namespace)
	return nil
}
