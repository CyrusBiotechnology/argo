package executor

import (
	"testing"

	wfv1 "github.com/cyrusbiotechnology/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cyrusbiotechnology/argo/workflow/executor/mocks"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	fakePodName     = "fake-test-pod-1234567890"
	fakeNamespace   = "default"
	fakeAnnotations = "/tmp/podannotationspath"
	fakeContainerID = "abc123"
)

func TestSaveParameters(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	templateWithOutParam := wfv1.Template{
		Outputs: wfv1.Outputs{
			Parameters: []wfv1.Parameter{
				{
					Name: "my-out",
					ValueFrom: &wfv1.ValueFrom{
						Path: "/path",
					},
				},
			},
		},
	}
	we := WorkflowExecutor{
		PodName:            fakePodName,
		Template:           templateWithOutParam,
		ClientSet:          fakeClientset,
		Namespace:          fakeNamespace,
		PodAnnotationsPath: fakeAnnotations,
		ExecutionControl:   nil,
		RuntimeExecutor:    &mockRuntimeExecutor,
		mainContainerID:    fakeContainerID,
	}
	mockRuntimeExecutor.On("GetFileContents", fakeContainerID, "/path").Return("has a newline\n", nil)
	err := we.SaveParameters()
	assert.Nil(t, err)
	assert.Equal(t, *we.Template.Outputs.Parameters[0].Value, "has a newline")
}

//func TestEvaluateErrorConditions(t *testing.T) {
//
//	we := WorkflowExecutor{}
//
//	conditions := []wfv1.ErrorCondition{
//		{
//			Name:           "testConditionMatch",
//			PatternMatched: "test log file",
//			Message:        "test condition was triggered",
//		},
//		{
//			Name:             "testConditionUnmatch",
//			PatternUnmatched: "unmatched log file",
//			Message:          "test condition was triggered",
//		},
//	}
//
//	logContent := []byte("test log file")
//
//	results, err := we.evaluatePatternConditions(&conditions, &logContent)
//	assert.Nil(t, err)
//
//	expectedResult := []wfv1.ErrorResult{
//		{
//			Name:    "testConditionMatch",
//			Message: "test condition was triggered",
//		},
//		{
//			Name:    "testConditionUnmatch",
//			Message: "test condition was triggered",
//		},
//	}
//
//	assert.Equal(t, results, expectedResult)
//
//	logContentNoMatch := []byte("unmatched log file")
//
//	results, err = we.evaluatePatternConditions(&conditions, &logContentNoMatch)
//	assert.Nil(t, err)
//
//	assert.Equal(t, results, []wfv1.ErrorResult(nil))
//
//}
