package logs

import (
	"github.com/fntlnz/kubectl-trace/pkg/meta"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"fmt"
	"io"

	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Logs struct {
	genericclioptions.IOStreams
	coreV1Client tcorev1.CoreV1Interface
	ctx          context.Context
}

func NewLogs(client tcorev1.CoreV1Interface, streams genericclioptions.IOStreams) *Logs {
	return &Logs{
		coreV1Client: client,
		IOStreams:    streams,
		ctx:          context.TODO(),
	}
}

const (
	podNotFoundError              = "no trace found to get logs from with the given selector"
	podPhaseNotAcceptedError      = "cannot get logs from a completed trace; current phase is %s"
	invalidPodContainersSizeError = "unexpected number of containers in trace job pod"
)

func (l *Logs) WithContext(c context.Context) {
	l.ctx = c
}

func (l *Logs) Run(jobID types.UID, namespace string) error {
	pl, err := l.coreV1Client.Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", meta.TraceIDLabelKey, jobID),
	})

	if err != nil {
		return err
	}

	if len(pl.Items) == 0 {
		return fmt.Errorf(podNotFoundError)
	}

	pod := &pl.Items[0]
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return fmt.Errorf(podPhaseNotAcceptedError, pod.Status.Phase)
	}

	if len(pod.Spec.Containers) != 1 {
		return fmt.Errorf(invalidPodContainersSizeError)
	}

	containerName := pod.Spec.Containers[0].Name

	// TODO(fntlnz): let the user choose to follow or not
	logOptions := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		Previous:   false,
		Timestamps: false,
	}

	logsRequest := l.coreV1Client.Pods(namespace).GetLogs(pod.Name, logOptions)

	go consumeRequest(logsRequest, l.IOStreams.Out)
	<-l.ctx.Done()

	return nil
}

func consumeRequest(request *rest.Request, out io.Writer) error {
	readCloser, err := request.Stream()
	if err != nil {
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(out, readCloser)
	return err
}

