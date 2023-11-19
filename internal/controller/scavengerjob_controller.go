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

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	core "cerit.cz/scavenger-job/api/v1"
)

// ScavengerJobReconciler reconciles a ScavengerJob object
type ScavengerJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;
//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.cerit.cz,resources=scavengerjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ScavengerJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ScavengerJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)
	var scavengerJob = core.ScavengerJob{}
	if err := r.Get(ctx, req.NamespacedName, &scavengerJob); err != nil {
		log.Error(err, "unable to fetch ScavengerJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if scavengerJob.Spec.Job.Template.Spec.Containers == nil {
		log.Info("Wrong Job Spec, skipping")
		return ctrl.Result{}, nil
	}
	if scavengerJob.Status.Job.StartTime == nil {
		log.Info("Job not found, creating one")
		var job = batch.Job{
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
		if err := r.Create(ctx, &job); err != nil {
			log.Error(err, "unable to create Job for ScavengerJob", "job", scavengerJob.Spec.Job)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScavengerJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&core.ScavengerJob{}).
		Complete(r)
}
