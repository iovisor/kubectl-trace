package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/fntlnz/kubectl-trace/tracejob"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	kubeconfig := viper.GetString("kubeconfig")
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

	go wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		pl, err := clientset.CoreV1().Pods(apiv1.NamespaceDefault).List(metav1.ListOptions{
			LabelSelector: "job-name=test-renzo",
		})

		if err != nil {
			panic(err)
		}
		if len(pl.Items) == 0 {
			panic("pod not found")
		}
		pod := &pl.Items[0]
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			panic(fmt.Errorf("cannot attach into a container in a completed pod; current phase is %s", pod.Status.Phase))
		}

		if len(pod.Spec.Containers) != 1 {
			panic("unexpected number of containers in trace job pod")
		}

		restClient := clientset.CoreV1().RESTClient().(*restclient.RESTClient)
		containerName := pod.Spec.Containers[0].Name

		attfn := defaultAttachFunc(restClient, pod.Name, containerName, config)

		err = attfn()
		if err != nil {
			return false, nil
		}

		return true, nil
	})

	s := <-c
	fmt.Println("signal:", s)
}

func defaultAttachFunc(restClient *restclient.RESTClient, podName string, containerName string, config *restclient.Config) func() error {
	raw := false
	return func() error {
		req := restClient.Post().
			Resource("pods").
			Name(podName).
			Namespace(apiv1.NamespaceDefault).
			SubResource("attach")
		req.VersionedParams(&corev1.PodAttachOptions{
			Container: containerName,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       raw,
		}, scheme.ParameterCodec)

		att := &DefaultRemoteAttach{}
		return att.Attach("POST", req.URL(), config, os.Stdin, os.Stdout, os.Stderr, raw, nil)
	}
}

type DefaultRemoteAttach struct{}

func (*DefaultRemoteAttach) Attach(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	})
}
