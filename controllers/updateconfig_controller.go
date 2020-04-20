/*


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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	customappv1 "prasad.com/kubecontroller/api/v1"
)

// UpdateConfigReconciler reconciles a UpdateConfig object
type UpdateConfigReconciler struct {
	client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=customapp.prasad.com,resources=updateconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=customapp.prasad.com,resources=updateconfigs/status,verbs=get;update;patch
func (r *UpdateConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// _ = context.Background()
	// _ = r.Log.WithValues("updateconfig", req.NamespacedName)

	// Fetch the Presentation instance
	instance := &customappv1.UpdateConfig{}
	err := r.client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	configMapChanged, err := r.ensureLatestConfigMap(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), req.NamespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	if configMapChanged {
		err = r.client.Delete(context.TODO(), found)
		if err != nil {
			return ctrl.Result{}, err
		}
		// err = r.client.Create(context.TODO(), pod)
		// if err != nil {
		// 	return err
		// }
	}

	return ctrl.Result{}, nil
}

func (r *UpdateConfigReconciler) ensureLatestConfigMap(instance *customappv1.UpdateConfig) (bool, error) {
	configMap := newConfigMap(instance)

	// // Set Presentation instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, configMap, r.scheme); err != nil {
	// 	return false, err
	// }

	// Check if this ConfigMap already exists
	foundMap := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundMap)
	if err != nil && errors.IsNotFound(err) {
		err = r.client.Create(context.TODO(), configMap)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	}

	if foundMap.Data["redis-config"] != configMap.Data["redis-config"] {
		err = r.client.Update(context.TODO(), configMap)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func newConfigMap(cr *customappv1.UpdateConfig) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-config",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"redis-config": cr.Spec.Updatedconfig,
		},
	}
}

func (r *UpdateConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&customappv1.UpdateConfig{}).
		Complete(r)
}
