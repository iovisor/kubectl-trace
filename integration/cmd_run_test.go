package integration

import (
	"regexp"

	"github.com/go-check/check"
)

func (k *KubectlTraceSuite) TestRunNode(c *check.C) {
	nodes, err := k.provider.ListNodes()
	c.Assert(err, check.IsNil)
	c.Assert(len(nodes), check.Equals, 1)

	nodeName := nodes[0].String()
	bpftraceProgram := `kprobe:do_sys_open { printf("%s: %s\n", comm, str(arg1)) }'`
	out := k.KubectlTraceCmd(c, "run", "-e", bpftraceProgram, nodeName)
	match, err := regexp.MatchString("trace (\\w+-){4}\\w+ created", out)
	c.Assert(err, check.IsNil)
	c.Assert(match, check.Equals, true)
}
