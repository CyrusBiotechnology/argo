package controller

import "github.com/cyrusbiotechnology/argo/pkg/apis/workflow/v1alpha1"

const TraceId "CyrusWorkflowTraceId"

func GetTraceFromWorkflow(wf *v1alpha1.Workflow) {
	wf.Annotations
}