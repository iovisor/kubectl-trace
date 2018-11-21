package attacher

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type Attacher struct {
	ctx          context.Context
	CoreV1Client tcorev1.CoreV1Interface
	logger       *zap.Logger
	Config       *restclient.Config
}

func NewAttacher(client tcorev1.CoreV1Interface, config *restclient.Config) *Attacher {
	return &Attacher{
		CoreV1Client: client,
		Config:       config,
		logger:       zap.NewNop(),
		ctx:          context.TODO(),
	}
}

const (
	podNotFoundError              = "no pod found to attach with the given selector"
	podPhaseNotAcceptedError      = "cannot attach into a container in a completed pod; current phase is %s"
	invalidPodContainersSizeError = "unexpected number of containers in trace job pod"
)

func (a *Attacher) WithLogger(l *zap.Logger) {
	if l == nil {
		a.logger = zap.NewNop()
		return
	}
	a.logger = l
}

func (a *Attacher) WithContext(c context.Context) {
	a.ctx = c
}

func (a *Attacher) AttachJob(jobName, namespace string) {
	a.Attach(fmt.Sprintf("job-name=%s", jobName), namespace)
}

func (a *Attacher) Attach(selector, namespace string) {
	go wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		pl, err := a.CoreV1Client.Pods(namespace).List(metav1.ListOptions{
			//LabelSelector: "job-name=test-renzo",
			LabelSelector: selector,
		})

		if err != nil {
			return false, err
		}

		if len(pl.Items) == 0 {
			return false, fmt.Errorf(podNotFoundError)
		}
		pod := &pl.Items[0]
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return false, fmt.Errorf(podPhaseNotAcceptedError, pod.Status.Phase)
		}

		if len(pod.Spec.Containers) != 1 {
			return false, fmt.Errorf(invalidPodContainersSizeError)
		}

		restClient := a.CoreV1Client.RESTClient().(*restclient.RESTClient)
		containerName := pod.Spec.Containers[0].Name

		attfn := defaultAttachFunc(restClient, pod.Name, containerName, a.Config)

		err = attfn()
		if err != nil {
			a.logger.Warn("attach retry", zap.Error(err))
			return false, nil
		}

		return true, nil
	})
	<-a.ctx.Done()
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

		att := &defaultRemoteAttach{}
		return att.Attach("POST", req.URL(), config, os.Stdin, os.Stdout, os.Stderr, raw, nil)
	}
}

type defaultRemoteAttach struct{}

func (*defaultRemoteAttach) Attach(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
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
