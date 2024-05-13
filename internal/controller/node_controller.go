package controller

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sjapi "cerit.cz/scavenger-job/api/v1"
	sjnode "cerit.cz/scavenger-job/pkg/node"
)

type NodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sj.cerit.cz,resources=scavengerjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sj.cerit.cz,resources=scavengerjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sj.cerit.cz,resources=scavengerjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.FromContext(ctx)

	_, shouldKill := sjnode.IsClusterBusy(ctx, r.Client, r.Log)
	if shouldKill {
		r.Log.Info("Node is busy, evicting running jobs")
		if err := r.evictSJ(ctx); err != nil {
			r.Log.Error(err, "unable to delete Job candidate")

		}
	}

	return THIRTY_SEC, nil
}

// evictSJ evicts the ScavengerJobs from the node with the lowest priority
// by deleting it's corresponding Job and updating the ScavengerJob status
func (r *NodeReconciler) evictSJ(ctx context.Context) error {
	candidateScavengerJob, err := r.getEvictionCandidate(ctx)
	if err != nil {
		return err
	}
	if candidateScavengerJob == nil {
		r.Log.Info("No running ScavengerJobs found, nothing to evict")
		return nil
	}
	r.Log.Info("Evicting lowest priority candidate", "ScavengerJob", candidateScavengerJob.Name)
	if err := DeleteSJjob(ctx, r.Client, r.Log, candidateScavengerJob); err != nil {
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

// getEvictionCandidate returns the ScavengerJob with the lowest priority
func (r *NodeReconciler) getEvictionCandidate(ctx context.Context) (*sjapi.ScavengerJob, error) {
	sjs := &sjapi.ScavengerJobList{}
	if err := r.List(ctx, sjs); err != nil {
		r.Log.Error(err, "unable to list ScavengerJobs")
		return nil, err
	}
	if len(sjs.Items) == 0 {
		return nil, nil
	}

	lowestUsage := math.MaxFloat64
	var candidateSJ *sjapi.ScavengerJob
	for _, sj := range sjs.Items {
		if sj.Status.Status == sjapi.ScavengerJobStatusTypeRunning {
			cpuInt, _ := strconv.ParseFloat(sj.Labels[CERIT_CPU], 64)
			memInt, _ := strconv.ParseFloat(sj.Labels[CERIT_MEM], 64)
			gpuInt, _ := strconv.ParseFloat(sj.Labels[CERIT_GPU], 64)
			usage := cpuInt + memInt + gpuInt
			if usage < lowestUsage {
				lowestUsage = usage
				candidateSJ = sj.DeepCopy()
			}
		}
	}

	return candidateSJ, nil
}

// DeleteSJjob deletes the Job corresponding to the ScavengerJob
func DeleteSJjob(ctx context.Context, cl client.Client, log logr.Logger, candidateScavengerJob *sjapi.ScavengerJob) error {
	jobToBeDeleted := &batch.Job{}
	if err := cl.Get(ctx, client.ObjectKey{Namespace: candidateScavengerJob.Namespace, Name: candidateScavengerJob.Name}, jobToBeDeleted); err != nil {
		log.Info("unable to get Job", "job", candidateScavengerJob.Name)
		return err
	}
	deletePolicy := metav1.DeletePropagationBackground
	zero := int64(0)
	deleteOptions := client.DeleteOptions{
		PropagationPolicy:  &deletePolicy,
		GracePeriodSeconds: &zero,
	}
	log.Info("Deleting job", "ScavengerJob", candidateScavengerJob.Name)
	if err := cl.Delete(ctx, jobToBeDeleted, &deleteOptions); err != nil {
		log.Error(err, "unable to delete Job", "job", candidateScavengerJob.Name)
		return err
	}
	return nil
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}
