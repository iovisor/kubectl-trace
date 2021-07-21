package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/iovisor/kubectl-trace/pkg/procfs"
	"github.com/spf13/cobra"
)

const (
	// MetadataDir is where trace-runner will output traces and metadata
	MetadataDir = "/tmp/kubectl-trace"

	bpftrace = "bpftrace"
	bcc      = "bcc"
	fake     = "fake"
)

var (
	bpfTraceBinaryPath = "/usr/bin/bpftrace"
	bccToolsDir        = "/usr/share/bcc/tools/"
	fakeToolsDir       = "/usr/share/fake/"
)

type TraceRunnerOptions struct {
	// The tracing system to use.
	// tracer = bpftrace | bcc | fake
	tracer string

	podUID string

	containerID string

	// In the case of bcc the name of the bcc program to execute.
	// In the case of bpftrace the path to contents of the user provided expression or program.
	program string

	// In the case of bcc the user provided arguments to pass on to program.
	// Not used for bpftrace.
	programArgs []string
}

func NewTraceRunnerOptions() *TraceRunnerOptions {
	return &TraceRunnerOptions{}
}

func NewTraceRunnerCommand() *cobra.Command {
	o := NewTraceRunnerOptions()
	cmd := &cobra.Command{
		PreRunE: func(c *cobra.Command, args []string) error {
			return o.Validate(c, args)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				fmt.Fprintln(os.Stdout, err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&o.tracer, "tracer", "bpftrace", "Tracing system to use")
	cmd.Flags().StringVar(&o.podUID, "pod-uid", "", "UID of target pod")
	cmd.Flags().StringVar(&o.containerID, "container-id", "", "ID of target container")
	cmd.Flags().StringVar(&o.program, "program", "/programs/program.bt", "Tracer input script or executable")
	cmd.Flags().StringArrayVar(&o.programArgs, "args", o.programArgs, "Arguments to pass through to executable in --program")
	return cmd
}

func (o *TraceRunnerOptions) Validate(cmd *cobra.Command, args []string) error {
	switch o.tracer {
	case bpftrace, bcc, fake:
	default:
		return fmt.Errorf("unknown tracer %s", o.tracer)
	}

	return nil
}

// Complete completes the setup of the command.
func (o *TraceRunnerOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *TraceRunnerOptions) Run() error {
	var err error
	var binary *string
	var args []string

	switch o.tracer {
	case bpftrace:
		binary, args, err = o.prepBpfTraceCommand()
	case bcc:
		binary, args, err = o.prepBccCommand()
	case fake:
		binary, args, err = o.prepFakeCommand()
	}

	if err != nil {
		return err
	}

	// Assume output is stdout until other backends are implemented.
	fmt.Println("if your program has maps to print, send a SIGINT using Ctrl-C, if you want to interrupt the execution send SIGINT two times")
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, os.Signal(syscall.SIGINT))

	go func() {
		killable := false
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-sigCh:
				if !killable {
					killable = true
					fmt.Println("\nfirst SIGINT received, now if your program had maps and did not free them it should print them out")
					continue
				}
				return
			}
		}
	}()

	var c *exec.Cmd
	if len(args) == 0 {
		c = exec.CommandContext(ctx, *binary)
	} else {
		c = exec.CommandContext(ctx, *binary, args...)
	}

	return runTraceCommand(c)
}

func runTraceCommand(c *exec.Cmd) error {
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (o *TraceRunnerOptions) prepBpfTraceCommand() (*string, []string, error) {
	programPath := o.program

	// Render $container_pid to actual process pid if scoped to container.
	if o.podUID != "" && o.containerID != "" {
		pid, err := procfs.FindPidByPodContainer(o.podUID, o.containerID)
		if err != nil {
			return nil, nil, err
		}
		f, err := ioutil.ReadFile(programPath)
		if err != nil {
			return nil, nil, err
		}
		programPath = path.Join(os.TempDir(), "program-container.bt")
		r := strings.Replace(string(f), "$container_pid", pid, -1)
		if err := ioutil.WriteFile(programPath, []byte(r), 0755); err != nil {
			return nil, nil, err
		}
	}

	return &bpfTraceBinaryPath, []string{programPath}, nil
}

func (o *TraceRunnerOptions) prepBccCommand() (*string, []string, error) {
	// Sanitize o.program by removing common prefix/suffixes.
	name := o.program
	name = strings.TrimPrefix(name, "/usr/bin/")
	name = strings.TrimPrefix(name, "/usr/sbin/")
	name = strings.TrimSuffix(name, "-bpfcc")

	program := bccToolsDir + name
	args := append([]string{}, o.programArgs...)

	if o.podUID != "" && o.containerID != "" {
		pid, err := procfs.FindPidByPodContainer(o.podUID, o.containerID)
		if err != nil {
			return nil, nil, err
		}

		for i, arg := range args {
			args[i] = strings.Replace(arg, "$container_pid", pid, -1)
		}
	}

	return &program, args, nil
}

func (o *TraceRunnerOptions) prepFakeCommand() (*string, []string, error) {
	name := path.Base(o.program)
	program := fakeToolsDir + name
	args := append([]string{}, o.programArgs...)

	if o.podUID != "" && o.containerID != "" {
		pid, err := procfs.FindPidByPodContainer(o.podUID, o.containerID)
		if err != nil {
			return nil, nil, err
		}

		for i, arg := range args {
			args[i] = strings.Replace(arg, "$container_pid", pid, -1)
		}
	}

	return &program, args, nil
}
