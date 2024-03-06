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

// log is for logging in this package.
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

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-core-cerit-cz-v1-scavengerjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=core.cerit.cz,resources=scavengerjobs,verbs=create;update,versions=v1,name=mscavengerjob.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ScavengerJob{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ScavengerJob) Default() {
	log.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-core-cerit-cz-v1-scavengerjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.cerit.cz,resources=scavengerjobs,verbs=create;update;delete,versions=v1,name=vscavengerjob.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ScavengerJob{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ScavengerJob) ValidateCreate() (admission.Warnings, error) {
	log.Info("validate create", "name", r.Name)

	if busy, _ := node.IsClusterBusy(context.Background(), wClient, log); busy {
		return admission.Warnings{"SJ not created"}, fmt.Errorf("cluster is busy, not creating new scavneger jobs")
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ScavengerJob) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	log.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ScavengerJob) ValidateDelete() (admission.Warnings, error) {
	log.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
