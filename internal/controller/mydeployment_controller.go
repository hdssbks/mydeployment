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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
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
//+kubebuilder:rbac:groups=apps,resources=controllerrevisions,verbs=get;list;watch;create;update;patch;delete

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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 如果deployment暂停，就不会创建删除pod，直接返回
	if mydeploy.Spec.Paused {
		return ctrl.Result{}, nil
	}

	// 查找最新的controllerrevision，
	listOpts := client.ListOptions{
		Namespace:     mydeploy.Namespace,
		LabelSelector: labels.SelectorFromValidatedSet(mydeploy.Spec.Selector.MatchLabels),
	}

	cr, err := r.getMaxRevision(ctx, mydeploy, listOpts)
	if err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "GetControllerRevision", err.Error())
		return ctrl.Result{}, err
	}

	// 如果cr为空，则创建cr
	if cr == nil {
		cr, err = r.createControllerRevision(ctx, mydeploy, 1)
		if err != nil {
			r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "CreateControllerRevision", err.Error())
			return ctrl.Result{}, err
		}
	}

	// 通过标签找到被deployment管理的pod
	pods := &corev1.PodList{}
	err = r.List(ctx, pods, &listOpts)
	if err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "GetPods", err.Error())
		return ctrl.Result{}, err
	}

	// 如果cr不为空，获取cr中的object
	obj := &kubebuilderv1beta1.MyDeployment{}
	if err := json.Unmarshal(cr.Data.Raw, obj); err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "Unmarshal", err.Error())
		return ctrl.Result{}, err
	}
	// 比较object中的Spec和mydeploy的Spec，如果不相同，则创建新的cr，并删除所有pod
	if !reflect.DeepEqual(obj.Spec.Template.Spec, mydeploy.Spec.Template.Spec) {
		if _, err := r.createControllerRevision(ctx, mydeploy, cr.Revision+1); err != nil {
			r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "CreateControllerRevision", err.Error())
			return ctrl.Result{}, err
		}
		for _, pod := range pods.Items {
			if err := r.Delete(ctx, &pod); err != nil {
				r.Recorder.Event(&pod, corev1.EventTypeWarning, "Delete Pod", "Delete Pod Failed")
			}
		}
		if err := r.updateStatus(ctx, mydeploy, &listOpts, cr); err != nil {
			if apierrors.IsConflict(err) {
				r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Conflict Requeue", err.Error())
				return ctrl.Result{Requeue: true}, nil
			} else {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "UpdateStatus", err.Error())
				return ctrl.Result{}, err
			}
		}
		//return ctrl.Result{}, nil
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
		if err := r.updateStatus(ctx, mydeploy, &listOpts, cr); err != nil {
			if apierrors.IsConflict(err) {
				r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Conflict Requeue", err.Error())
				return ctrl.Result{Requeue: true}, nil
			} else {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "UpdateStatus", err.Error())
				return ctrl.Result{}, err
			}
		}
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
		if err := r.updateStatus(ctx, mydeploy, &listOpts, cr); err != nil {
			if apierrors.IsConflict(err) {
				r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Conflict Requeue", err.Error())
				return ctrl.Result{Requeue: true}, nil
			} else {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "UpdateStatus", err.Error())
				return ctrl.Result{}, err
			}
		}

	// spec大于实际，创建pod
	default:
		r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Diff Pods", "less than spec, create pods")
		// create pods
		for i := 0; i < diff; i++ {
			pod := utils.NewPod(mydeploy)
			err := ctrl.SetControllerReference(mydeploy, pod, r.Scheme)
			if err != nil {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "SetControllerReference", err.Error())
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, pod); err != nil {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "CreatePod", err.Error())
				return ctrl.Result{}, err
			}
		}
		if err := r.updateStatus(ctx, mydeploy, &listOpts, cr); err != nil {
			if apierrors.IsConflict(err) {
				r.Recorder.Event(mydeploy, corev1.EventTypeNormal, "Conflict Requeue", err.Error())
				return ctrl.Result{Requeue: true}, nil
			} else {
				r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "UpdateStatus", err.Error())
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *MyDeploymentReconciler) updateStatus(ctx context.Context, deploy *kubebuilderv1beta1.MyDeployment, listOpts *client.ListOptions, cr *appsv1.ControllerRevision) error {
	logger := log.FromContext(ctx)

	// 以matchLables查找mydeploy关联的pod
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, listOpts); err != nil {
		logger.Error(err, "can not list pods")
		return err
	}

	// 在podlist中查找状态为ready的pod
	readyPods := int32(0)
	for _, pod := range pods.Items {
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				readyPods++
			}
		}
	}

	deploy.Status.CurrentReplicas = int32(len(pods.Items))
	deploy.Status.Replicas = *deploy.Spec.Replicas
	deploy.Status.AvailableReplicas = readyPods
	deploy.Status.ReadyReplicas = readyPods
	deploy.Status.CurrentRevision = cr.Name

	if err := r.Status().Update(ctx, deploy); err != nil {
		//r.Recorder.Event(deploy, corev1.EventTypeWarning, "Status Update", err.Error())
		return err
	}
	return nil
}

func (r *MyDeploymentReconciler) getMaxRevision(ctx context.Context, mydeploy *kubebuilderv1beta1.MyDeployment, listOpts client.ListOptions) (*appsv1.ControllerRevision, error) {
	crs := &appsv1.ControllerRevisionList{}
	if err := r.List(ctx, crs, &listOpts); err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "ListControllerRevision", err.Error())
		return nil, err
	}
	if len(crs.Items) == 0 {
		return nil, nil
	}
	maxRevision := int64(0)
	index := 0
	for i, cr := range crs.Items {
		if cr.Revision > maxRevision {
			maxRevision = cr.Revision
			index = i
		}
	}
	return &crs.Items[index], nil
}

func (r *MyDeploymentReconciler) createControllerRevision(ctx context.Context, mydeploy *kubebuilderv1beta1.MyDeployment, revision int64) (*appsv1.ControllerRevision, error) {
	//bytesDeploy := make([]byte, 0)
	bytesDeploy, err := json.Marshal(mydeploy)
	if err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "Unmarshal", err.Error())
		panic(err)
	}

	cr := &appsv1.ControllerRevision{
		ObjectMeta: ctrl.ObjectMeta{
			Namespace: mydeploy.Namespace,
			Name:      mydeploy.Name + "-" + utils.RandStr(5),
			Labels:    mydeploy.Spec.Selector.MatchLabels,
		},
		Data: runtime.RawExtension{
			Raw:    bytesDeploy,
			Object: mydeploy,
		},
		Revision: revision,
	}

	if err := ctrl.SetControllerReference(mydeploy, cr, r.Scheme); err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "SetControllerReference", err.Error())
		return nil, err
	}
	if err := r.Create(ctx, cr); err != nil {
		r.Recorder.Event(mydeploy, corev1.EventTypeWarning, "CreateControllerRevision", err.Error())
		return nil, err
	}
	return cr, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubebuilderv1beta1.MyDeployment{}).
		Owns(&corev1.Pod{}).
		Owns(&appsv1.ControllerRevision{}).
		Complete(r)
}
