package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/fntlnz/kubectl-trace/pkg/attacher"
	"github.com/fntlnz/kubectl-trace/pkg/signals"
	"github.com/fntlnz/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/uuid"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var runCmd = &cobra.Command{
	Use:   "run NODE [-e PROGRAM] [-f FILENAME] [-n NAMESPACE]",
	Short: "Execute a bpftrace program against a NODE in your kubernetes cluster",
	Long: `File names and programs are accepted.

Examples:
  # Count system calls using tracepoints on a specific node
  kubectl trace run kubernetes-node-emt8.c.myproject.internal -e 'kprobe:do_sys_open { printf("%s: %s\n", comm, str(arg1)) }''

  # Execute a program from file on a specific node
  kubectl trace run kubernetes-node-emt8.c.myproject.internal -f read.bt
`,
	Run: run,
}

var program string
var programfile string
var namespace string

func init() {
	runCmd.Flags().StringVarP(&program, "program-literal", "e", "", "Literal string containing a bpftrace program")
	runCmd.Flags().StringVarP(&programfile, "program-file", "f", "", "File containing a bpftrace program")
	runCmd.Flags().StringVarP(&namespace, "namespace", "n", apiv1.NamespaceDefault, "Name of the node where to do the trace")
}

func run(cmd *cobra.Command, args []string) {
	log, _ := zap.NewProduction()
	defer log.Sync()

	if len(programfile) > 0 {
		b, err := ioutil.ReadFile(programfile)
		if err != nil {
			log.Fatal("error opening program file", zap.Error(err))
		}
		program = string(b)
	}
	if len(program) == 0 {
		log.Fatal("program not provided")
	}

	if len(args) == 0 {
		log.Fatal("node not provided")
	}
	node := args[0]

	ctx := context.Background()
	ctx = signals.WithStandardSignals(ctx)

	kubeconfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		log.Fatal("cannot create kubernetes client from provider KUBECONFIG", zap.Error(err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("cannot create kubernetes config from provider KUBECONFIG", zap.Error(err))
	}

	jobsClient := clientset.BatchV1().Jobs(namespace)

	juid := uuid.NewUUID()
	tc := &tracejob.TraceJobClient{
		JobClient:    jobsClient,
		ConfigClient: clientset.CoreV1().ConfigMaps(namespace),
	}

	tj := tracejob.TraceJob{
		Name:      fmt.Sprintf("kubectl-trace-%s", string(juid)),
		Namespace: namespace,
		ID:        string(juid),
		Hostname:  node,
		Program:   program,
	}
	job, err := tc.CreateJob(tj)
	if err != nil {
		log.Fatal("cannot create kubernetes job client", zap.Error(err))
	}

	a := attacher.NewAttacher(clientset.CoreV1(), config)
	a.WithLogger(log)
	a.WithContext(ctx)

	a.AttachJob(tj.ID, job.Namespace)

	<-ctx.Done()
}
