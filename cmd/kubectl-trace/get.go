package main

import (
	"fmt"
	"os"

	"text/tabwriter"

	"github.com/fntlnz/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var getCmd = &cobra.Command{
	Use:   "get [TRACEID] [-n NAMESPACE]",
	Short: "Get all the running traces in a kubernetes cluster",
	Long: `Get all the running traces in a kubernetes cluster

Examples:
	# Get all traces in a namespace
	kubectl trace get -n mynamespace

	# Get only a specific trace
	kubectl trace get 656ee75a-ee3c-11e8-9e7a-8c164500a77e

	# Get only a specific trace in a specific namespace
	kubectl trace get 656ee75a-ee3c-11e8-9e7a-8c164500a77e -n mynamespace

Limitations:
  - Currently work only with a single namespace at time
	- It does not contain yet status and age for the trace
`,
	Run: get,
}

func get(cmd *cobra.Command, args []string) {
	log, _ := zap.NewProduction()
	defer log.Sync()

	kubeconfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		log.Fatal("cannot create kubernetes client from provider KUBECONFIG", zap.Error(err))
	}

	var uuid *string
	if len(args) > 0 {
		uuid = &args[0]
	}

	namespace := viper.GetString("namespace")

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("cannot create kubernetes config from provider KUBECONFIG", zap.Error(err))
	}

	jobsClient := clientset.BatchV1().Jobs(namespace)

	tc := &tracejob.TraceJobClient{
		JobClient:    jobsClient,
		ConfigClient: clientset.CoreV1().ConfigMaps(namespace),
	}

	tf := tracejob.TraceJobFilter{
		ID: uuid,
	}

	jobs, err := tc.GetJob(tf)

	if err != nil {
		log.Fatal("error getting jobs with provided filter", zap.Error(err), zap.Any("filter", tf))
	}

	jobsTablePrint(jobs)

}

// TODO(fntlnz): This needs better printing, perhaps we could use the humanreadable table from k8s itself
// to be consistent with the main project.
func jobsTablePrint(jobs []tracejob.TraceJob) {
	format := "%s\t%s\t%s\t%s\t%s\t"
	if len(jobs) == 0 {
		fmt.Println("No resources found.")
		return
	}
	// initialize tabwriter
	w := new(tabwriter.Writer)
	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	// TODO(fntlnz): Do the status and age fields, we don't have a way to get them now, so reporting
	// them as missing.
	fmt.Fprintf(w, format, "NAMESPACE", "NAME", "STATUS", "AGE", "HOSTNAME")
	for _, j := range jobs {
		fmt.Fprintf(w, "\n"+format, j.Namespace, j.Name, "<missing>", "<missing>", j.Hostname)
	}
}
