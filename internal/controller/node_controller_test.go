package controller

import (
	"context"
	"testing"

	sjapi "cerit.cz/scavenger-job/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var sjs = &sjapi.ScavengerJobList{
	Items: []sjapi.ScavengerJob{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sj1",
				Labels: map[string]string{
					CERIT_CPU: "1.0",
					CERIT_MEM: "1.0",
				},
			},
			Status: sjapi.ScavengerJobStatus{
				Status: sjapi.ScavengerJobStatusTypeRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sj2",
				Labels: map[string]string{
					CERIT_CPU: "2.0",
					CERIT_MEM: "2.0",
				},
			},
			Status: sjapi.ScavengerJobStatus{
				Status: sjapi.ScavengerJobStatusTypeRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sj3",
				Labels: map[string]string{
					CERIT_CPU: "1.0",
					CERIT_MEM: "2.0",
				},
			},
			Status: sjapi.ScavengerJobStatus{
				Status: sjapi.ScavengerJobStatusTypeRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sj4",
				Labels: map[string]string{
					CERIT_CPU: "0.0",
					CERIT_MEM: "2.0",
				},
			},
			Status: sjapi.ScavengerJobStatus{
				Status: sjapi.ScavengerJobStatusTypeRunning,
			},
		},
	},
}

func TestGetEvictionCandidate(t *testing.T) {
	sjapi.AddToScheme(scheme.Scheme)
	r := &NodeReconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(sjs).Build(),
		Log:    ctrl.Log.WithName("controllers").WithName("Node"),
		Scheme: scheme.Scheme,
	}
	sj, err := r.getEvictionCandidate(context.TODO())
	if err != nil {
		t.Fatalf("getEvictionCandidate failed: %v", err)
	}
	if sj == nil {
		t.Fatalf("Expected scavenger job, got nil")
	}

	if sj.ObjectMeta.Name != "sj1" {
		t.Fatalf("Expected sj1, got %v", sj.ObjectMeta.Name)
	}
}
