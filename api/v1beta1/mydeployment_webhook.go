/*
Copyright 2025.

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

package v1beta1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var mydeploymentlog = logf.Log.WithName("mydeployment-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *MyDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-kubebuilder-zq-com-v1beta1-mydeployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=kubebuilder.zq.com,resources=mydeployments,verbs=create;update,versions=v1beta1,name=mmydeployment.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &MyDeployment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MyDeployment) Default() {
	mydeploymentlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	if r.Spec.Selector == nil {
		r.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: r.Spec.Template.ObjectMeta.Labels,
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-kubebuilder-zq-com-v1beta1-mydeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubebuilder.zq.com,resources=mydeployments,verbs=create;update,versions=v1beta1,name=vmydeployment.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MyDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MyDeployment) ValidateCreate() (admission.Warnings, error) {
	mydeploymentlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, r.validMyDeployment()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MyDeployment) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	mydeploymentlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, r.validMyDeployment()
}

func (r *MyDeployment) validMyDeployment() error {
	if r.Spec.Template.ObjectMeta.Labels == nil {
		return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, field.ErrorList{
			field.Invalid(field.NewPath(r.Name), r.Spec.Template.ObjectMeta.Labels, ".spec.template.metadata.labels is required"),
		})
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MyDeployment) ValidateDelete() (admission.Warnings, error) {
	mydeploymentlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
