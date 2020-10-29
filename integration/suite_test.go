package integration

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-check/check"
	"github.com/iovisor/kubectl-trace/pkg/cmd"
	"github.com/pkg/errors"
	"gotest.tools/icmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/fs"
)

var (
	KubectlTraceBinary = os.Getenv("TEST_KUBECTLTRACE_BINARY")
)

type KubectlTraceSuite struct {
	kubeConfigPath string
	// kindContext    *cluster.Context

	provider *cluster.Provider
	name     string
}

func init() {
	if KubectlTraceBinary == "" {
		KubectlTraceBinary = "kubectl-trace"
	}

	check.Suite(&KubectlTraceSuite{})
}

func (k *KubectlTraceSuite) SetUpSuite(c *check.C) {
	var err error
	k.name, err = generateClusterName()
	c.Assert(err, check.IsNil)

	k.provider = cluster.NewProvider()
	// Create the cluster
	err = k.provider.Create(
		k.name,
		cluster.CreateWithRetain(false),
		cluster.CreateWithWaitForReady(time.Duration(0)),
		cluster.CreateWithKubeconfigPath(k.kubeConfigPath),

		// todo > we need a logger
		// cluster.ProviderWithLogger(logger),
		// runtime.GetDefault(logger),
	)
	c.Assert(err, check.IsNil)

	nodes, err := k.provider.ListNodes(k.name)
	c.Assert(err, check.IsNil)

	// Copy the bpftrace into a tar
	dir, err := fs.TempDir("", "image-tar")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	imageTarPath := filepath.Join(dir, "image.tar")

	err = save(cmd.ImageName+":"+cmd.ImageTag, imageTarPath)
	c.Assert(err, check.IsNil)

	// Copy the bpftrace image to the nodes
	for _, n := range nodes {
		err = loadImage(imageTarPath, n)
		c.Assert(err, check.IsNil)
	}
}

func (k *KubectlTraceSuite) TearDownSuite(c *check.C) {
	kubeConfig, err := k.provider.KubeConfig(k.name, false)
	c.Assert(err, check.IsNil)
	err = k.provider.Delete(k.name, kubeConfig)
	c.Assert(err, check.IsNil)
}

func Test(t *testing.T) { check.TestingT(t) }

func (k *KubectlTraceSuite) KubectlTraceCmd(c *check.C, args ...string) string {
	args = append([]string{fmt.Sprintf("--kubeconfig=%s", k.kubeConfigPath)}, args...)
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

// loads an image tarball onto a node
func loadImage(imageTarName string, node nodes.Node) error {
	f, err := os.Open(imageTarName)
	if err != nil {
		return errors.Wrap(err, "failed to open image")
	}
	defer f.Close()
	return nodeutils.LoadImageArchive(node, f)
}

// save saves image to dest, as in `docker save`
func save(image, dest string) error {
	return exec.Command("docker", "save", "-o", dest, image).Run()
}
