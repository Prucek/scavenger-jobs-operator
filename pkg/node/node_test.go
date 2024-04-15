package node

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUpdatePreviousUsage(t *testing.T) {
	testcases := []struct {
		name                  string
		fakeClient            client.Client
		previousUsage         []float64
		currentUsage          float64
		expectedPreviousUsage []float64
	}{
		{
			name:                  "Simple",
			previousUsage:         []float64{.05, .10},
			currentUsage:          .2,
			expectedPreviousUsage: []float64{.05, .10, .2},
		},
		{
			name:                  "Empty",
			previousUsage:         []float64{},
			currentUsage:          .2,
			expectedPreviousUsage: []float64{0.2},
		},
		{
			name:                  "Full",
			previousUsage:         []float64{.05, .10, .12, .2, .3, .4, .34, .41, .5},
			currentUsage:          .2,
			expectedPreviousUsage: []float64{.10, .12, .2, .3, .4, .34, .41, .5, .2},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			isResourceFull("CPU", &tc.previousUsage, tc.currentUsage, logr.Logger{})
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
