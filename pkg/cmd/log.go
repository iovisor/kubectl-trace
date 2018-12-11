package cmd

import (
	"context"
	"fmt"

	"github.com/fntlnz/kubectl-trace/pkg/factory"
	"github.com/fntlnz/kubectl-trace/pkg/logs"
	"github.com/fntlnz/kubectl-trace/pkg/meta"
	"github.com/fntlnz/kubectl-trace/pkg/signals"
	"github.com/fntlnz/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

var (
	logShort = `` // Wrap with i18n.T()
	logLong  = logShort + `

...`

	logExamples = `
  # ...
  %[1]s trace log -h

  # ...
  %[1]s trace log`
)

// LogOptions ...
type LogOptions struct {
	genericclioptions.IOStreams
	traceID      *types.UID
	traceName    *string
	namespace    string
	clientConfig *rest.Config
}

// NewLogOptions provides an instance of LogOptions with default values.
func NewLogOptions(streams genericclioptions.IOStreams) *LogOptions {
	return &LogOptions{
		IOStreams: streams,
	}
}

// NewLogCommand provides the log command wrapping LogOptions.
func NewLogCommand(factory factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewLogOptions(streams)

	cmd := &cobra.Command{
		Use:                   "log (TRACE_ID | TRACE_NAME)",
		DisableFlagsInUseLine: true,
		Short:                 logShort,
		Long:                  logLong,                             // Wrap with templates.LongDesc()
		Example:               fmt.Sprintf(logExamples, "kubectl"), // Wrap with templates.Examples()
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

func (o *LogOptions) Validate(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("(TRACE_ID | TRACE_NAME) is a required argument for the log command")
	}

	return nil
}

func (o *LogOptions) Complete(factory factory.Factory, cmd *cobra.Command, args []string) error {
	// Prepare namespace
	var err error
	o.namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	// Prepare client
	o.clientConfig, err = factory.ToRESTConfig()
	if err != nil {
		return err
	}

	return nil
}

func (o *LogOptions) Run() error {
	jobsClient, err := batchv1client.NewForConfig(o.clientConfig)
	if err != nil {
		return err
	}

	client, err := corev1client.NewForConfig(o.clientConfig)
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
	nl := logs.NewLogs(client, o.IOStreams)
	nl.WithContext(ctx)
	nl.Run(job.ID, job.Namespace)
	return nil
}
