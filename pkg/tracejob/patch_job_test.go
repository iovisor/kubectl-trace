package tracejob

import (
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TODO (dalehamel): move this back into job_test.go

type patchTest struct {
	patchType string
	patch     []byte
}

var patchJSON1 = []byte(`
- op: replace
  path: "/spec/backOffLimit"
  value: 123
- op: add
  path: "/spec/template/hostPID"
  value: true
- op: remove
  path: "/spec/completions"
`)
var patchJSON2 = []byte(`[
  { "op": "replace", "path": "/spec/backOffLimit", "value": 123 },
  { "op": "add", "path": "/spec/template/hostPID", "value": true },
  { "op": "remove", "path": "/spec/completions"}
]`)

var patchMerge = []byte(`
spec:
  backoffLimit: 123
  completions: null
  template:
    hostPID: true
`)

// merge key for PodSpec.Volumes is "name", and Container.volumeMounts is "mountPath"
// ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#podspec-v1-core
var addHostVolumeMountMerge = []byte(`
spec:
  template:
    spec:
      containers:
        - name: kubectl-trace
          volumeMounts:
            - name: trace-output-2
              mountPath: /tmp/kubectl-trace
            - name: extra-volume
              mountPath: /home/some/volume
              readOnly: true
      volumes:
        - name: trace-output
          emptyDir:
            sizeLimit: 10Mi
        - name: extra-volume
          hostPath:
            path: /home/some/volume
            type: Directory
`)

var testCases = []patchTest{
	{patchType: "json", patch: patchJSON1},
	{patchType: "json", patch: patchJSON2},
	{patchType: "merge", patch: patchMerge},
	{patchType: "strategic", patch: patchMerge},
}

func TestPatchJobJSON(t *testing.T) {
	for _, c := range testCases {
		job := getJob()
		newJob, err := patchJob(job, c.patchType, c.patch)
		if err != nil {
			t.Error(err)
		}

		// Update expected value
		job.Spec.BackoffLimit = int32Ptr(123)
		job.Spec.Template.Spec.HostPID = true
		job.Spec.Completions = nil

		if reflect.DeepEqual(job, newJob) {
			t.Errorf("patch %s job does not match expected", c.patchType)
		}
	}
}

func getJob() *batchv1.Job {
	return &batchv1.Job{
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds:   int64Ptr(60),
			TTLSecondsAfterFinished: int32Ptr(5),
			Parallelism:             int32Ptr(1),
			Completions:             int32Ptr(1),
			BackoffLimit:            int32Ptr(1),
		},
	}
}

func getJobWithContainer() *batchv1.Job {
	return &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "kubectl-trace",
							Image: "hello",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "sys",
									MountPath: "/sys",
									ReadOnly:  true,
								},
								{
									Name:      "trace-output",
									MountPath: "/tmp/kubectl-trace",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "sys",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "trace-output",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{
									SizeLimit: quantityPtr(resource.MustParse("1Gi")),
								},
							},
						},
					},
				},
			},
		},
	}
}

func newHostPathType(pathType string) *v1.HostPathType {
	hostPathType := new(v1.HostPathType)
	*hostPathType = v1.HostPathType(pathType)
	return hostPathType
}

func getPatchedJobWithContainer() *batchv1.Job {
	return &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "kubectl-trace",
							Image: "hello",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "sys",
									MountPath: "/sys",
									ReadOnly:  true,
								},
								{
									Name:      "trace-output-2",
									MountPath: "/tmp/kubectl-trace",
									ReadOnly:  true,
								},
								{
									Name:      "extra-volume",
									MountPath: "/home/some/volume",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "sys",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "trace-output",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{
									SizeLimit: quantityPtr(resource.MustParse("10Mi")),
								},
							},
						},
						{
							Name: "extra-volume",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/home/some/volume",
									Type: newHostPathType(string(v1.HostPathDirectory)),
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestContainerStrategicPatch(t *testing.T) {
	job := getJobWithContainer()
	newJob, err := patchJob(job, "strategic", addHostVolumeMountMerge)
	if err != nil {
		t.Error(err)
	}

	expected := getPatchedJobWithContainer()

	if !reflect.DeepEqual(newJob, expected) {
		t.Errorf("patch host volume job does not match expected")
	}
}
