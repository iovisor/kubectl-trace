package tracejob

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "default"
)

type jobSuite struct {
	suite.Suite
	client *TraceJobClient
}

func TestJobSuite(t *testing.T) {
	suite.Run(t, &jobSuite{})
}

func (j *jobSuite) SetupTest() {
	j.client = NewTraceJobClient(fake.NewSimpleClientset(), testNamespace)
}

func (j *jobSuite) TestCreateJob() {
	testJobName := "test-basic-create"
	tj := TraceJob{
		Name: testJobName,
	}

	job, err := j.client.CreateJob(tj)

	assert.Nil(j.T(), err)
	assert.NotNil(j.T(), job)

	joblist, err := j.client.JobClient.List(context.TODO(), metav1.ListOptions{})
	assert.Nil(j.T(), err)
	assert.NotNil(j.T(), joblist)

	assert.Len(j.T(), joblist.Items, 1)
	assert.Equal(j.T(), joblist.Items[0].Spec.Template.Spec.Containers[0].Name, testJobName)
}

func (j *jobSuite) TestCreateJobWithGoogleAppSecret() {
	testJobName := "test-create-with-google-app-secret"
	tj := TraceJob{
		Name:            testJobName,
		GoogleAppSecret: "test-gcp-secret",
	}

	job, err := j.client.CreateJob(tj)

	assert.Nil(j.T(), err)
	assert.NotNil(j.T(), job)

	joblist, err := j.client.JobClient.List(context.TODO(), metav1.ListOptions{})
	assert.Nil(j.T(), err)
	assert.NotNil(j.T(), joblist)

	assert.Len(j.T(), joblist.Items, 1)
	assert.Equal(j.T(), joblist.Items[0].Spec.Template.Spec.Containers[0].Name, testJobName)

	assert.Len(j.T(), joblist.Items[0].Spec.Template.Spec.Containers[0].Env, 1)
	assert.Equal(j.T(), joblist.Items[0].Spec.Template.Spec.Containers[0].Env[0].Name, "GOOGLE_APPLICATION_CREDENTIALS")
}

func (j *jobSuite) TestJobNodeSelectorUsesNodeName() {
	testJobName := "test-node-selector"
	testNodeName := "ip-10-0-1-123.ec2.internal"
	tj := TraceJob{
		Name: testJobName,
		Target: TraceJobTarget{
			Node: testNodeName,
		},
	}

	job := tj.Job()

	// Verify that the job uses MatchFields with metadata.name instead of kubernetes.io/hostname
	assert.NotNil(j.T(), job.Spec.Template.Spec.Affinity)
	assert.NotNil(j.T(), job.Spec.Template.Spec.Affinity.NodeAffinity)
	assert.NotNil(j.T(), job.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	
	nodeSelectorTerms := job.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	assert.Len(j.T(), nodeSelectorTerms, 1)
	
	matchFields := nodeSelectorTerms[0].MatchFields
	assert.Len(j.T(), matchFields, 1)
	
	assert.Equal(j.T(), "metadata.name", matchFields[0].Key)
	assert.Equal(j.T(), "In", string(matchFields[0].Operator))
	assert.Len(j.T(), matchFields[0].Values, 1)
	assert.Equal(j.T(), testNodeName, matchFields[0].Values[0])
}

func (j *jobSuite) TestJobHostnameExtraction() {
	testNodeName := "ip-10-0-1-123.ec2.internal"
	
	// Test with new MatchFields approach
	jobWithMatchFields := &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					Affinity: &apiv1.Affinity{
						NodeAffinity: &apiv1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
								NodeSelectorTerms: []apiv1.NodeSelectorTerm{
									apiv1.NodeSelectorTerm{
										MatchFields: []apiv1.NodeSelectorRequirement{
											apiv1.NodeSelectorRequirement{
												Key:      "metadata.name",
												Operator: apiv1.NodeSelectorOpIn,
												Values:   []string{testNodeName},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	nodeName, err := jobHostname(*jobWithMatchFields)
	assert.Nil(j.T(), err)
	assert.Equal(j.T(), testNodeName, nodeName)
	
	// Test backward compatibility with old MatchExpressions approach
	jobWithMatchExpressions := &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					Affinity: &apiv1.Affinity{
						NodeAffinity: &apiv1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
								NodeSelectorTerms: []apiv1.NodeSelectorTerm{
									apiv1.NodeSelectorTerm{
										MatchExpressions: []apiv1.NodeSelectorRequirement{
											apiv1.NodeSelectorRequirement{
												Key:      "kubernetes.io/hostname",
												Operator: apiv1.NodeSelectorOpIn,
												Values:   []string{"ip-10-0-1-123"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	hostname, err := jobHostname(*jobWithMatchExpressions)
	assert.Nil(j.T(), err)
	assert.Equal(j.T(), "ip-10-0-1-123", hostname)
}
