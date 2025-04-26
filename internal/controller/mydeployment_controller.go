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

package controller

import (
	"context"
	"fmt"
	"github.com/hdssbks/mydeployment/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubebuilderv1beta1 "github.com/hdssbks/mydeployment/api/v1beta1"
)

// MyDeploymentReconciler reconciles a MyDeployment object
type MyDeploymentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=kubebuilder.zq.com,resources=mydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubebuilder.zq.com,resources=mydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubebuilder.zq.com,resources=mydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *MyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	mydeploy := &kubebuilderv1beta1.MyDeployment{}
	if err := r.Get(ctx, req.NamespacedName, mydeploy); err != nil {
		logger.Error(err, fmt.Sprintf("unable to fetch MyDeployment: %s", req.NamespacedName))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 如果deployment暂停，就不会创建删除pod，直接返回
	if mydeploy.Spec.Paused {
		return ctrl.Result{}, nil
	}

	// 通过标签找到被deployment管理的pod
	pods := &corev1.PodList{}
	listOpts := client.ListOptions{
		Namespace:     mydeploy.Namespace,
		LabelSelector: labels.SelectorFromValidatedSet(mydeploy.Spec.Selector.MatchLabels),
	}
	err := r.List(ctx, pods, &listOpts)
	if err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "GetPods", err.Error())
		return ctrl.Result{}, err
	}

	// 删除中的Pod不参与计算
	unTerminatedPods := make([]corev1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() {
			unTerminatedPods = append(unTerminatedPods, pod)
		}
	}

	// 比较mydeploy.Spec.Replicas和unTerminatedPods中的数量
	diff := int(*mydeploy.Spec.Replicas) - len(unTerminatedPods)
	switch {
	// spec等于实际，直接返回
	case diff == 0:
		logger.Info(fmt.Sprintf("no pods found for MyDeployment: %s", mydeploy.Name))
		return ctrl.Result{}, nil

	// spec小于实际，删除pod
	case diff < 0:
		r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Diff Pods", "more than spec, delete pods")
		// delete pods
		for i := 0; i < 0-diff; i++ {
			logger.Info(fmt.Sprintf("deleting pod: %s", unTerminatedPods[i].Name))
			if err := r.Delete(ctx, &unTerminatedPods[i]); err != nil {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "DeletePod", err.Error())
				return ctrl.Result{}, err
			}
			r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "DeletePod", fmt.Sprintf("delete pod %s", unTerminatedPods[i].Name))
		}

	// spec大于实际，创建pod
	default:
		r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Diff Pods", "less than spec, create pods")
		// create pods
		for i := 0; i < diff; i++ {
			pod := utils.NewPod(mydeploy)
			if err := r.Create(ctx, pod); err != nil {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "CreatePod", err.Error())
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubebuilderv1beta1.MyDeployment{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
