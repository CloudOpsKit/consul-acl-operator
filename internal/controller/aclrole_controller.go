/*
Copyright 2024.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	consulcloudopskitorgv1alpha1 "github.com/CloudOpsKit/consul-acl-operator/api/v1alpha1"
)

const aclRoleFinalizer = "aclrole.consul.cloudopskit.finalizer"

// AclRoleReconciler reconciles a AclRole object
type AclRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
    Config        operatorConfig.Config
    EventRecorder record.EventRecorder
}

//+kubebuilder:rbac:groups=consul.cloudopskit.org.cloudopskit.org,resources=aclroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=consul.cloudopskit.org.cloudopskit.org,resources=aclroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=consul.cloudopskit.org.cloudopskit.org,resources=aclroles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AclRole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AclRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
reqLogger := r.Log.With("aclrole", req.NamespacedName)
	reqLogger.Debug("Reconciling AclRole")

	aclRole := &consulv1beta1.AclRole{}

	if err := r.Get(ctx, req.NamespacedName, aclRole); err != nil {
		if client.IgnoreNotFound(err) == nil {
			reqLogger.Warn("AclRole resource not found")
			if err = consul.DeleteAclRole(reqLogger, r.Config, req.Namespace+"_"+req.Name); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	if aclRole.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(aclRole, aclRoleFinalizer) {
			if err := r.finalizeAclRole(reqLogger, r.Config, aclRole); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(aclRole, aclRoleFinalizer)
			if err := r.Update(ctx, aclRole); err != nil {
				reqLogger.Error(err, "Failed to delete finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(aclRole, aclRoleFinalizer) {
		controllerutil.AddFinalizer(aclRole, aclRoleFinalizer)
		if err := r.Update(ctx, aclRole); err != nil {
			reqLogger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	consulClient, err := api.NewClient(&r.Config.ConsulConfig)
	if err != nil {
		r.EventRecorder.Eventf(aclRole, v1.EventTypeWarning, events.EventStatusFailed, "Unable to create Consul client")
		return ctrl.Result{}, err
	}
	policies, err := aclRole.GetPolicies(r.Config.ConsulConfig)
	if err != nil {
		r.EventRecorder.Eventf(aclRole, v1.EventTypeWarning, events.EventStatusFailed, "Cannot get ACL policies")

		objStatus := aclRole.Status.DeepCopy()
		objStatus.Health = status.HealthStatusMissing

		if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	aclRoleName := fmt.Sprintf("%s_%s", aclRole.Namespace, aclRole.Name)

	role, _, err := consulClient.ACL().RoleReadByName(aclRoleName, nil)
	if err != nil || role == nil {
		r.EventRecorder.Eventf(aclRole, v1.EventTypeNormal, events.EventStatusInfo, "AclRole does not exist")

		objStatus := aclRole.Status.DeepCopy()
		objStatus.Health = status.HealthStatusMissing

		if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
			return ctrl.Result{}, err
		}

	} else {
		isPoliciesSame, err := consulv1beta1.IsPoliciesEqual(policies, role.Policies)
		if err != nil {
			r.EventRecorder.Eventf(aclRole, v1.EventTypeNormal, events.EventStatusInfo, "AclRole policies are not the same")
		}

		if role.ID == aclRole.Status.ID && role.Description == aclRole.Spec.Description && isPoliciesSame {
			objStatus := aclRole.Status.DeepCopy()
			objStatus.Health = status.HealthStatusHealthy

			if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
				return ctrl.Result{}, err
			}

			r.EventRecorder.Eventf(aclRole, v1.EventTypeNormal, events.EventStatusInfo, "AclRole is up to date")

			return ctrl.Result{}, err
		} else {
			objStatus := aclRole.Status.DeepCopy()
			objStatus.Health = status.HealthStatusProgressing

			if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	id, err := consul.CreateOrUpdateAclRole(reqLogger, r.Config, &api.ACLRole{
		ID:          aclRole.Status.ID,
		Name:        aclRoleName,
		Description: aclRole.Spec.Description,
		Policies:    policies,
	})
	if err != nil {
		r.EventRecorder.Eventf(aclRole, v1.EventTypeWarning, events.EventStatusFailed, "CreateOrUpdateAclRole failed")

		objStatus := aclRole.Status.DeepCopy()
		objStatus.Health = status.HealthStatusDegraded

		if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if aclRole.Status.ID != id || aclRole.Status.ID == "" {
		aclRole.Status.ID = id
		err = r.Status().Update(ctx, aclRole)
		if err != nil {
			objStatus := aclRole.Status.DeepCopy()
			objStatus.Health = status.HealthStatusDegraded

			if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
				return ctrl.Result{}, err
			}
		}

		objStatus := aclRole.Status.DeepCopy()
		objStatus.Health = status.HealthStatusHealthy

		if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		objStatus := aclRole.Status.DeepCopy()
		objStatus.Health = status.HealthStatusHealthy

		if err := r.updateAclRoleStatus(aclRole, objStatus); err != nil {
			return ctrl.Result{}, err
		}

		r.EventRecorder.Eventf(aclRole, v1.EventTypeNormal, events.EventStatusUpdated, "CreateOrUpdateAclRole created")
	}

	return ctrl.Result{}, nil
}

func (r *AclRoleReconciler) finalizeAclRole(reqLogger *zap.SugaredLogger, config operatorConfig.Config, m *consulv1beta1.AclRole) error {
	if err := consul.DeleteAclRole(reqLogger, config, m.Namespace+"_"+m.Name); err != nil {
		return err
	}
	reqLogger.Debug("Successfully finalized AclRole")
	return nil
}

func (r *AclRoleReconciler) updateAclRoleStatus(aclRoleController *consulv1beta1.AclRole, status *consulv1beta1.AclRoleStatus) error {
	patch := client.MergeFrom(aclRoleController.DeepCopy())
	aclRoleController.Status = *status
	return r.Status().Patch(context.TODO(), aclRoleController, patch)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AclRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&consulcloudopskitorgv1alpha1.AclRole{}).
		Complete(r)
}
