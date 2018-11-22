package tracejob

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1typed "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1typed "k8s.io/client-go/kubernetes/typed/core/v1"
)

type TraceJobClient struct {
	JobClient    batchv1typed.JobInterface
	ConfigClient corev1typed.ConfigMapInterface
}

type TraceJob struct {
	Name      string
	ID        string
	Namespace string
	Hostname  string
	Program   string
}

func (t *TraceJobClient) DeleteJob(nj TraceJob) error {
	selectorOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("fntlnz.wtf/kubectl-trace-id=%s", nj.ID),
	}
	jl, err := t.JobClient.List(selectorOptions)

	if err != nil {
		return err
	}

	for _, j := range jl.Items {
		err := t.JobClient.Delete(j.Name, nil)
		if err != nil {
			return err
		}
	}

	cl, err := t.ConfigClient.List(selectorOptions)

	if err != nil {
		return err
	}

	for _, c := range cl.Items {
		err := t.ConfigClient.Delete(c.Name, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// todo(fntlnz): deal with programs that needs the user to send a signal to complete,
// like how the hist() function does
// Will likely need to allocate a TTY for this one thing.
func (t *TraceJobClient) CreateJob(nj TraceJob) (*batchv1.Job, error) {
	bpfTraceCmd := []string{
		"bpftrace",
		"/programs/program.bt",
	}

	commonMeta := metav1.ObjectMeta{
		Name:      nj.Name,
		Namespace: nj.Namespace,
		Labels: map[string]string{
			"fntlnz.wtf/kubectl-trace":    nj.Name,
			"fntlnz.wtf/kubectl-trace-id": nj.ID,
		},
		Annotations: map[string]string{
			"fntlnz.wtf/kubectl-trace":    nj.Name,
			"fntlnz.wtf/kubectl-trace-id": nj.ID,
		},
	}

	cm := &apiv1.ConfigMap{
		ObjectMeta: commonMeta,
		Data: map[string]string{
			"program.bt": nj.Program,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: commonMeta,
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: int32Ptr(5),
			Parallelism:             int32Ptr(1),
			Completions:             int32Ptr(1),
			// This is why your tracing job is being killed after 100 seconds,
			// someone should work on it to make it configurable and let it run
			// indefinitely by default.
			ActiveDeadlineSeconds: int64Ptr(100), // TODO(fntlnz): allow canceling from kubectl and increase this,
			BackoffLimit:          int32Ptr(1),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: commonMeta,
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						apiv1.Volume{
							Name: "program",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: cm.Name,
									},
								},
							},
						},
						apiv1.Volume{
							Name: "modules",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/lib/modules",
								},
							},
						},
						apiv1.Volume{
							Name: "sys",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
					},
					Containers: []apiv1.Container{
						apiv1.Container{
							Name:    nj.Name,
							Image:   "quay.io/fntlnz/kubectl-trace-bpftrace:master", //TODO(fntlnz): yes this should be configurable!
							Command: bpfTraceCmd,
							VolumeMounts: []apiv1.VolumeMount{
								apiv1.VolumeMount{
									Name:      "program",
									MountPath: "/programs",
									ReadOnly:  true,
								},
								apiv1.VolumeMount{
									Name:      "modules",
									MountPath: "/lib/modules",
									ReadOnly:  true,
								},
								apiv1.VolumeMount{
									Name:      "sys",
									MountPath: "/sys",
									ReadOnly:  true,
								},
							},
							SecurityContext: &apiv1.SecurityContext{
								Privileged: boolPtr(true),
							},
						},
					},
					RestartPolicy: "Never",
					Affinity: &apiv1.Affinity{
						NodeAffinity: &apiv1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
								NodeSelectorTerms: []apiv1.NodeSelectorTerm{
									apiv1.NodeSelectorTerm{
										MatchExpressions: []apiv1.NodeSelectorRequirement{
											apiv1.NodeSelectorRequirement{
												Key:      "kubernetes.io/hostname",
												Operator: apiv1.NodeSelectorOpIn,
												Values:   []string{nj.Hostname},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := t.ConfigClient.Create(cm); err != nil {
		return nil, err
	}
	return t.JobClient.Create(job)
}

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }
