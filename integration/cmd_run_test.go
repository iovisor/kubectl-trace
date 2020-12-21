package integration

import (
	"regexp"

	"github.com/stretchr/testify/assert"
)

func (k *KubectlTraceSuite) TestRunNode() {
	nodes, err := k.provider.ListNodes(k.name)
	assert.Nil(k.T(), err)
	assert.Equal(k.T(), 1, len(nodes))

	nodeName := nodes[0].String()
	bpftraceProgram := `kprobe:do_sys_open { printf("%s: %s\n", comm, str(arg1)) }'`
	out := k.KubectlTraceCmd("run", "-e", bpftraceProgram, nodeName)
	match, err := regexp.MatchString("trace (\\w+-){4}\\w+ created", out)
	assert.Nil(k.T(), err)
	assert.True(k.T(), match)
}
