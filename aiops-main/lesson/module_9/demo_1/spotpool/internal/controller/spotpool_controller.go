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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devopsgeektimev1 "github.com/lyzhang1999/spotpool/api/v1"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// SpotpoolReconciler reconciles a Spotpool object
type SpotpoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=devops.geektime.devopscamp.gk,resources=spotpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops.geektime.devopscamp.gk,resources=spotpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops.geektime.devopscamp.gk,resources=spotpools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Spotpool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *SpotpoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	spotpool := &devopsgeektimev1.Spotpool{}
	err := r.Get(ctx, req.NamespacedName, spotpool)
	if err != nil {
		log.Error(err, "unable to fetch Spotpool")
		return ctrl.Result{}, err
	}

	// 获取腾讯云 Running 的实例数量，这里需要注意需要一直检查，因为 IP 地址的分配是异步的
	runningList, err := r.getRunningInstanceIDs(spotpool)
	if err != nil {
		log.Error(err, "unable to get running instance ids")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	runningCount := len(runningList)

	switch {
	case runningCount < int(spotpool.Spec.Minimum):
		// 创建实例
		delta := spotpool.Spec.Minimum - int32(runningCount)
		log.Info("creating instances", "delta", delta)
		err = r.runInstances(spotpool, delta)
		if err != nil {
			log.Error(err, "unable to create instances")
			return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
		}
	case runningCount > int(spotpool.Spec.Maximum):
		// 删除实例
		delta := int32(runningCount) - spotpool.Spec.Maximum
		log.Info("terminating instances", "delta", delta)
		err = r.terminateInstances(spotpool, delta)
		if err != nil {
			log.Error(err, "unable to terminate instances")
			return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
		}
	}

	return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
}

func (r *SpotpoolReconciler) terminateInstances(spotpool *devopsgeektimev1.Spotpool, count int32) error {
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return err
	}

	runningInstances, err := r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return err
	}

	instancesIds := runningInstances[:count]
	request := cvm.NewTerminateInstancesRequest()
	request.InstanceIds = common.StringPtrs(instancesIds)

	// 获取返回
	response, err := client.TerminateInstances(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return err
	}
	// other errors
	if err != nil {
		return err
	}

	fmt.Println("Terminate response: ", response)
	fmt.Println("terminate instances success", instancesIds)

	// 更新 status
	_, err = r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return err
	}
	return nil
}

func (r *SpotpoolReconciler) runInstances(spotpool *devopsgeektimev1.Spotpool, count int32) error {
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return err
	}
	request := cvm.NewRunInstancesRequest()
	request.ImageId = common.StringPtr(spotpool.Spec.ImageId)
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(spotpool.Spec.AvaliableZone),
	}
	request.InstanceChargeType = common.StringPtr(spotpool.Spec.InstanceChargeType)
	request.InstanceCount = common.Int64Ptr(int64(count))
	request.InstanceName = common.StringPtr("spotpool" + time.Now().Format("20060102150405"))
	request.InstanceType = common.StringPtr(spotpool.Spec.InstanceType)
	request.InternetAccessible = &cvm.InternetAccessible{
		InternetChargeType:      common.StringPtr("BANDWIDTH_POSTPAID_BY_HOUR"),
		InternetMaxBandwidthOut: common.Int64Ptr(1),
		PublicIpAssigned:        common.BoolPtr(true),
	}
	request.LoginSettings = &cvm.LoginSettings{
		Password: common.StringPtr("Password123"),
	}
	request.SecurityGroupIds = common.StringPtrs(spotpool.Spec.SecurityGroupId)
	request.SystemDisk = &cvm.SystemDisk{
		DiskType: common.StringPtr("CLOUD_BSSD"),
		DiskSize: common.Int64Ptr(100),
	}
	request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{
		SubnetId: common.StringPtr(spotpool.Spec.SubnetId),
		VpcId:    common.StringPtr(spotpool.Spec.VpcId),
	}

	// print request
	fmt.Println(request.ToJsonString())

	// 创建实例
	response, err := client.RunInstances(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return err
	}
	// other errors
	if err != nil {
		return err
	}

	// 获取到返回的 instancesid
	instanceIds := make([]string, 0, len(response.Response.InstanceIdSet))
	for _, instanceId := range response.Response.InstanceIdSet {
		instanceIds = append(instanceIds, *instanceId)
	}

	fmt.Println("run instances success", instanceIds)
	// 更新 status
	_, err = r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return err
	}
	return nil
}

func (r *SpotpoolReconciler) getRunningInstanceIDs(spotpool *devopsgeektimev1.Spotpool) ([]string, error) {
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return nil, err
	}

	request := cvm.NewDescribeInstancesRequest()
	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	var instances []devopsgeektimev1.Instances
	var runningInstanceIDs []string
	for _, instance := range response.Response.InstanceSet {
		if *instance.InstanceState == "RUNNING" || *instance.InstanceState == "PENDING" || *instance.InstanceState == "STARTING" {
			runningInstanceIDs = append(runningInstanceIDs, *instance.InstanceId)
		}
		// 检查实例的公网 IP，如果不存在公网 IP，则继续重试
		if len(instance.PublicIpAddresses) == 0 {
			return nil, fmt.Errorf("instance %s does not have public ip", *instance.InstanceId)
		}
		instances = append(instances, devopsgeektimev1.Instances{
			InstanceId: *instance.InstanceId,
			PublicIp:   *instance.PublicIpAddresses[0],
		})
	}
	// 更新 status
	spotpool.Status.Instances = instances
	err = r.Status().Update(context.Background(), spotpool)
	if err != nil {
		return nil, err
	}
	return runningInstanceIDs, nil
}

func (r *SpotpoolReconciler) createCVMClient(spec devopsgeektimev1.SpotpoolSpec) (*cvm.Client, error) {
	credential := common.NewCredential(spec.SecretId, spec.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "POST"
	cpf.HttpProfile.ReqTimeout = 30
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
		Complete(r)
}
