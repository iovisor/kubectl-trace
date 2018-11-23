package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/davecgh/go-spew/spew"
	"github.com/fntlnz/kubectl-trace/pkg/factory"
	"github.com/fntlnz/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"

	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
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
  %[1]s trace run pod/nginx nginx -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }"
  %[1]s trace run pod/nginx nginx -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }"`

	runCommand                    = "run"
	usageString                   = "(POD | TYPE/NAME)"
	requiredArgErrString          = fmt.Sprintf("%s is a required argument for the %s command", usageString, runCommand)
	containerAsArgOrFlagErrString = "specify container inline as argument or via its flag"
	bpftraceMissingErrString      = "the bpftrace program is mandatory"
	bpftraceDoubleErrString       = "specify the bpftrace program either via an external file or via a literal string, not both"
	bpftraceEmptyErrString        = "the bpftrace programm cannot be empty"
)

// RunOptions ...
type RunOptions struct {
	genericclioptions.IOStreams

	namespace         string
	explicitNamespace bool

	// Local to this command
	container   string
	eval        string
	program     string
	resourceArg string

	client batchv1client.BatchV1Interface
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
			if err := o.Run(); err != nil {
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
	containerFlagDefined := cmd.Flag("container").Changed
	switch len(args) {
	case 1:
		o.resourceArg = args[0]
		break
	// 2nd argument interpreted as container when provided
	case 2:
		o.resourceArg = args[0]
		o.container = args[1]
		if containerFlagDefined {
			return fmt.Errorf(containerAsArgOrFlagErrString)
		}
		break
	default:
		return fmt.Errorf(requiredArgErrString)
	}

	if !cmd.Flag("eval").Changed && !cmd.Flag("program").Changed {
		return fmt.Errorf(bpftraceMissingErrString)
	}
	if cmd.Flag("eval").Changed == cmd.Flag("program").Changed {
		return fmt.Errorf(bpftraceDoubleErrString)
	}
	if (cmd.Flag("eval").Changed && len(o.eval) == 0) || (cmd.Flag("program").Changed && len(o.program) == 0) {
		return fmt.Errorf(bpftraceEmptyErrString)
	}

	// todo > complete validation
	// - make errors
	// - make validators
	// if len(o.container) == 0 {
	// 	return fmt.Errorf("invalid container")
	// }

	return nil
}

// Complete completes the setup of the command.
func (o *RunOptions) Complete(factory factory.Factory, cmd *cobra.Command, args []string) error {
	// Prepare program
	if len(o.program) > 0 {
		b, err := ioutil.ReadFile(o.program)
		if err != nil {
			return fmt.Errorf("error opening program file")
		}
		o.program = string(b)
	} else {
		o.program = o.eval
	}

	// Prepare namespace
	var err error
	o.namespace, o.explicitNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	// Look for the target object
	x := factory.
		NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(o.namespace).
		SingleResourceType().
		ResourceNames("pods", o.resourceArg). // Search pods by default
		Do()

	obj, err := x.Object()
	if err != nil {
		return err
	}

	spew.Dump(obj)

	// Check we got a pod or a node
	// isPod := false
	switch obj.(type) {
	case *v1.Pod:
		// isPod = true
		break
	case *v1.Node:
		break
	default:
		return fmt.Errorf("first argument must be %s", usageString)
	}

	// Check we also have container if we got a pod
	// if o.container == "" && isPod {
	// 	return fmt.Errorf("missing pod container")
	// }

	// Prepare client
	clientConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	o.client, err = batchv1client.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	// todo > setup printer
	// printer, err := o.PrintFlags.ToPrinter()
	// if err != nil {
	// 	return err
	// }
	// o.print = func(obj runtime.Object) error {
	// 	return printer.PrintObj(obj, o.Out)
	// }

	return nil
}

// Run executes the run command.
func (o *RunOptions) Run() error {
	tj := tracejob.TraceJob{
		Namespace: o.namespace,
		Program:   o.program,
		Hostname:  o.resourceArg,
	}

	spew.Dump(tj)
	fmt.Println(o.container)

	_, err := o.client.Jobs(o.namespace).Create(tracejob.Create(tj))
	if err != nil {
		return err
	}

	// todo > what to print here: this trace job all job trace jobs?
	// o.print(_)
	return nil
}
