package tracejob

import (
	"fmt"

	"github.com/fntlnz/kubectl-trace/pkg/meta"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
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

type TraceJobFilter struct {
	Name *string
	ID   *string
}

func (t *TraceJobClient) findJobsWithFilter(nf TraceJobFilter) ([]batchv1.Job, error) {
	selectorOptions := metav1.ListOptions{}

	if nf.Name != nil {
		selectorOptions = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceLabelKey, *nf.Name),
		}
	}

	if nf.ID != nil {
		selectorOptions = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceIDLabelKey, *nf.ID),
		}
	}

	jl, err := t.JobClient.List(selectorOptions)

	if err != nil {
		return nil, err
	}
	return jl.Items, nil
}

func (t *TraceJobClient) findConfigMapsWithFilter(nf TraceJobFilter) ([]apiv1.ConfigMap, error) {
	selectorOptions := metav1.ListOptions{}

	if nf.Name != nil {
		selectorOptions = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceLabelKey, *nf.Name),
		}
	}

	if nf.ID != nil {
		selectorOptions = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", meta.TraceIDLabelKey, *nf.ID),
		}
	}

	cm, err := t.ConfigClient.List(selectorOptions)

	if err != nil {
		return nil, err
	}
	return cm.Items, nil
}

func (t *TraceJobClient) GetJob(nf TraceJobFilter) ([]TraceJob, error) {

	jl, err := t.findJobsWithFilter(nf)
	if err != nil {
		return nil, err
	}
	tjobs := []TraceJob{}

	for _, j := range jl {
		labels := j.GetLabels()
		name, ok := labels[meta.TraceLabelKey]
		if !ok {
			name = ""
		}
		id, ok := labels[meta.TraceIDLabelKey]
		if !ok {
			id = ""
		}
		hostname, err := jobHostname(j)
		if err != nil {
			hostname = ""
		}
		tj := TraceJob{
			Name:      name,
			ID:        id,
			Namespace: j.Namespace,
			Hostname:  hostname,
		}
		tjobs = append(tjobs, tj)
	}

	return tjobs, nil
}

func (t *TraceJobClient) DeleteJob(nf TraceJobFilter) error {
	jl, err := t.findJobsWithFilter(nf)
	if err != nil {
		return err
	}

	dp := metav1.DeletePropagationForeground
	for _, j := range jl {
		err := t.JobClient.Delete(j.Name, &metav1.DeleteOptions{
			PropagationPolicy: &dp,
		})
		if err != nil {
			return err
		}
	}

	cl, err := t.findConfigMapsWithFilter(nf)

	if err != nil {
		return err
	}

	for _, c := range cl {
		err := t.ConfigClient.Delete(c.Name, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// Create setup a new Job for bpftrace program.
func Create(j TraceJob) *batchv1.Job {
	j.ID = string(uuid.NewUUID())
	j.Name = fmt.Sprintf("%s%s", meta.TracePrefix, j.ID)

	bpftraceCommand := []string{
		"bpftrace",
		"/programs/program.bt",
	}

	commonMeta := metav1.ObjectMeta{
		Name:      j.Name,
		Namespace: j.Namespace,
		Labels: map[string]string{
			meta.TraceLabelKey:   j.Name,
			meta.TraceIDLabelKey: j.ID,
		},
		Annotations: map[string]string{
			meta.TraceLabelKey:   j.Name,
			meta.TraceIDLabelKey: j.ID,
		},
	}

	cm := &apiv1.ConfigMap{
		ObjectMeta: commonMeta,
		Data: map[string]string{
			"program.bt": j.Program,
		},
	}

	return &batchv1.Job{
		ObjectMeta: commonMeta,
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: int32Ptr(5),
			Parallelism:             int32Ptr(1),
			Completions:             int32Ptr(1),
			// This is why your tracing job is being killed after 100 seconds,
			// someone should work on it to make it configurable and let it run indefinitely by default.
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
							Name:    j.Name,
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
												Values:   []string{j.Hostname},
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

// todo(fntlnz): deal with programs that needs the user to send a signal to complete,
// like how the hist() function does
// Will likely need to allocate a TTY for this one thing.
// func (t *TraceJobClient) CreateJob(j TraceJob) (*batchv1.Job, error) {

// 	if _, err := t.ConfigClient.Create(cm); err != nil {
// 		return nil, err
// 	}
// 	return t.JobClient.Create(job)
// }

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }

func jobHostname(j batchv1.Job) (string, error) {
	aff := j.Spec.Template.Spec.Affinity
	if aff == nil {
		return "", fmt.Errorf("affinity not found for job")
	}

	nodeAff := aff.NodeAffinity

	if nodeAff == nil {
		return "", fmt.Errorf("node affinity not found for job")
	}

	requiredScheduling := nodeAff.RequiredDuringSchedulingIgnoredDuringExecution

	if requiredScheduling == nil {
		return "", fmt.Errorf("node affinity RequiredDuringSchedulingIgnoredDuringExecution not found for job")
	}
	nst := requiredScheduling.NodeSelectorTerms
	if len(nst) == 0 {
		return "", fmt.Errorf("node selector terms are empty in node affinity for job")
	}

	me := nst[0].MatchExpressions

	if len(me) == 0 {
		return "", fmt.Errorf("node selector terms match expressions are empty in node affinity for job")
	}

	for _, v := range me {
		if v.Key == "kubernetes.io/hostname" {
			if len(v.Values) == 0 {
				return "", fmt.Errorf("hostname affinity found but no values in it for job")
			}

			return v.Values[0], nil
		}
	}

	return "", fmt.Errorf("hostname not found for job")
}
