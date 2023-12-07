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

	sjapi "cerit.cz/scavenger-job/api/v1"
)

// ScavengerJobReconciler reconciles a ScavengerJob object
type ScavengerJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;delete

func (r *ScavengerJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.Log = log.FromContext(ctx)

	if r.isClusterBusy(ctx) {
		r.Log.Info("Cluster is busy, not creating new jobs")
		if err := r.deleteJobCandidates(ctx); err != nil {
			r.Log.Error(err, "unable to delete Job candidates")
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	scavengerJob := &sjapi.ScavengerJob{}
	if err := r.Get(ctx, req.NamespacedName, scavengerJob); err != nil {
		r.Log.Error(err, "unable to fetch ScavengerJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	deepCopiedScavengerJob := scavengerJob.DeepCopy()
	if deepCopiedScavengerJob.Spec.Job.Template.Spec.Containers == nil {
		r.Log.Info("Wrong Job Spec, skipping")
		return ctrl.Result{}, nil
	}
	if deepCopiedScavengerJob.Status.StartTime == nil ||
		deepCopiedScavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeInterrupted {
		r.Log.Info("Job for Scavenger Job not found, creating one")
		return r.createJob(ctx, deepCopiedScavengerJob, req)

	} else {
		r.Log.Info("Job for Scavenger Job found, updating status")
		return r.updateStatus(ctx, deepCopiedScavengerJob)
	}
}

func (r *ScavengerJobReconciler) createJob(ctx context.Context, scavengerJob *sjapi.ScavengerJob, req ctrl.Request) (ctrl.Result, error) {
	if r.wouldClusterBeBusy(ctx) {
		r.Log.Info("Cluster would be busy, not creating new jobs")
		return ctrl.Result{}, nil
	}

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

	if err := ctrl.SetControllerReference(scavengerJob, job, r.Scheme); err != nil {
		r.Log.Error(err, "unable to set controller reference for Job", "job", job)
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, job); err != nil {
		r.Log.Error(err, "unable to create Job for ScavengerJob", "job", scavengerJob.Spec.Job)
		return ctrl.Result{}, err
	}

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
	if scavengerJob.Status.Status == sjapi.ScavengerJobStatusTypeInterrupted {
		r.Log.Info("ScavengerJob was interrupted, recreating job")
		scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeRunning
	} else {
		scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeNotStarted
		scavengerJob.Status.StartTime = job.Status.StartTime
	}
	scavengerJob.Status.RunningTimeStamps = append(scavengerJob.Status.RunningTimeStamps, *job.Status.StartTime)
	if err := r.Status().Update(ctx, scavengerJob); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func (r *ScavengerJobReconciler) updateStatus(ctx context.Context, scavengerJob *sjapi.ScavengerJob) (ctrl.Result, error) {
	pods := &corev1.PodList{}

	if err := r.List(ctx, pods, client.MatchingLabels{"job-name": scavengerJob.Name}); err != nil {
		r.Log.Error(err, "unable to list pods", "pods", pods)
		return ctrl.Result{}, err
	}
	if len(pods.Items) == 0 {
		r.Log.Info("No pods found, should't happen")
		// jobs was probably deleted, we have to restart the job?
		return ctrl.Result{}, nil
	}
	if len(pods.Items) > 1 {
		r.Log.Info("More than one pod found, should't happen")
		return ctrl.Result{}, nil
	}
	pod := pods.Items[0]
	switch pod.Status.Phase {
	case "Pending":
		scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypePending
	case "Running":
		scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeRunning
	case "Succeeded":
		scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeCompleted
		now := metav1.NewTime(time.Now())
		scavengerJob.Status.CompletionTimeStamp = &now
	case "Failed":
		scavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeFailing
	}
	if err := r.Status().Update(ctx, scavengerJob); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// TODO: retriece metrics and check if cluster is busy
func (r *ScavengerJobReconciler) isClusterBusy(ctx context.Context) bool {
	// DUMMY IMPLEMENTATION
	jobs := batch.JobList{}
	if err := r.List(ctx, &jobs); err != nil {
		r.Log.Error(err, "unable to list pods")
	}
	if len(jobs.Items) > 3 {
		return true
	}
	return false
}

func (r *ScavengerJobReconciler) wouldClusterBeBusy(ctx context.Context) bool {
	// DUMMY IMPLEMENTATION
	jobs := batch.JobList{}
	if err := r.List(ctx, &jobs); err != nil {
		r.Log.Error(err, "unable to list jobs")
	}
	if len(jobs.Items) > 2 {
		return true
	}
	return false
}

// TODO: add pod priority and preemption
func (r *ScavengerJobReconciler) deleteJobCandidates(ctx context.Context) error {
	// DUMMY IMPLEMENTATION
	sjs := &sjapi.ScavengerJobList{}
	if err := r.List(ctx, sjs); err != nil {
		r.Log.Error(err, "unable to list ScavengerJobs")
		return err
	}
	// get first name - TODO: get the one with the lowest priority
	if len(sjs.Items) == 0 {
		r.Log.Info("No ScavengerJobs found, nothing to delete")
		return nil
	}

	candidateScavengerJob := &sjapi.ScavengerJob{}
	for _, sj := range sjs.Items {
		if sj.Status.Status == sjapi.ScavengerJobStatusTypeRunning {
			candidateScavengerJob = sj.DeepCopy()
			break
		}
	}

	if candidateScavengerJob.Status.Status != sjapi.ScavengerJobStatusTypeRunning {
		r.Log.Info("No running ScavengerJobs found, nothing to delete")
		return nil
	}

	r.Log.Info("Deleting job of Scavenger Job", "ScavengerJob", candidateScavengerJob.Name)
	jobToBeDeleted := &batch.Job{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: candidateScavengerJob.Namespace, Name: candidateScavengerJob.Name},
		jobToBeDeleted); err != nil {
		r.Log.Error(err, "unable to get Job", "job", candidateScavengerJob.Name)
		return err
	}
	deepCopiedJobToBeDeleted := jobToBeDeleted.DeepCopy()
	deletePolicy := metav1.DeletePropagationBackground
	zero := int64(0)
	deleteOptions := client.DeleteOptions{
		PropagationPolicy:  &deletePolicy,
		GracePeriodSeconds: &zero,
	}
	if err := r.Delete(ctx, deepCopiedJobToBeDeleted, &deleteOptions); err != nil {
		r.Log.Error(err, "unable to delete Job", "job", candidateScavengerJob.Name)
		return err
	}
	// Update scavenger job status
	candidateScavengerJob.Status.Status = sjapi.ScavengerJobStatusTypeInterrupted
	candidateScavengerJob.Status.InterruptionTimeStamps = append(candidateScavengerJob.Status.InterruptionTimeStamps, metav1.NewTime(time.Now()))
	if err := r.Status().Update(ctx, candidateScavengerJob); err != nil {
		r.Log.Error(err, "unable to update ScavengerJob status", "ScavengerJob", candidateScavengerJob.Name)
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScavengerJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sjapi.ScavengerJob{}).
		Owns(&batch.Job{}).
		Complete(r)
}
