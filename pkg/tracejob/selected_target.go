/*

Currently, we are doing a bunch of crazy stuff in order to build our trace job.

We have two ways of specifying the thing we actually want to trace:

- From a literal target resource ( run.go 's o.resourceArg, taken from arg[0])
  - resourceArg is overloaded, it can be either the node, or the pod which we would like to trace
    - In the future, it may be other types, eg, deployment/NAME, or statefulset/NAME, etc.
  - additionally, o.container is specified further with arg[1], and may be required for cases where resourceArg is not a Node.

Complicating this even further, we also permit for the target to be given by a Selector string

Because of this, the selector string is parsed into a struct with a mutable hash, and we end up using
the selector struct as a way to store our actual target.

To simplify this, we should separate the **desired target** from the **selected target**.

If the pod / container are specified with positional arguments, then the selector **must not**
include a selector for the pod or container, but is permitted to have a process selector.

It is also valid to have no positional arguments at all, and rely entirely on the
selector to specify the pod / container to target.

1. Resolve the Target
  - Look at any positional arguments (from CLI)
  - Look at the Selector
  - Perform validations
  - Resolve to:
    - At minimum, always a node to run the tracejob on (inferred through pod, and perhaps through deployment → pod)
    - If a pod is being targeted, it may also have a Pod target and a Container
2. Create the TraceJob
  - The fully specify target (the output from 1, above) is provided as a parameter
  - The expectation is that a TraceJob is an entity that is scheduled on a given node:
    - Other logic (1) is responsible for determining the correct node to run on
    - Only necessary selector arguments are passed to the trace-runner, as an argument within the trace job

*/

package tracejob

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/iovisor/kubectl-trace/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

/*
This struct will be used when creating the TraceJob
*/

type TraceJobTarget struct {
	Node        string // Used for tracejob NodeSelector
	PodUID      string // passed as argument to trace-runner
	ContainerID string // passed as argument to trace-runner
}

/*
ResolveTraceJobTarget will:

- take a kubernetes resource, possibly namespaced, and resolve it to a concrete node that the tracejob can run on
  - if the resource is a node, the scope of the trace will be the entire node
  - if the resource is a pod, or something that can be resolved to a pod:
    - resource is namespaced, the targetNamespace is required
    - if the pod has multiple containers, a container name is required, otherwise the only container will be used

clientset       - a kubernetes clientset, used for resolving the target resource
resource        - a string, indicating the kubernetes resource to be traced. Currently supported: node, pod
container       - (optional) the container to trace, required only if needed to disambiguate
targetNamespace - (optional) the target namespace of a given resource, required if the resource is namespaced

*/

func ResolveTraceJobTarget(clientset kubernetes.Interface, resource, container, targetNamespace string) (*TraceJobTarget, error) {

	target := TraceJobTarget{}

	var resourceType string
	var resourceID string

	resourceParts := strings.Split(resource, "/")

	if len(resourceParts) < 1 || len(resourceParts) > 2 {
		return nil, fmt.Errorf("Invalid resource string %s", resource)
	} else if len(resourceParts) == 1 {
		resourceType = "node"
		resourceID = resourceParts[0]
	} else if len(resourceParts) == 2 {
		resourceType = resourceParts[0]
		resourceID = resourceParts[1]
	}

	switch resourceType {
	case "node":
		nodeClient := clientset.CoreV1().Nodes()
		allocatable, err := NodeIsAllocatable(clientset, resourceID)
		if err != nil {
			return nil, err
		}
		if !allocatable {
			return nil, errors.NewErrorUnallocatable(fmt.Sprintf("Node %s is not allocatable", resourceID))
		}
		node, err := nodeClient.Get(context.TODO(), resourceID, metav1.GetOptions{})
		if err != nil {
			return nil, errors.NewErrorInvalid(fmt.Sprintf("Failed to locate a node for %s %v", resourceID, err))
		}

		labels := node.GetLabels()
		val, ok := labels["kubernetes.io/hostname"]
		if !ok {
			return nil, errors.NewErrorInvalid("label kubernetes.io/hostname not found in node")
		}
		target.Node = val

	case "pod":
		podClient := clientset.CoreV1().Pods(targetNamespace)
		pod, err := podClient.Get(context.TODO(), resourceID, metav1.GetOptions{})
		if err != nil {
			return nil, errors.NewErrorInvalid(fmt.Sprintf("Could not locate pod %s, err: %v", resourceID, err))
		}

		allocatable, err := NodeIsAllocatable(clientset, pod.Spec.NodeName)
		if err != nil {
			return nil, err
		}
		if !allocatable {
			return nil, errors.NewErrorUnallocatable(fmt.Sprintf("Pod %s is not scheduled on an allocatable node", resourceID))
		}

		err = resolvePodToTarget(podClient, resourceID, container, targetNamespace, &target)
		if err != nil {
			return nil, err
		}

	case "deploy", "deployment":
		deployClient := clientset.AppsV1().Deployments(targetNamespace)
		deployment, err := deployClient.Get(context.TODO(), resourceID, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelMap, err := metav1.LabelSelectorAsMap(deployment.Spec.Selector)
		if err != nil {
			return nil, err
		}
		podClient := clientset.CoreV1().Pods(targetNamespace)
		pods, err := podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelMap).String()})
		if err != nil {
			return nil, err
		}

		var selectedPod *v1.Pod
		for _, pod := range pods.Items {
			allocatable, err := NodeIsAllocatable(clientset, pod.Spec.NodeName)
			if err != nil {
				continue
			}

			if allocatable {
				selectedPod = &pod
				break
			}
		}

		if selectedPod == nil {
			return nil, errors.NewErrorUnallocatable(fmt.Sprintf("No pods for deployment %s were on allocatable nodes", resourceID))
		}

		err = resolvePodToTarget(podClient, selectedPod.Name, container, targetNamespace, &target)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.NewErrorInvalid(fmt.Sprintf("Unsupported resource type %s for %s\n", resourceType, resourceID))
	}

	return &target, nil
}

func NodeIsAllocatable(clientset kubernetes.Interface, hostname string) (bool, error) {
	// if the job is neither Successful nor Failed, here we need to check to make sure that it is at least
	// schedulable on the node. To do that
	//
	// 	 retrieve a list of all pods currently scheduled on the node
	//	 check the number of pods available on the node
	//   make sure that there are "enough" pods available
	//   	 - if not, mark as "Unschedulable"
	//   account for Unschedulable elsewhere in the operator logic?

	allPods, err := allPodsForNode(clientset, hostname)
	if err != nil {
		return false, fmt.Errorf("could not retrieve pods %v", err)
	}

	var nonTerminalPods []*v1.Pod
	for _, pod := range allPods.Items {
		if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			continue
		}

		nonTerminalPods = append(nonTerminalPods, &pod)
	}

	fmt.Printf("Got %d pods, %d non-terminal pods", len(allPods.Items), len(nonTerminalPods))

	nodeClient := clientset.CoreV1().Nodes()
	node, err := nodeClient.Get(context.TODO(), hostname, metav1.GetOptions{})

	if err != nil {
		return false, fmt.Errorf("could not get node object %v", err)
	}

	maxPods, ok := node.Status.Allocatable.Pods().AsInt64()
	if !ok {
		err = fmt.Errorf("quantity was not an integer: %s", node.Status.Allocatable.Pods().String())
		return false, fmt.Errorf("could not parse the number of allocatable pods %v", err)
	}

	if len(nonTerminalPods) >= int(maxPods) {
		return false, nil
	}

	// FIXME check resources, check for cordoned before giving the all-clear

	return true, nil
}

func allPodsForNode(clientset kubernetes.Interface, nodeName string) (*v1.PodList, error) {
	// https://github.com/kubernetes/kubectl/blob/1199011a44e83ded0d4f1d582e731bc4212aa9f5/pkg/describe/describe.go#L3473
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + nodeName + ",status.phase!=" + string(v1.PodSucceeded) + ",status.phase!=" + string(v1.PodFailed))
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
		// TODO should we chunk this? RE https://github.com/kubernetes/kubectl/blob/1199011a44e83ded0d4f1d582e731bc4212aa9f5/pkg/describe/describe.go#L3484
	}

	podList, err := clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}

	return podList, nil
}

func resolvePodToTarget(podClient corev1.PodInterface, resourceID, container, targetNamespace string, target *TraceJobTarget) error {
	pod, err := podClient.Get(context.TODO(), resourceID, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("failed to locate a pod for %s; error was %s", resourceID, err.Error())
	}
	if pod.Spec.NodeName == "" {
		return fmt.Errorf("cannot attach a trace program to a pod that is not currently scheduled on a node")
	}

	var targetContainer string
	target.Node = pod.Spec.NodeName
	target.PodUID = string(pod.UID)

	if len(pod.Spec.Containers) == 1 {
		targetContainer = pod.Spec.Containers[0].Name
	} else {
		// FIXME verify container is not empty
		targetContainer = container
	}

	for _, s := range pod.Status.ContainerStatuses {
		if s.Name == targetContainer {
			containerID := strings.TrimPrefix(s.ContainerID, "docker://")
			containerID = strings.TrimPrefix(containerID, "containerd://")
			target.ContainerID = containerID
			break
		}
	}

	if target.ContainerID == "" {
		return fmt.Errorf("no containers found for the provided pod %s and container %s combination", pod.Name, targetContainer)
	}
	return nil
}
