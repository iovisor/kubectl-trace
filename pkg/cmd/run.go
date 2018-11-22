package cmd

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/fntlnz/kubectl-trace/factory"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	// "k8s.io/kubernetes/pkg/kubectl/util/templates"
)

var (
	runShort = `Execute a bpftrace program on resources` // Wrap with i18n.T()

	runLong = runShort

	runExamples = `
  # Count system calls using tracepoints on a specific node
  %[1]s trace run node/kubernetes-node-emt8.c.myproject.internal -e 'kprobe:do_sys_open { printf("%s: %s\n", comm, str(arg1)) }'

  # Execute a bpftrace program from file on a specific node
  %[1]s trace run node/kubernetes-node-emt8.c.myproject.internal -p read.bt

  # Run an bpftrace inline program on a pod container
  %[1]s trace run pod/nginx -c nginx -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }"
  %[1]s trace run pod/nginx nginx -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }"`

	runCommand                    = "run"
	usageString                   = "(POD | TYPE/NAME)"
	requiredArgErrString          = fmt.Sprintf("%s is a required argument for the %s command", usageString, runCommand)
	containerAsArgOrFlagErrString = "specify container inline as argument or via its flag"
	bpftraceMissingErrString      = "the bpftrace program is mandatory"
)

// RunOptions ...
type RunOptions struct {
	genericclioptions.IOStreams

	namespace string

	// Local to this command
	container   string
	eval        string
	program     string
	resourceArg string
}

// NewRunOptions provides an instance of RunOptions with default values.
func NewRunOptions(streams genericclioptions.IOStreams) *RunOptions {
	return &RunOptions{
		IOStreams: streams,
	}
}

// NewRunCommand provides the run command wrapping RunOptions.
func NewRunCommand(factory factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewRunOptions(streams)

	cmd := &cobra.Command{
		Use:          fmt.Sprintf("%s %s [-c CONTAINER]", runCommand, usageString),
		Short:        runShort,
		Long:         runLong,                             // Wrap with templates.LongDesc()
		Example:      fmt.Sprintf(runExamples, "kubectl"), // Wrap with templates.Examples()
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

	cmd.Flags().StringVarP(&o.container, "container", "c", o.container, "Specify the container")
	cmd.Flags().StringVarP(&o.eval, "eval", "e", "", "Literal string to be evaluated as a bpftrace program")
	cmd.Flags().StringVarP(&o.program, "program", "p", "", "File containing a bpftrace program")

	return cmd
}

// Validate validates the arguments and flags populating RunOptions accordingly.
func (o *RunOptions) Validate(cmd *cobra.Command, args []string) error {
	containerDefined := cmd.Flag("container").Changed
	switch len(args) {
	case 1:
		o.resourceArg = args[0]
		if !containerDefined {
			return fmt.Errorf(containerAsArgOrFlagErrString)
		}
		break
	// 2nd argument interpreted as container when provided
	case 2:
		o.resourceArg = args[0]
		o.container = args[1]
		if containerDefined {
			return fmt.Errorf(containerAsArgOrFlagErrString)
		}
		break
	default:
		return fmt.Errorf(requiredArgErrString)
	}

	if !cmd.Flag("eval").Changed && !cmd.Flag("program").Changed {
		return fmt.Errorf(bpftraceMissingErrString)
	}

	// todo > complete validation
	// - make errors
	// - make validators
	if len(o.container) == 0 {
		return fmt.Errorf("invalid container")
	}
	// if len(o.eval) == 0 || file not exist(o.program) || file is empty(o.program) {
	// 	return fmt.Errorf("invalid bpftrace program")
	// }

	return nil
}

// Complete completes the setup of the command.
func (o *RunOptions) Complete(factory factory.Factory, cmd *cobra.Command, args []string) error {
	spew.Dump(o)

	o.namespace, _, _ = factory.ToRawKubeConfigLoader().Namespace()

	spew.Dump(o)

	// get resource by pof | type/name
	// get container

	return nil
}
