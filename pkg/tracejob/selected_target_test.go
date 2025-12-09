package tracejob

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type selectedTargetSuite struct {
	suite.Suite
	clientset *fake.Clientset
}

func TestSelectedTargetSuite(t *testing.T) {
	suite.Run(t, &selectedTargetSuite{})
}

func (s *selectedTargetSuite) SetupTest() {
	s.clientset = fake.NewSimpleClientset()
}

func (s *selectedTargetSuite) TestResolveNodeTargetUsesNodeName() {
	// Create a test node with fully qualified name and different hostname label
	testNodeName := "ip-10-0-1-123.ec2.internal"
	testHostnameLabel := "ip-10-0-1-123"
	
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			Labels: map[string]string{
				"kubernetes.io/hostname": testHostnameLabel,
			},
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				v1.ResourcePods: resource.MustParse("110"),
			},
		},
	}
	
	_, err := s.clientset.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
	assert.Nil(s.T(), err)
	
	// Test resolving the node target
	target, err := ResolveTraceJobTarget(s.clientset, testNodeName, "", "")
	
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), target)
	// The target should use the actual node name, not the hostname label
	assert.Equal(s.T(), testNodeName, target.Node)
}

func (s *selectedTargetSuite) TestResolvePodTargetUsesNodeName() {
	// Create a test node
	testNodeName := "ip-10-0-1-123.ec2.internal"
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			Labels: map[string]string{
				"kubernetes.io/hostname": "ip-10-0-1-123",
			},
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				v1.ResourcePods: resource.MustParse("110"),
			},
		},
	}
	
	_, err := s.clientset.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
	assert.Nil(s.T(), err)
	
	// Create a test pod scheduled on the node
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "test-pod-uid",
		},
		Spec: v1.PodSpec{
			NodeName: testNodeName,
			Containers: []v1.Container{
				{
					Name: "test-container",
				},
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:        "test-container",
					ContainerID: "docker://abc123",
				},
			},
		},
	}
	
	_, err = s.clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.Nil(s.T(), err)
	
	// Test resolving the pod target
	target, err := ResolveTraceJobTarget(s.clientset, "pod/test-pod", "test-container", "default")
	
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), target)
	// The target should use the actual node name from pod.Spec.NodeName
	assert.Equal(s.T(), testNodeName, target.Node)
	assert.Equal(s.T(), "test-pod-uid", target.PodUID)
	assert.Equal(s.T(), "abc123", target.ContainerID)
}

func (s *selectedTargetSuite) TestResolveDeploymentTargetUsesNodeName() {
	// Create a test node
	testNodeName := "ip-10-0-1-123.ec2.internal"
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			Labels: map[string]string{
				"kubernetes.io/hostname": "ip-10-0-1-123",
			},
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				v1.ResourcePods: resource.MustParse("110"),
			},
		},
	}
	
	_, err := s.clientset.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
	assert.Nil(s.T(), err)
	
	// Create a test deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
		},
	}
	
	_, err = s.clientset.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
	assert.Nil(s.T(), err)
	
	// Create a test pod for the deployment
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-pod",
			Namespace: "default",
			UID:       "test-pod-uid",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
		Spec: v1.PodSpec{
			NodeName: testNodeName,
			Containers: []v1.Container{
				{
					Name: "test-container",
				},
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:        "test-container",
					ContainerID: "docker://def456",
				},
			},
		},
	}
	
	_, err = s.clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.Nil(s.T(), err)
	
	// Test resolving the deployment target
	target, err := ResolveTraceJobTarget(s.clientset, "deployment/test-deployment", "test-container", "default")
	
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), target)
	// The target should use the actual node name from pod.Spec.NodeName
	assert.Equal(s.T(), testNodeName, target.Node)
	assert.Equal(s.T(), "test-pod-uid", target.PodUID)
	assert.Equal(s.T(), "def456", target.ContainerID)
}
