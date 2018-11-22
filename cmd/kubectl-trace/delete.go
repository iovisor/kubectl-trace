package main

import (
	"github.com/fntlnz/kubectl-trace/pkg/tracejob"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var deleteCmd = &cobra.Command{
	Use:   "delete TRACEID",
	Short: "",
	Long:  "",
	Run:   delete,
}

func delete(cmd *cobra.Command, args []string) {
	log, _ := zap.NewProduction()
	defer log.Sync()

	uuid := args[0]
	if len(args) == 0 {
		log.Fatal("TRACEID not provided")
	}
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

	tc := &tracejob.TraceJobClient{
		JobClient:    jobsClient,
		ConfigClient: clientset.CoreV1().ConfigMaps(namespace),
	}

	tj := tracejob.TraceJob{
		ID: uuid,
	}

	err = tc.DeleteJob(tj)

	if err != nil {
		log.Fatal("error deleting trace execution from cluster", zap.Error(err))
	}

	log.Info("trace execution deleted")

}
