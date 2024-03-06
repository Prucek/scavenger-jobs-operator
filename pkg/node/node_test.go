package node

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpdatePreviousUsage(t *testing.T) {

	fakeSeededClient := fakeclient.NewClientBuilder().WithRuntimeObjects(&corev1.Node{
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourcePods: *resource.NewQuantity(20, resource.DecimalSI),
			},
		},
	}).WithRuntimeObjects(&corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test3",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test4",
				},
			},
		},
	}).Build()

	fakeMultiNode := fakeclient.NewClientBuilder().WithRuntimeObjects(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourcePods: *resource.NewQuantity(20, resource.DecimalSI),
			},
		},
	}).WithRuntimeObjects(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node2",
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourcePods: *resource.NewQuantity(20, resource.DecimalSI),
			},
		},
	}).WithRuntimeObjects(&corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test3",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test4",
				},
			},
		},
	}).Build()

	testcases := []struct {
		name                  string
		fakeClient            client.Client
		previousUsage         []float64
		expectedPreviousUsage []float64
	}{
		{
			name:                  "Simple",
			fakeClient:            fakeSeededClient,
			previousUsage:         []float64{.05, .10},
			expectedPreviousUsage: []float64{.05, .10, .2},
		},
		{
			name:                  "Empty",
			fakeClient:            fakeSeededClient,
			previousUsage:         []float64{},
			expectedPreviousUsage: []float64{0.2},
		},
		{
			name:                  "Full",
			fakeClient:            fakeSeededClient,
			previousUsage:         []float64{.05, .10, .12, .2, .3, .4, .34, .41, .5},
			expectedPreviousUsage: []float64{.10, .12, .2, .3, .4, .34, .41, .5, .2},
		},
		{
			name:                  "Multinode",
			fakeClient:            fakeMultiNode,
			previousUsage:         []float64{.05, .10, .12, .2, .3, .4, .34, .41, .5},
			expectedPreviousUsage: []float64{.10, .12, .2, .3, .4, .34, .41, .5, .1},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			updatePreviousUsage(&tc.previousUsage, context.Background(), tc.fakeClient, logr.Logger{})
			if !reflect.DeepEqual(tc.previousUsage, tc.expectedPreviousUsage) {
				t.Errorf("Expected previousUsage %v, but got %v", tc.expectedPreviousUsage, tc.previousUsage)
			}
		})
	}
}

func TestGetThreshold(t *testing.T) {
	testcases := []struct {
		name                    string
		previousUsage           []float64
		expectedBottomThreshold float64
		expectedTopThreshold    float64
	}{
		{
			name:                    "Simple",
			previousUsage:           []float64{.05, .10},
			expectedBottomThreshold: 0.875,
			expectedTopThreshold:    0.9416666666666667,
		},
		{
			name:                    "Empty",
			previousUsage:           []float64{},
			expectedBottomThreshold: 0.9,
			expectedTopThreshold:    0.9666666666666667,
		},
		{
			name:                    "Full",
			previousUsage:           []float64{.05, .10, .12, .2, .3, .4, .34, .41, .5},
			expectedBottomThreshold: 0.7509702159190658,
			expectedTopThreshold:    0.8176368825857324,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bottom, top := getThresholds(tc.previousUsage, 0.1)
			if bottom != float64(tc.expectedBottomThreshold) {
				t.Errorf("Expected bottom threshold %v, but got %v", tc.expectedBottomThreshold, bottom)
			}
			if top != float64(tc.expectedTopThreshold) {
				t.Errorf("Expected top threshold %v, but got %v", tc.expectedTopThreshold, top)
			}
		})
	}
}
