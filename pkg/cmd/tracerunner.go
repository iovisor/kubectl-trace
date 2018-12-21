package cmd

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fntlnz/mountinfo"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

const runFolder = "/var/run"

type TraceRunnerOptions struct {
	podUID             string
	containerName      string
	inPod              bool
	programPath        string
	bpftraceBinaryPath string
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
				return nil
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&o.containerName, "container", "c", o.containerName, "Specify the container")
	cmd.Flags().StringVarP(&o.podUID, "poduid", "p", o.podUID, "Specify the pod UID")
	cmd.Flags().StringVarP(&o.programPath, "program", "f", "program.bt", "Specify the bpftrace program path")
	cmd.Flags().StringVarP(&o.bpftraceBinaryPath, "bpftracebinary", "b", "/bin/bpftrace", "Specify the bpftrace binary path")
	cmd.Flags().BoolVar(&o.inPod, "inpod", false, "Wether or not run this bpftrace in a pod's container process namespace")
	return cmd
}

func (o *TraceRunnerOptions) Validate(cmd *cobra.Command, args []string) error {
	// TODO(fntlnz): do some more meaningful validation here, for now just checking if they are there
	if o.inPod == true && (len(o.containerName) == 0 || len(o.podUID) == 0) {
		return fmt.Errorf("poduid and container must be specified when inpod=true")
	}
	return nil
}

func (o *TraceRunnerOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *TraceRunnerOptions) Run() error {
	if o.inPod == false {
		c := exec.Command(o.bpftraceBinaryPath, o.programPath)
		c.Stdout = os.Stdout
		c.Stdin = os.Stdin
		c.Stderr = os.Stderr
		return c.Run()
	}

	pid, err := findPidByPodContainer(o.podUID, o.containerName)
	if err != nil {
		return err
	}
	if pid == nil {
		return fmt.Errorf("pid not found")
	}
	if len(*pid) == 0 {
		return fmt.Errorf("invalid pid found")
	}

	// pid found, enter its process namespace
	pidns := path.Join("/proc", *pid, "/ns/pid")
	pidnsfd, err := syscall.Open(pidns, syscall.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("error retrieving process namespace %s %v", pidns, err)
	}
	defer syscall.Close(pidnsfd)
	syscall.RawSyscall(unix.SYS_SETNS, uintptr(pidnsfd), 0, 0)

	rootfs := path.Join("/proc", *pid, "root")
	bpftracebinaryName, err := temporaryFileName("bpftrace")
	if err != nil {
		return err
	}
	temporaryProgramName := fmt.Sprintf("%s-%s", bpftracebinaryName, "program.bt")

	binaryPathProcRootfs := path.Join(rootfs, bpftracebinaryName)
	if err := copyFile(o.bpftraceBinaryPath, binaryPathProcRootfs, 0755); err != nil {
		return err
	}

	programPathProcRootfs := path.Join(rootfs, temporaryProgramName)
	if err := copyFile(o.programPath, programPathProcRootfs, 0644); err != nil {
		return err
	}

	if err := syscall.Chroot(rootfs); err != nil {
		os.Remove(binaryPathProcRootfs)
		return err
	}

	defer os.Remove(bpftracebinaryName)

	c := exec.Command(bpftracebinaryName, temporaryProgramName)

	c.Stdout = os.Stdout
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	return c.Run()
}

func copyFile(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("bpftrace binary not found in host: %v", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)

	if err != nil {
		return fmt.Errorf("unable to create file in destination: %v", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("unable to copy file to destination: %v", err)
	}

	err = out.Sync()

	if err != nil {
		return err
	}
	return nil
}

func findPidByPodContainer(podUID, containerName string) (*string, error) {
	d, err := os.Open("/proc")

	if err != nil {
		return nil, err
	}

	defer d.Close()

	for {
		dirs, err := d.Readdir(10)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, di := range dirs {
			if !di.IsDir() {
				continue
			}
			dname := di.Name()
			if dname[0] < '0' || dname[0] > '9' {
				continue
			}

			mi, err := mountinfo.GetMountInfo(path.Join("/proc", dname, "mountinfo"))
			if err != nil {
				continue
			}

			for _, m := range mi {
				root := m.Root
				if strings.Contains(root, podUID) && strings.Contains(root, containerName) {
					return &dname, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no process found for specified pod and container")
}

func temporaryFileName(prefix string) (string, error) {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(runFolder, prefix+hex.EncodeToString(randBytes)), nil
}
