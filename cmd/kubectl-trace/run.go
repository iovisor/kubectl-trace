package main

import (
	"os"

	"github.com/fntlnz/kubectl-trace/tracejob"
	"github.com/spf13/cobra"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute a bpftrace program against your kubernetes cluster",
	Long: `File names and programs are accepted.

Examples:
  # Count system calls using tracepoints on a specific node
  kubectl trace run node kubernetes-node-emt8.c.myproject.internal -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }

  # Execute a program from file on a specific node
  kubectl trace run node kubernetes-node-emt8.c.myproject.internal read.bt
`,
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	jobsClient := clientset.BatchV1().Jobs(apiv1.NamespaceDefault)
	_, err = tracejob.CreateJob(jobsClient)
	if err != nil {
		panic(err)
	}
}
