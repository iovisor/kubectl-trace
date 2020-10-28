package cmd

import (
	"fmt"

	"github.com/iovisor/kubectl-trace/pkg/logs"
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
	logShort    = `Print the logs for a specific trace execution` // Wrap with i18n.T()
	logLong     = logShort
	logExamples = `
  # Logs from a trace using its name
  %[1]s trace logs kubectl-trace-d5842929-0b78-11e9-a9fa-40a3cc632df1

  # Logs from a trace using its id
  %[1]s trace log 5594d7e1-0b78-11e9-b7f1-40a3cc632df1

  # Follow logs
  %[1]s trace logs kubectl-trace-d5842929-0b78-11e9-a9fa-40a3cc632df1 -f

  # Add timestamp to logs
  %[1]s trace logs kubectl-trace-d5842929-0b78-11e9-a9fa-40a3cc632df1 --timestamp
`
)

// LogOptions ...
type LogOptions struct {
	genericclioptions.IOStreams
	traceID      *types.UID
	traceName    *string
	namespace    string
	clientConfig *rest.Config
	follow       bool
	timestamps   bool
}

// NewLogOptions provides an instance of LogOptions with default values.
func NewLogOptions(streams genericclioptions.IOStreams) *LogOptions {
	return &LogOptions{
		IOStreams:  streams,
		follow:     false,
		timestamps: false,
	}
}

// NewLogCommand provides the log command wrapping LogOptions.
func NewLogCommand(factory cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewLogOptions(streams)

	cmd := &cobra.Command{
		Use:                   "logs (TRACE_ID | TRACE_NAME) [-f]",
		DisableFlagsInUseLine: true,
		Aliases:               []string{"log"},
		Short:                 logShort,
		Long:                  logLong,                             // Wrap with templates.LongDesc()
		Example:               fmt.Sprintf(logExamples, "kubectl"), // Wrap with templates.Examples()
		Args:                  cobra.ExactArgs(1),
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

	cmd.Flags().BoolVarP(&o.follow, "follow", "f", o.follow, "Specify if the logs should be streamed")
	cmd.Flags().BoolVar(&o.timestamps, "timestamps", o.timestamps, "Include timestamps on each line in the log output")
	return cmd
}

// Validate validates the arguments and flags populating LogOptions accordingly.
func (o *LogOptions) Validate(cmd *cobra.Command, args []string) error {
	if meta.IsObjectName(args[0]) {
		o.traceName = &args[0]
	} else {
		tid := types.UID(args[0])
		o.traceID = &tid
	}

	return nil
}

// Complete completes the setup of the command.
func (o *LogOptions) Complete(factory cmdutil.Factory, cmd *cobra.Command, args []string) error {
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

	nl := logs.NewLogs(client, o.IOStreams)
	nl.Run(job.ID, job.Namespace, o.follow, o.timestamps)
	return nil
}
