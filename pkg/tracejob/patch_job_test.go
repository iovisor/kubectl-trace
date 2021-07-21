package tracejob

import (
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
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
