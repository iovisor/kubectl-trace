package tracejob

import (
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1typed "k8s.io/client-go/kubernetes/typed/batch/v1"
)

func CreateJob(jobClient batchv1typed.JobInterface) (*batchv1.Job, error) {
	bpfTraceCmd := []string{
		"bpftrace",
		"-e",
		`kprobe:do_sys_open { printf("%s: %s\n", comm, str(arg1)) }`,
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-renzo",
			Labels: map[string]string{
				"test": "renzo",
			},
			Annotations: map[string]string{
				"test": "renzo",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:           int32Ptr(1),
			Completions:           int32Ptr(1),
			ActiveDeadlineSeconds: int64Ptr(100), // TODO(fntlnz): allow canceling from kubectl and increase this
			BackoffLimit:          int32Ptr(1),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-renzo-pod",
					Labels: map[string]string{
						"test": "renzo",
					},
					Annotations: map[string]string{
						"test": "renzo",
					},
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
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
							Name:    "test-renzo-container",
							Image:   "quay.io/fntlnz/kubectl-trace-bpftrace:master",
							Command: bpfTraceCmd,
							VolumeMounts: []apiv1.VolumeMount{
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
					NodeSelector:  map[string]string{
						//"": "", // TODO(fntlnz): pass this thing
					},
					//Affinity: &apiv1.Affinity{
					//NodeAffinity: &apiv1.NodeAffinity{
					//RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
					//NodeSelectorTerms: nil,
					//},
					//PreferredDuringSchedulingIgnoredDuringExecution: nil,
					//},
					//},
				},
			},
		},
	}

	return jobClient.Create(job)
}

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }
