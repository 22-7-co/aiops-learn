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
	"fmt"
	"time"

	devopsgeektimev1 "22-7/operator-aiops/api/v1"
	v1 "22-7/operator-aiops/api/v1"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// SpotpoolReconciler reconciles a Spotpool object
type SpotpoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=devops.geektime.devopscamp.gk,resources=spotpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=devops.geektime.devopscamp.gk,resources=spotpools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=devops.geektime.devopscamp.gk,resources=spotpools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Spotpool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *SpotpoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	spotpool := &v1.Spotpool{}
	err := r.Get(ctx, req.NamespacedName, spotpool)
	if err != nil {
		log.Error(err, "unable to fetch Spotpool")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//  获取腾讯云 Running 的使用数量，这里需要注意一些检查，因为IP地址分配是异步的
	runningList, err := r.getRunningInstanceIds(spotpool)
	if err != nil {
		log.Error(err, "unable to get running instance ids")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	runningCount := len(runningList)
	// 判断实例数量
	switch {
	case runningCount < int(spotpool.Spec.Minimum):
		// 申请实例
		delta := spotpool.Spec.Minimum - int32(runningCount)
		log.Info("creating instances", "delta", delta)
		err = r.runInstances(spotpool, delta)
		if err != nil {
			log.Error(err, "unable to create instances")
			return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
		}
	case runningCount > int(spotpool.Spec.Maximum):
		// 删除实例
		dalta := int32(runningCount) - spotpool.Spec.Maximum
		log.Info("deleting instances", "delta", dalta)
		err = r.terminateInstances(spotpool, dalta)
		if err != nil {
			log.Error(err, "unable to terminate instances")
			return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (r *SpotpoolReconciler) runInstances(spotpool *v1.Spotpool, count int32) error {
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return err
	}
	request := &cvm.RunInstancesRequest{
		ImageId: common.StringPtr(spotpool.Spec.ImageId),
		Placement: &cvm.Placement{
			Zone: common.StringPtr(spotpool.Spec.AvaliableZone),
		},
		InstanceChargeType: common.StringPtr(spotpool.Spec.InstanceChargeType),
		InstanceType:       common.StringPtr(spotpool.Spec.InstanceType),
		InstanceName:       common.StringPtr("spotpool" + time.Now().Format("20060102150405")),
		InstanceCount:      common.Int64Ptr(int64(count)),
		InternetAccessible: &cvm.InternetAccessible{
			InternetChargeType:      common.StringPtr("BANDWIDTH_POSTPAID_BY_HOUT"),
			InternetMaxBandwidthOut: common.Int64Ptr(100),
			PublicIpAssigned:        common.BoolPtr(true),
		},
		LoginSettings: &cvm.LoginSettings{
			Password: common.StringPtr("Password123"),
		},
		SecurityGroupIds: common.StringPtrs(spotpool.Spec.SecurityGroupId),
		SystemDisk: &cvm.SystemDisk{
			DiskType: common.StringPtr("CLOUD_BSSD"),
			DiskSize: common.Int64Ptr(100),
		},
		VirtualPrivateCloud: &cvm.VirtualPrivateCloud{
			VpcId:    common.StringPtr(spotpool.Spec.VpcId),
			SubnetId: common.StringPtr(spotpool.Spec.SubnetId),
		},
	}

	response, err := client.RunInstances(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return err
	}
	if err != nil {
		return err
	}

	// 获取到返回的实例ID
	instanceIds := make([]string, 0, len(response.Response.InstanceIdSet))
	for _, instanceId := range response.Response.InstanceIdSet {
		instanceIds = append(instanceIds, *instanceId)
	}
	fmt.Println("run instance success", instanceIds)
	// 更新 status
	_, err = r.getRunningInstanceIds(spotpool)
	if err != nil {
		return err
	}
	return nil
}
func (r *SpotpoolReconciler) terminateInstances(spotpool *v1.Spotpool, count int32) error {
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return err
	}

	// 获取腾讯云 当前使用的IDs
	runningInstanceIds, err := r.getRunningInstanceIds(spotpool)
	if err != nil {
		return err
	}
	request := &cvm.TerminateInstancesRequest{
		InstanceIds: common.StringPtrs(runningInstanceIds[:count]),
	}
	_, err = client.TerminateInstances(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return err
	}
	if err != nil {
		return err
	}
	fmt.Println("terminate instance success", runningInstanceIds[:count])

	// 更新 status
	_, err = r.getRunningInstanceIds(spotpool)
	if err != nil {
		return err
	}

	return nil
}

func (r *SpotpoolReconciler) getRunningInstanceIds(spotpool *v1.Spotpool) ([]string, error) {
	// 获取腾讯云 Running 的使用数量，这里需要注意一些检查，因为IP地址分配是异步的
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeInstancesRequest()

	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, nil
	}
	var instances []v1.Instances
	var runningInstanceIds []string
	for _, instance := range response.Response.InstanceSet {
		if *instance.InstanceState == "RUNNING" || *instance.InstanceState == "PENDING" || *instance.InstanceState == "STOPPED" {
			runningInstanceIds = append(runningInstanceIds, *instance.InstanceId)
		}
		// 检查实例的公网ip，如果不存在公网ip，则继续重试
		if len(instance.PublicIpAddresses) == 0 {
			return nil, fmt.Errorf("instance %s has no public ip", *instance.InstanceId)
		}
		instances = append(instances, v1.Instances{
			InstanceID: *instance.InstanceId,
			PublicIP:   *instance.PublicIpAddresses[0],
		})
	}
	// 更新 Status
	spotpool.Status.Instances = instances
	err = r.Status().Update(context.Background(), spotpool)
	if err != nil {
		return nil, err
	}
	return runningInstanceIds, nil
}

func (r *SpotpoolReconciler) createCVMClient(spec v1.SpotpoolSpec) (*cvm.Client, error) {
	// 创建腾讯云 CVM 客户端
	credential := common.NewCredential(spec.SecretId, spec.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile = &profile.HttpProfile{
		ReqMethod: "POST",
		Endpoint:  "cvm.tencentcloudapi.com",
	}
	cpf.SignMethod = "HmacSHA1"
	client, err := cvm.NewClient(credential, spec.Region, cpf)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotpoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsgeektimev1.Spotpool{}).
		Named("spotpool").
		Complete(r)
}
