package integration

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-check/check"
	"github.com/iovisor/kubectl-trace/pkg/cmd"
	"gotest.tools/icmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/fs"
)

var (
	KubectlTraceBinary = os.Getenv("TEST_KUBECTLTRACE_BINARY")
)

type KubectlTraceSuite struct {
	kubeConfigPath string
	kindContext    *cluster.Context
}

func init() {
	if KubectlTraceBinary == "" {
		KubectlTraceBinary = "kubectl-trace"
	}

	check.Suite(&KubectlTraceSuite{})
}

func (k *KubectlTraceSuite) SetUpSuite(c *check.C) {
	clusterName, err := generateClusterName()
	c.Assert(err, check.IsNil)
	kctx := cluster.NewContext(clusterName)

	err = kctx.Create(create.Retain(false), create.WaitForReady(time.Duration(0)))
	c.Assert(err, check.IsNil)
	k.kindContext = kctx

	nodes, err := kctx.ListNodes()

	c.Assert(err, check.IsNil)

	// copy the bpftrace into a tar
	dir, err := fs.TempDir("", "image-tar")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	imageTarPath := filepath.Join(dir, "image.tar")

	err = docker.Save(cmd.ImageNameTag, imageTarPath)
	c.Assert(err, check.IsNil)

	f, err := os.Open(imageTarPath)
	c.Assert(err, check.IsNil)

	// copy the bpftrace image to the nodes
	for _, n := range nodes {
		err = n.LoadImageArchive(f)
		c.Assert(err, check.IsNil)
	}
}

func (k *KubectlTraceSuite) TearDownSuite(c *check.C) {
	err := k.kindContext.Delete()
	c.Assert(err, check.IsNil)
}

func Test(t *testing.T) { check.TestingT(t) }

func (k *KubectlTraceSuite) KubectlTraceCmd(c *check.C, args ...string) string {
	args = append([]string{fmt.Sprintf("--kubeconfig=%s", k.kindContext.KubeConfigPath())}, args...)
	res := icmd.RunCommand(KubectlTraceBinary, args...)
	c.Assert(res.ExitCode, check.Equals, icmd.Success.ExitCode)
	return res.Combined()
}

func generateClusterName() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToLower(fmt.Sprintf("%X", buf)), nil
}
