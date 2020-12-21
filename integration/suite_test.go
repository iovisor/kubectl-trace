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

	"github.com/iovisor/kubectl-trace/pkg/cmd"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	suite.Suite

	kubeConfigPath string
	// kindContext    *cluster.Context

	provider *cluster.Provider
	name     string
}

func init() {
	if KubectlTraceBinary == "" {
		KubectlTraceBinary = "kubectl-trace"
	}
}

func (k *KubectlTraceSuite) SetupSuite() {
	var err error
	k.name, err = generateClusterName()
	assert.Nil(k.T(), err)

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
	assert.Nil(k.T(), err)

	nodes, err := k.provider.ListNodes(k.name)
	assert.Nil(k.T(), err)

	// Copy the bpftrace into a tar
	dir, err := fs.TempDir("", "image-tar")
	assert.Nil(k.T(), err)
	defer os.RemoveAll(dir)
	imageTarPath := filepath.Join(dir, "image.tar")

	err = save(cmd.ImageName+":"+cmd.ImageTag, imageTarPath)
	assert.Nil(k.T(), err)

	// Copy the bpftrace image to the nodes
	for _, n := range nodes {
		err = loadImage(imageTarPath, n)
		assert.Nil(k.T(), err)
	}
}

func (k *KubectlTraceSuite) TeardownSuite() {
	kubeConfig, err := k.provider.KubeConfig(k.name, false)
	assert.Nil(k.T(), err)
	err = k.provider.Delete(k.name, kubeConfig)
	assert.Nil(k.T(), err)
}

func TestKubectlTraceSuite(t *testing.T) {
	suite.Run(t, &KubectlTraceSuite{})
}

func (k *KubectlTraceSuite) KubectlTraceCmd(args ...string) string {
	args = append([]string{fmt.Sprintf("--kubeconfig=%s", k.kubeConfigPath)}, args...)
	res := icmd.RunCommand(KubectlTraceBinary, args...)
	assert.Equal(k.T(), icmd.Success.ExitCode, res.ExitCode)
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
