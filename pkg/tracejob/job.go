package tracejob

import (
	"fmt"

	"github.com/fntlnz/kubectl-trace/pkg/meta"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	utilpointer "k8s.io/utils/pointer"
)

// TraceJob represents objects we can translate to kubernetes jobs for tracing purposes.
type TraceJob interface {
	Object() *batchv1.Job
	ID() string
}

// traceJob struct contains the info needed to create a trace job.
type traceJob struct {
	id        types.UID
	name      string
	namespace string
	hostname  string
	program   string
}

// NewTraceJob creates a new TraceJob.
func NewTraceJob(namespace, hostname, program string) TraceJob {
	id := uuid.NewUUID()
	return &traceJob{
		id:        id,
		name:      fmt.Sprintf("%s%s", meta.TracePrefix, id),
		namespace: namespace,
		hostname:  hostname,
		program:   program,
	}
}

// ID returns the trace job identifier.
func (j *traceJob) ID() string {
	return string(j.id)
}

// Create setup a new Job for bpftrace program.
func (j *traceJob) Object() *batchv1.Job {
	bpftraceCommand := []string{
		"bpftrace",
		"/programs/program.bt",
	}

	commonMeta := metav1.ObjectMeta{
		Name:      j.name,
		Namespace: j.namespace,
		Labels: map[string]string{
			meta.TraceLabelKey:   j.name,
			meta.TraceIDLabelKey: j.ID(),
		},
		Annotations: map[string]string{
			meta.TraceLabelKey:   j.name,
			meta.TraceIDLabelKey: j.ID(),
		},
	}

	cm := &apiv1.ConfigMap{
		ObjectMeta: commonMeta,
		Data: map[string]string{
			"program.bt": j.program,
		},
	}

	return &batchv1.Job{
		ObjectMeta: commonMeta,
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: utilpointer.Int32Ptr(5),
			Parallelism:             utilpointer.Int32Ptr(1),
			Completions:             utilpointer.Int32Ptr(1),
			// This is why your tracing job is being killed after 100 seconds,
			// someone should work on it to make it configurable and let it run indefinitely by default.
			ActiveDeadlineSeconds: utilpointer.Int64Ptr(100), // TODO(fntlnz): allow canceling from kubectl and increase this,
			BackoffLimit:          utilpointer.Int32Ptr(1),
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
							Name:    j.name,
							Image:   "quay.io/fntlnz/kubectl-trace-bpftrace:master", //TODO(fntlnz): yes this should be configurable!
							TTY:     true,
							Stdin:   true,
							Command: bpftraceCommand,
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
								Privileged: utilpointer.BoolPtr(true),
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
												Values: []string{
													j.hostname,
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
		},
	}
}

// type TraceJobClient struct {
// 	JobClient    batchv1typed.JobInterface
// 	ConfigClient corev1typed.ConfigMapInterface
// }

// type TraceJobFilter struct {
// 	Name *string
// 	ID   *string
// }

// func (t *TraceJobClient) findJobsWithFilter(nf TraceJobFilter) ([]batchv1.Job, error) {
// 	selectorOptions := metav1.ListOptions{}

// 	if nf.Name != nil {
// 		selectorOptions = metav1.ListOptions{
// 			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceLabelKey, *nf.Name),
// 		}
// 	}

// 	if nf.ID != nil {
// 		selectorOptions = metav1.ListOptions{
// 			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceIDLabelKey, *nf.ID),
// 		}
// 	}

// 	jl, err := t.JobClient.List(selectorOptions)

// 	if err != nil {
// 		return nil, err
// 	}
// 	return jl.Items, nil
// }

// func (t *TraceJobClient) findConfigMapsWithFilter(nf TraceJobFilter) ([]apiv1.ConfigMap, error) {
// 	selectorOptions := metav1.ListOptions{}

// 	if nf.Name != nil {
// 		selectorOptions = metav1.ListOptions{
// 			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceLabelKey, *nf.Name),
// 		}
// 	}

// 	if nf.ID != nil {
// 		selectorOptions = metav1.ListOptions{
// 			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceIDLabelKey, *nf.ID),
// 		}
// 	}

// 	cm, err := t.ConfigClient.List(selectorOptions)

// 	if err != nil {
// 		return nil, err
// 	}
// 	return cm.Items, nil
// }

// func (t *TraceJobClient) GetJob(nf TraceJobFilter) ([]TraceJob, error) {

// 	jl, err := t.findJobsWithFilter(nf)
// 	if err != nil {
// 		return nil, err
// 	}
// 	tjobs := []TraceJob{}

// 	for _, j := range jl {
// 		labels := j.GetLabels()
// 		name, ok := labels[meta.TraceLabelKey]
// 		if !ok {
// 			name = ""
// 		}
// 		id, ok := labels[meta.TraceIDLabelKey]
// 		if !ok {
// 			id = ""
// 		}
// 		hostname, err := jobHostname(j)
// 		if err != nil {
// 			hostname = ""
// 		}
// 		tj := TraceJob{
// 			Name:      name,
// 			ID:        id,
// 			Namespace: j.Namespace,
// 			Hostname:  hostname,
// 		}
// 		tjobs = append(tjobs, tj)
// 	}

// 	return tjobs, nil
// }

// func (t *TraceJobClient) DeleteJob(nf TraceJobFilter) error {
// 	jl, err := t.findJobsWithFilter(nf)
// 	if err != nil {
// 		return err
// 	}

// 	dp := metav1.DeletePropagationForeground
// 	for _, j := range jl {
// 		err := t.JobClient.Delete(j.Name, &metav1.DeleteOptions{
// 			PropagationPolicy: &dp,
// 		})
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	cl, err := t.findConfigMapsWithFilter(nf)

// 	if err != nil {
// 		return err
// 	}

// 	for _, c := range cl {
// 		err := t.ConfigClient.Delete(c.Name, nil)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func jobHostname(j batchv1.Job) (string, error) {
// 	aff := j.Spec.Template.Spec.Affinity
// 	if aff == nil {
// 		return "", fmt.Errorf("affinity not found for job")
// 	}

// 	nodeAff := aff.NodeAffinity

// 	if nodeAff == nil {
// 		return "", fmt.Errorf("node affinity not found for job")
// 	}

// 	requiredScheduling := nodeAff.RequiredDuringSchedulingIgnoredDuringExecution

// 	if requiredScheduling == nil {
// 		return "", fmt.Errorf("node affinity RequiredDuringSchedulingIgnoredDuringExecution not found for job")
// 	}
// 	nst := requiredScheduling.NodeSelectorTerms
// 	if len(nst) == 0 {
// 		return "", fmt.Errorf("node selector terms are empty in node affinity for job")
// 	}

// 	me := nst[0].MatchExpressions

// 	if len(me) == 0 {
// 		return "", fmt.Errorf("node selector terms match expressions are empty in node affinity for job")
// 	}

// 	for _, v := range me {
// 		if v.Key == "kubernetes.io/hostname" {
// 			if len(v.Values) == 0 {
// 				return "", fmt.Errorf("hostname affinity found but no values in it for job")
// 			}

// 			return v.Values[0], nil
// 		}
// 	}

// 	return "", fmt.Errorf("hostname not found for job")
// }
