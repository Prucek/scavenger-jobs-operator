package controller

import (
	"testing"
	"time"

	sjapi "cerit.cz/scavenger-job/api/v1"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	CONST_TIME = time.Date(2000, time.August, 23, 6, 0, 0, 0, time.UTC)
)

func TestGetSJRequestsSeconds(t *testing.T) {
	testcases := []struct {
		name           string
		sj             *sjapi.ScavengerJob
		cpuReq, memReq float64
	}{
		{
			name: "Simple",
			sj: &sjapi.ScavengerJob{
				Spec: sjapi.ScavengerJobSpec{
					Job: batch.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("1"),
												corev1.ResourceMemory: resource.MustParse("1Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
					},
				},
			},
			cpuReq: 1.8,
			memReq: 1800,
		},
		{
			name: "Simple 2",
			sj: &sjapi.ScavengerJob{
				Spec: sjapi.ScavengerJobSpec{
					Job: batch.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("250m"),
												corev1.ResourceMemory: resource.MustParse("64Mi"),
											},
										},
									},
								},
							},
						},
					},
				},
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
					},
				},
			},
			cpuReq: 0.45,
			memReq: 112.5,
		},
		{
			name: "Complex",
			sj: &sjapi.ScavengerJob{
				Spec: sjapi.ScavengerJobSpec{
					Job: batch.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("250m"),
												corev1.ResourceMemory: resource.MustParse("128Mi"),
											},
										},
									},
									{
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("1"),
												corev1.ResourceMemory: resource.MustParse("1Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
						{Time: time.Now().Add(-10 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-20 * time.Minute)},
					},
				},
			},
			cpuReq: 0.75,
			memReq: 675,
		},

		{
			name: "Complex with Checkpoint",
			sj: &sjapi.ScavengerJob{
				Spec: sjapi.ScavengerJobSpec{
					Job: batch.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("250m"),
												corev1.ResourceMemory: resource.MustParse("128Mi"),
											},
										},
									},
									{
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("1"),
												corev1.ResourceMemory: resource.MustParse("1Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
						{Time: time.Now().Add(-10 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-20 * time.Minute)},
					},
					LastCheckpoint: &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
				},
			},
			cpuReq: 0.375,
			memReq: 337.5,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cpuReq, memReq, _ := getSJRequestsSeconds(tc.sj)
			if cpuReq != tc.cpuReq {
				t.Errorf("Expected cpuReq %v, but got %v", tc.cpuReq, cpuReq)
			}
			if memReq != tc.memReq {
				t.Errorf("Expected memReq %v, but got %v", tc.memReq, memReq)
			}
		})
	}
}

func TestCalculateRunTime(t *testing.T) {
	testcases := []struct {
		name    string
		sj      *sjapi.ScavengerJob
		runTime int
	}{
		{
			name: "Simple",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
					},
				},
			},
			runTime: 30 * 60,
		},
		{
			name: "With InterruptionTimeStamps",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
						{Time: time.Now().Add(-10 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-20 * time.Minute)},
					},
				},
			},
			runTime: 10 * 60,
		},
		{
			name: "Not running",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-20 * time.Minute)},
					},
				},
			},
			runTime: 0,
		},
		{
			name: "Checkpointed",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{},
					LastCheckpoint:         &metav1.Time{Time: time.Now().Add(-20 * time.Minute)},
				},
			},
			runTime: 20 * 60,
		},
		{
			name: "Complex",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
						{Time: time.Now().Add(-10 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-20 * time.Minute)},
					},
					LastCheckpoint: &metav1.Time{Time: time.Now().Add(-15 * time.Minute)},
				},
			},
			runTime: 10 * 60,
		},
		{
			name: "Complex 2",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
						{Time: time.Now().Add(-10 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-20 * time.Minute)},
					},
					LastCheckpoint: &metav1.Time{Time: time.Now().Add(-7 * time.Minute)},
				},
			},
			runTime: 7 * 60,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			runTime := calculateRunTime(tc.sj)
			if runTime != tc.runTime {
				t.Errorf("Expected runTime %v, but got %v", tc.runTime, runTime)
			}
		})
	}
}

func TestCheckpointTime(t *testing.T) {
	testcases := []struct {
		name                   string
		sj                     *sjapi.ScavengerJob
		expectedCheckpointTime *metav1.Time
	}{
		{
			name: "Empty InterruptionTimeStamps",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-60 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{},
				},
			},
			expectedCheckpointTime: &metav1.Time{Time: CONST_TIME.Add(-60 * time.Minute).Add(CHECKPOINT_TIME)},
		},
		{
			name: "With InterruptionTimeStamps",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-60 * time.Minute)},
						{Time: CONST_TIME.Add(-30 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-50 * time.Minute)},
					},
				},
			},
			expectedCheckpointTime: &metav1.Time{Time: CONST_TIME.Add(-30 * time.Minute).Add(CHECKPOINT_TIME)},
		},
		{
			name: "Not running",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-60 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-50 * time.Minute)},
					},
				},
			},
			expectedCheckpointTime: nil,
		},
		{
			name: "With LastCheckpoint",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-60 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{
						{Time: CONST_TIME.Add(-50 * time.Minute)},
					},
					LastCheckpoint: &metav1.Time{Time: CONST_TIME.Add(-30 * time.Minute)},
				},
			},
			expectedCheckpointTime: &metav1.Time{Time: CONST_TIME.Add(-30 * time.Minute).Add(CHECKPOINT_TIME)},
		},
		{
			name: "Not checkpoint time",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now()},
					},
					InterruptionTimeStamps: []metav1.Time{},
				},
			},
			expectedCheckpointTime: nil,
		},
		{
			name: "Not checkpoint time 2",
			sj: &sjapi.ScavengerJob{
				Status: sjapi.ScavengerJobStatus{
					RunningTimeStamps: []metav1.Time{
						{Time: time.Now().Add(-30 * time.Minute)},
					},
					InterruptionTimeStamps: []metav1.Time{},
					LastCheckpoint:         &metav1.Time{Time: time.Now()},
				},
			},
			expectedCheckpointTime: nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkpointTime := checkpointTime(tc.sj)
			if !checkpointTime.Equal(tc.expectedCheckpointTime) {
				t.Errorf("Expected checkpointTime %v, but got %v", tc.expectedCheckpointTime, checkpointTime)
			}
		})
	}

}
