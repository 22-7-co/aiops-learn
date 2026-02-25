/*
Copyright 2026.

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

	autoscalingv1 "cronhpa/api/v1"

	"github.com/robfig/cron/v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CronHPAReconciler reconciles a CronHPA object
type CronHPAReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=autoscaling.aiops.com,resources=cronhpas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.aiops.com,resources=cronhpas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling.aiops.com,resources=cronhpas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CronHPA object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *CronHPAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// TODO(user): your logic here
	log.Info("Reconciling CronHPA")
	var cronHPA autoscalingv1.CronHPA
	if err := r.Get(ctx, req.NamespacedName, &cronHPA); err != nil {
		log.Error(err, "unable to fetch CronHPA")
		return ctrl.Result{}, err
	}
	now := time.Now()
	var earliestNextRunTime *time.Time
	// 遍历 jobs, 检查调度时间并且更新目标工作负载的副本数
	for _, job := range cronHPA.Spec.Jobs {
		lastRunTime := cronHPA.Status.LastRunTime[job.Name]
		// 计算下一次时间
		nextScheduledTime, err := r.getNextScheduledTime(job.Schedule, lastRunTime.Time)
		if err != nil {
			log.Error(err, "failed to get next  scheduled time")
			return reconcile.Result{}, err
		}
		log.Info("Job info: ", "name", job.Name, "lastRunTime", lastRunTime, "nextScheduledTime", nextScheduledTime)
		// 检查当前时间是否已经到达或者超过了计划的运行时间
		if now.After(nextScheduledTime) || now.Equal(nextScheduledTime) {
			// 更新目标工作负载的副本数
			log.Info("update replicas", "job", job.Name, "targetSize", job.TargetSize)
			if err := r.updateDeploymentReplicas(ctx, &cronHPA, cronHPA.Spec.ScaleTargetRef, job); err != nil {
				log.Error(err, "failed to update deployment replicas")
				return reconcile.Result{}, err
			}
			// 更新 CronHPA 状态
			cronHPA.Status.CurrentReplicas = job.TargetSize
			cronHPA.Status.LastScaleTime = &metav1.Time{Time: now}

			// 更新作业的最后运行时间
			if cronHPA.Status.LastRunTime == nil {
				cronHPA.Status.LastRunTime = make(map[string]metav1.Time)
			}
			cronHPA.Status.LastRunTime[job.Name] = metav1.Time{Time: now}

			// 计算下一次运行时间(从现在开始)
			nextRunTime, _ := r.getNextScheduledTime(job.Schedule, now)
			if earliestNextRunTime == nil || nextRunTime.Before(*earliestNextRunTime) {
				earliestNextRunTime = &nextRunTime
			}
		} else {
			// 如果当前时间没到达计划时间，就把这个时间当作下一次运行时间(确保程序不会执行已过期的计划。)
			if earliestNextRunTime == nil || now.Before(*earliestNextRunTime) {
				earliestNextRunTime = &nextScheduledTime
			}
		}
	}
	// 更新 CronHPA 实例状态
	if err := r.Status().Update(ctx, &cronHPA); err != nil {
		return reconcile.Result{}, err
	}

	// 如果有下一次运行时间，设置重新入队
	if earliestNextRunTime != nil {
		requeueAfter := earliestNextRunTime.Sub(now)
		if requeueAfter < 0 {
			requeueAfter = time.Second // 如果计算出的时间已经过去，则在1秒后重新入队
		}
		log.Info("Requeue after", "time", requeueAfter)
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func (r *CronHPAReconciler) getNextScheduledTime(schedule string, after time.Time) (time.Time, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(schedule)
	if err != nil {
		return time.Time{}, err
	}
	return cronSchedule.Next(after), nil
}

func (r *CronHPAReconciler) updateDeploymentReplicas(ctx context.Context, cronHPA *autoscalingv1.CronHPA, scaleTargetRef autoscalingv1.ScaleTargetRefrence, job autoscalingv1.JobSpec) error {
	log := logf.FromContext(ctx)

	// deployment 对象
	deployment := &appsv1.Deployment{}
	deploymentKey := types.NamespacedName{
		Namespace: cronHPA.Namespace,
		Name:      scaleTargetRef.Name,
	}
	if err := r.Get(ctx, deploymentKey, deployment); err != nil {
		log.Error(err, "Failed to get deployment")
		return err
	}
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == job.TargetSize {
		log.Info("Deployment replicas already match target size")
		return nil
	}
	// 更新 deployment 的副本数
	deployment.Spec.Replicas = &job.TargetSize
	if err := r.Update(ctx, deployment); err != nil {
		log.Error(err, "Failed to update deployment")
		return err
	}
	log.Info("deployment replicas updated", "targetSize", job.TargetSize)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronHPAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingv1.CronHPA{}).
		Named("cronhpa").
		Complete(r)
}
