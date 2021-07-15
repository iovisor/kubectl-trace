package integration

import (
	"regexp"
	"time"

	"github.com/stretchr/testify/assert"

	batchv1 "k8s.io/api/batch/v1"
)

type outputAsserter func(string, []byte)

func (k *KubectlTraceSuite) TestRunNode() {
	nodeName := k.GetTestNode()
	bpftraceProgram := `kprobe:do_sys_open { printf("%s: %s\n", comm, str(arg1)) }'`
	out := k.KubectlTraceCmd("run", "--namespace="+k.namespace(), "--imagename="+k.RunnerImage(), "-e", bpftraceProgram, nodeName)
	assert.Regexp(k.T(), "trace (\\w+-){4}\\w+ created", out)
}

func (k *KubectlTraceSuite) TestReturnErrOnErr() {
	nodeName := k.GetTestNode()

	bpftraceProgram := `kprobe:not_a_real_kprobe { printf("%s: %s\n", comm, str(arg1)) }'`
	out := k.KubectlTraceCmd("run", "--namespace="+k.namespace(), "--imagename="+k.RunnerImage(), "-e", bpftraceProgram, nodeName)
	assert.Regexp(k.T(), regexp.MustCompile("trace [a-f0-9-]{36} created"), out)

	var job batchv1.Job

	for {
		jobs := k.GetJobs().Items
		assert.Equal(k.T(), 1, len(jobs))

		job = jobs[0]
		if len(job.Status.Conditions) > 0 {
			break // on the first condition
		}

		time.Sleep(1 * time.Second)
	}

	assert.Equal(k.T(), 1, len(job.Status.Conditions))
	assert.Equal(k.T(), "Failed", string(job.Status.Conditions[0].Type))
	assert.Equal(k.T(), int32(0), job.Status.Succeeded, "No jobs in the batch should have succeeded")
	assert.Greater(k.T(), job.Status.Failed, int32(1), "There should be at least one failed job")
}
