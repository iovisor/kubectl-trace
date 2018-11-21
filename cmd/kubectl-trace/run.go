package main

import (
	"context"

	"github.com/fntlnz/kubectl-trace/attacher"
	"github.com/fntlnz/kubectl-trace/tracejob"
	"github.com/influxdata/platform/kit/signals"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var runCmd = &cobra.Command{
	Use:   "run NODE [-e PROGRAM] [FILENAME]",
	Short: "Execute a bpftrace program against a NODE in your kubernetes cluster",
	Long: `File names and programs are accepted.

Examples:
  # Count system calls using tracepoints on a specific node
  kubectl trace run kubernetes-node-emt8.c.myproject.internal -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }

  # Execute a program from file on a specific node
  kubectl trace run kubernetes-node-emt8.c.myproject.internal read.bt
`,
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()
	ctx = signals.WithStandardSignals(ctx)

	kubeconfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		logger.Fatal("cannot create kubernetes client from provider KUBECONFIG", zap.Error(err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatal("cannot create kubernetes config from provider KUBECONFIG", zap.Error(err))
	}

	namespace := apiv1.NamespaceDefault

	jobsClient := clientset.BatchV1().Jobs(namespace)
	job, err := tracejob.CreateJob(jobsClient)
	if err != nil {
		logger.Fatal("cannot create kubernetes job client", zap.Error(err))
	}

	a := attacher.NewAttacher(clientset.CoreV1(), config)
	a.WithLogger(logger)
	a.WithContext(ctx)

	a.AttachJob(job.Name, job.Namespace)

	<-ctx.Done()
}
