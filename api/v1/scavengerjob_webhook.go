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

package v1

import (
	"context"
	"fmt"

	"cerit.cz/scavenger-job/pkg/node"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	log     = logf.Log.WithName("scavengerjob-resource")
	wClient client.Client
)

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ScavengerJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	wClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-core-cerit-cz-v1-scavengerjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.cerit.cz,resources=scavengerjobs,verbs=create;update;delete,versions=v1,name=vscavengerjob.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ScavengerJob{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ScavengerJob) ValidateCreate() (admission.Warnings, error) {
	log.Info("validate create", "name", r.Name)
	if len(r.Spec.Job.Template.Spec.Containers) == 0 {
		return admission.Warnings{"SJ not created"}, fmt.Errorf("ScavengerJob must have at least one container")
	}
	for _, container := range r.Spec.Job.Template.Spec.Containers {
		if container.Resources.Requests == nil {
			return admission.Warnings{"SJ not created"}, fmt.Errorf("container %s does not have any Requests", container.Name)
		}
	}
	if busy, _ := node.IsClusterBusy(context.Background(), wClient, log); busy {
		return admission.Warnings{"SJ not created"}, fmt.Errorf("cluster is busy, not creating new ScavengerJob")
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ScavengerJob) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ScavengerJob) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
