/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	sjapi "cerit.cz/scavenger-job/api/v1"
	"cerit.cz/scavenger-job/pkg/node"
)

// ScavengerJobReconciler reconciles a ScavengerJob object
type ScavengerJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var (
	THIRTY_SEC        = ctrl.Result{RequeueAfter: time.Second * 30}
	CHECKPOINT_TIME   = time.Minute * 3 //for tesing, in prod 30 min?
	GIGABYTE          = 1024 * 1024 * 1024.0
	CERIT_CPU         = "cerit.io/cputime"
	CERIT_MEM         = "cerit.io/memtime"
	CERIT_GPU         = "cerit.io/gputime"
	SJ_PRIORITY_CLASS = "scavenger-jobs-operator-priority"
)

//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

func (r *ScavengerJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.FromContext(ctx)

	scavengerJob := &sjapi.ScavengerJob{}
	if err := r.Get(ctx, req.NamespacedName, scavengerJob); err != nil {
		r.Log.Info("ScavengerJob has been deleted", "ScavengerJob", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	if scavengerJob.Status.StartTime == nil ||
		scavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeInterrupted {
		// ScavengerJob has not started yet or was interrupted, so create a new Job
		if err := r.handleJobCreation(ctx, scavengerJob, req); err != nil {
			r.Log.Error(err, "unable to create Job for ScavengerJob", "ScavengerJob", scavengerJob.Name)
		}

	} else if scavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeSucceeded ||
		scavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeFailed {
		// ScavengerJob has completed or is failing, so handle it
		r.handleCompletedJob(ctx, scavengerJob)
		return ctrl.Result{}, nil
	} else {
		// ScavengerJob is running
		r.handleStatusUpdate(ctx, scavengerJob)
		r.handleLabelUpdate(ctx, scavengerJob)
	}
	return THIRTY_SEC, nil
}

// handleJobCreation creates a Job for the ScavengerJob
func (r *ScavengerJobReconciler) handleJobCreation(ctx context.Context, scavengerJob *sjapi.ScavengerJob, req ctrl.Request) error {
	isBusy, _ := node.IsClusterBusy(ctx, r.Client, r.Log)
	if isBusy {
		// Cluster is busy, don't create new jobs
		return nil
	}
	r.Log.Info("Creating job for SJ", "ScavengerJob", scavengerJob.Name)

	job := &batch.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scavengerJob.Name,
			Namespace: scavengerJob.Namespace,
		},
		Spec: scavengerJob.Spec.Job,
	}
	job.Spec.Template.Spec.PriorityClassName = SJ_PRIORITY_CLASS

	if err := ctrl.SetControllerReference(scavengerJob, job, r.Scheme); err != nil {
		r.Log.Error(err, "unable to set controller reference for Job", "job", job)
		return err
	}
	if err := r.Create(ctx, job); err != nil {
		r.Log.Error(err, "unable to create Job for ScavengerJob", "job", scavengerJob.Spec.Job)
		return err
	}
	// Wait until the job has started
	wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 3*time.Second, false, func(ctx context.Context) (bool, error) {
		if err := r.Get(ctx, req.NamespacedName, job); err != nil {
			r.Log.Error(err, "unable to get created job", "job", job)
			return false, err
		}
		if job.Status.StartTime != nil {
			return true, nil
		}
		return false, nil
	})
	// Update scavenger job status
	if scavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeInterrupted {
		r.Log.Info("ScavengerJob is progressing again", "ScavengerJob", scavengerJob.Name)
	} else {
		scavengerJob.Status.StartTime = job.Status.StartTime
	}
	scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeRunning
	scavengerJob.Status.RunningTimeStamps = append(scavengerJob.Status.RunningTimeStamps, *job.Status.StartTime)
	if err := r.Status().Update(ctx, scavengerJob); err != nil {
		return err
	}
	return nil
}

// handleStatusUpdate updates the status of the ScavengerJob
func (r *ScavengerJobReconciler) handleStatusUpdate(ctx context.Context, scavengerJob *sjapi.ScavengerJob) {
	isChange := false
	status, err := r.propagatePodStatus(ctx, scavengerJob)
	if err != nil {
		r.Log.Error(err, "unable to propagate pod status", "ScavengerJob", scavengerJob.Name)
	}
	if status != scavengerJob.Status.Status {
		scavengerJob.Status.Status = status
		isChange = true
	}
	time := checkpointTime(scavengerJob)
	if time != nil {
		scavengerJob.Status.LastCheckpoint = time
		isChange = true
	}
	if isChange {
		if err := r.Status().Update(ctx, scavengerJob); err != nil {
			r.Log.Error(err, "unable to update ScavengerJob status", "ScavengerJob", scavengerJob.Name)
		}
	}
}

// propagatePodStatus propagates the status of the pod to the ScavengerJob
func (r *ScavengerJobReconciler) propagatePodStatus(ctx context.Context, scavengerJob *sjapi.ScavengerJob) (sjapi.ScavengerJobStatusType, error) {
	pods := &corev1.PodList{}
	err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, false, func(ctx context.Context) (bool, error) {
		if err := r.List(ctx, pods, client.MatchingLabels{"job-name": scavengerJob.Name}); err != nil {
			r.Log.Error(err, "unable to list pods", "pods", pods)
			return false, err
		}
		if len(pods.Items) != 1 {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		r.Log.Error(err, "No pods or more than 1 pod found for ScavengerJob", "ScavengerJob", scavengerJob.Name)
		return sjapi.ScavengerJobStatusTypeFailed, err
	}
	pod := pods.Items[0]
	var statusChange sjapi.ScavengerJobStatusType
	switch pod.Status.Phase {
	case "Pending":
		statusChange = sjapi.ScavengerJobStatusTypePending
	case "Running":
		statusChange = sjapi.ScavengerJobStatusTypeRunning
	case "Succeeded":
		statusChange = sjapi.ScavengerJobStatusTypeSucceeded
		now := metav1.NewTime(time.Now())
		scavengerJob.Status.CompletionTimeStamp = &now
	case "Failed":
		statusChange = sjapi.ScavengerJobStatusTypeFailed
	}
	return statusChange, nil
}

// handleLabelUpdate updates the labels of the ScavengerJob with the current resource requests multiplied by the time it has been running
func (r *ScavengerJobReconciler) handleLabelUpdate(ctx context.Context, scavengerJob *sjapi.ScavengerJob) {
	if scavengerJob.Status.Status != sjapi.ScavengerJobStatusTypeRunning {
		// ScavengerJob is pending and not fully running yet
		return
	}
	cpuReqxSec, memReqxSec, gpuReqxSec := getSJRequestsSeconds(scavengerJob)

	if scavengerJob.Labels == nil {
		scavengerJob.Labels = map[string]string{}
	}
	scavengerJob.Labels[CERIT_CPU] = strconv.FormatFloat(cpuReqxSec, 'f', 2, 64)
	scavengerJob.Labels[CERIT_MEM] = strconv.FormatFloat(memReqxSec, 'f', 2, 64)
	scavengerJob.Labels[CERIT_GPU] = strconv.FormatFloat(gpuReqxSec, 'f', 2, 64)

	if err := r.Update(ctx, scavengerJob); err != nil {
		r.Log.Error(err, "unable to update ScavengerJob labels", "ScavengerJob", scavengerJob.Name)
	}
}

// getSJRequestsSeconds calculates the resource requests of the ScavengerJob multiplied by the time it has been running
func getSJRequestsSeconds(scavengerJob *sjapi.ScavengerJob) (cpuReq, memReq, gpuReq float64) {
	sec := calculateRunTime(scavengerJob)
	for _, container := range scavengerJob.Spec.Job.Template.Spec.Containers {
		cpuf, _ := container.Resources.Requests.Cpu().AsDec().UnscaledBig().Float64()
		cpuReq += cpuf / 1000
		memf, _ := container.Resources.Requests.Memory().AsDec().UnscaledBig().Float64()
		memReq += memf / GIGABYTE
		// TODO add GPU compatibilty
		quantity, e := container.Resources.Requests[node.NvidiaGPU]
		if !e {
			gpuReq += float64(quantity.Value())
		}
	}
	return cpuReq * float64(sec), memReq * float64(sec), gpuReq * float64(sec)
}

// calculateRunTime calculates the time the ScavengerJob has been running since the last checkpoint/interruption
func calculateRunTime(scavengerJob *sjapi.ScavengerJob) int {
	now := time.Now()
	duration := time.Duration(0)
	lastRun := scavengerJob.Status.RunningTimeStamps[len(scavengerJob.Status.RunningTimeStamps)-1].Time
	duration = now.Sub(lastRun)
	if scavengerJob.Status.LastCheckpoint != nil {
		if scavengerJob.Status.LastCheckpoint.Time.After(lastRun) {
			duration = now.Sub(scavengerJob.Status.LastCheckpoint.Time)
		}
	}
	return int(duration.Seconds())
}

// checkpointTime checks if the ScavengerJob has reached the checkpoint time and updates the LastCheckpoint field
func checkpointTime(scavengerJob *sjapi.ScavengerJob) *metav1.Time {
	lastTime := scavengerJob.Status.RunningTimeStamps[len(scavengerJob.Status.RunningTimeStamps)-1]
	if isCheckpointTime(scavengerJob) {
		if scavengerJob.Status.LastCheckpoint != nil {
			lastTime = *scavengerJob.Status.LastCheckpoint
		}
		time := metav1.NewTime(lastTime.Time.Add(CHECKPOINT_TIME))
		return &time
	}
	return nil
}

// isCheckpointTime checks if the ScavengerJob has reached the checkpoint time
func isCheckpointTime(scavengerJob *sjapi.ScavengerJob) bool {
	now := metav1.NewTime(time.Now())
	lastRunTS := scavengerJob.Status.RunningTimeStamps[len(scavengerJob.Status.RunningTimeStamps)-1]
	if scavengerJob.Status.LastCheckpoint != nil {
		if now.Time.Sub(scavengerJob.Status.LastCheckpoint.Time) >= CHECKPOINT_TIME {
			return true
		}
	} else if now.Time.Sub(lastRunTS.Time) >= CHECKPOINT_TIME {
		if len(scavengerJob.Status.InterruptionTimeStamps) == 0 {
			return true
		}
		lastIntTs := scavengerJob.Status.InterruptionTimeStamps[len(scavengerJob.Status.InterruptionTimeStamps)-1]
		if lastRunTS.Time.After(lastIntTs.Time) {
			return true
		}
	}
	return false
}

// handleCompletedJob handles the ScavengerJob that has completed
func (r *ScavengerJobReconciler) handleCompletedJob(ctx context.Context, scavengerJob *sjapi.ScavengerJob) error {
	if scavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeFailed {
		r.Log.Info("ScavengerJob has failed", "ScavengerJob", scavengerJob.Name)
	} else {
		r.Log.Info("ScavengerJob has been completed", "ScavengerJob", scavengerJob.Name)
	}
	if err := DeleteSJjob(ctx, r.Client, r.Log, scavengerJob); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScavengerJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sjapi.ScavengerJob{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
