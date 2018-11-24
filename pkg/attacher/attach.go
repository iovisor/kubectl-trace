package attacher

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/fntlnz/kubectl-trace/pkg/meta"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/term"

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

func (a *Attacher) AttachJob(traceJobID string, namespace string) {
	a.Attach(fmt.Sprintf("%s=%s", meta.TraceIDLabelKey, traceJobID), namespace)
}

func (a *Attacher) Attach(selector, namespace string) {
	go wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		pl, err := a.CoreV1Client.Pods(namespace).List(metav1.ListOptions{
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

		t, err := setupTTY(os.Stdout, os.Stdin)
		if err != nil {
			return false, err
		}
		err = t.Safe(defaultAttachFunc(restClient, pod.Name, containerName, pod.Namespace, a.Config, t))

		if err != nil {
			a.logger.Warn("attach retry", zap.Error(err))
			return false, nil
		}

		return true, nil
	})
	<-a.ctx.Done()
}

func defaultAttachFunc(restClient *restclient.RESTClient, podName string, containerName string, namespace string, config *restclient.Config, t term.TTY) func() error {
	return func() error {
		req := restClient.Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("attach")
		req.VersionedParams(&corev1.PodAttachOptions{
			Container: containerName,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       t.Raw,
		}, scheme.ParameterCodec)

		att := &defaultRemoteAttach{}
		return att.Attach("POST", req.URL(), config, t.In, t.Out, os.Stderr, t.Raw, t.MonitorSize(t.GetSize()))
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

func setupTTY(out io.Writer, in io.Reader) (term.TTY, error) {
	t := term.TTY{
		Out: out,
		In:  in,
		Raw: true,
	}

	if !t.IsTerminalIn() {
		return t, fmt.Errorf("unable to use a TTY if the input is not a terminal")
	}

	return t, nil
}
