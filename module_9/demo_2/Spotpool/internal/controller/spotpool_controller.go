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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	// 检查 Kong API 网关的初始配置
	err = r.CheckKongAiProxy(spotpool)
	if err != nil {
		log.Error(err, "unable to check kong gateway")
		return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
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

	err = r.SyncKongUpstream(spotpool)
	if err != nil {
		log.Error(err, "unable to sync kong upstream")
	}

	return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
}

func (r *SpotpoolReconciler) CheckKongAiProxy(spotpool *v1.Spotpool) error {
	kongURL := fmt.Sprintf("http://%s:8001", spotpool.Spec.KongGatewayIP)
	resp, err := http.Get(kongURL + "/services")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	postBody, err := json.Marshal(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(postBody, &result); err != nil {
		return err
	}

	serviecs, ok := result["data"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	// 检查是否存在 name=ai-proxy 的 service
	for _, service := range serviecs {
		if serviceMap, ok := service.(map[string]interface{}); ok {
			if name, exists := serviceMap["name"].(string); exists && name == "ai-proxy" {
				fmt.Printf("service ai-proxy exists in kong gateway\n")
				return nil
			}
		}
	}

	// 如果没有，创建 ai-proxy service
	if err := r.createKongService(spotpool); err != nil {
		return err
	}
	fmt.Printf("service ai-proxy created\n")
	return nil
}

func (r *SpotpoolReconciler) createKongService(spotpool *v1.Spotpool) error {
	KongURL := fmt.Sprintf("http://%s:8001", spotpool.Spec.KongGatewayIP)

	// 创建 Upsteam
	upstreamURL := fmt.Sprintf("%s/upstreams", KongURL)
	upstreamData := map[string]string{
		"name": "llama2-upstream",
	}
	upstreamBody, err := json.Marshal(upstreamData)
	if err != nil {
		return err
	}
	upstreamResp, err := http.Post(upstreamURL, "application/json", strings.NewReader(string(upstreamBody)))
	if err != nil {
		return err
	}
	defer upstreamResp.Body.Close()

	// 构造创建服务的请求
	serviceData := map[string]string{
		"name":     "ai-proxy",
		"host":     "llama2-upstream",
		"portocal": "http",
		"path":     "/api/chat",
	}

	serviceBody, _ := json.Marshal(serviceData)

	serviceResp, err := http.Post(fmt.Sprintf("%s/services", KongURL), "application/json", strings.NewReader(string(serviceBody)))
	if err != nil {
		return err
	}
	defer serviceResp.Body.Close()

	if serviceResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create service failed, unexpected status code: %d", serviceResp.StatusCode)
	}

	// 创建  Route
	routeURL := fmt.Sprintf("%s/services/ai-proxy/routes", KongURL)
	routeData := map[string]interface{}{
		"name":  "ai-proxy-route",
		"paths": []string{"~/ollama-chat$"},
	}
	routeBody, _ := json.Marshal(routeData)

	// 发起请求
	routeResp, err := http.Post(routeURL, "application/json", strings.NewReader(string(routeBody)))
	if err != nil {
		return err
	}
	defer routeResp.Body.Close()

	if routeResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create route failed, unexpected status code: %d", routeResp.StatusCode)
	}

	fmt.Printf("route ai-proxy-route created\n")
	return nil
}

// SyncKongUpstream 同步 Kong 中 upstream 的 targets 与 spotpool 当前实例。
// 它会根据 spotpool 当前实例添加或删除 targets。
// 如果 upstream 为空，则会将所有实例添加为 targets。
// 如果某个 target 不在 spotpool 当前实例中，则会删除该 target。
// 如果某个实例不在 upstream 当前 targets 中，则会将该实例添加为 target。
func (r *SpotpoolReconciler) SyncKongUpstream(spotpool *v1.Spotpool) error {
	kongURL := fmt.Sprintf("http://%s:8001/upstreams/llama2-upstream/targets", spotpool.Spec.KongGatewayIP)

	// 获取当前的 targets
	resp, err := http.Get(kongURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 读取解析 body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 使用 map 解析 json
	var targetResponse map[string]interface{}
	err = json.Unmarshal(body, &targetResponse)
	if err != nil {
		return err
	}
	// 获取当前的 targets
	currentTargets, ok := targetResponse["data"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}
	// 从  spotpool 中获取当前的  instances
	instances := spotpool.Status.Instances
	// 创建一个目标 IP的集合，便于后续比较
	instancesIPs := make(map[string]bool)
	for _, instance := range instances {
		instancesIPs[instance.PublicIP] = true
	}

	// 如果当前的 targets 为空，那么直接添加所有的 instances
	if len(currentTargets) == 0 {
		for _, instance := range instances {
			instanceIP := fmt.Sprintf("%s:8080", instance.PublicIP)
			postData := map[string]string{
				"target": instanceIP,
			}
			postBody, err := json.Marshal(postData)
			if err != nil {
				return err
			}

			postResq, err := http.Post(kongURL, "application/json", strings.NewReader(string(postBody)))
			if err != nil {
				return err
			}
			defer postResq.Body.Close()

			if postResq.StatusCode != http.StatusCreated {
				return fmt.Errorf("create target failed,unexpected status code: %d", postResq.StatusCode)
			}
			fmt.Printf("create target %s success\n", instanceIP)
		}
	}

	// 遍历当前的 targets, 检查与当前实例是否匹配
	for _, target := range currentTargets {
		targetMap, ok := target.(map[string]interface{})
		if !ok {
			continue
		}
		// 获取目标IP
		targetIP := targetMap["target"].(string)
		targetHost := strings.Split(targetIP, ":")[0]

		// 匹配逻辑
		if _, exists := instancesIPs[targetHost]; !exists {
			// 目标IP不在实例列表里，删除这个target
			targetID := targetMap["target"].(string)
			deleteURL := fmt.Sprintf("http://%s:8001/upstreams/llama2-upstream/targets/%s", spotpool.Spec.KongGatewayIP, targetID)
			req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
			if err != nil {
				return err
			}

			delResp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer delResp.Body.Close()

			if delResp.StatusCode != http.StatusNoContent {
				return fmt.Errorf("delete target failed, status code: %d", delResp.StatusCode)
			}
			fmt.Printf("delete target %s success\n", targetIP)
		}
	}

	// 检查实例列表中是否有 target 不存在的情况
	for _, instance := range instances {
		instanceIP := fmt.Sprintf("%s:8080", instance.PublicIP)
		found := false

		// 检查当前目标是否存在
		for _, target := range currentTargets {
			targetMap, ok := target.(map[string]interface{})
			if !ok {
				continue
			}
			if targetMap["target"].(string) == instanceIP {
				found = true
				break
			}
		}

		// 如果实例在目标列表中不存在，那么添加这个 target
		if !found {
			postData := map[string]string{
				"target": instanceIP,
			}
			postBody, err := json.Marshal(postData)
			if err != nil {
				return err
			}

			postResp, err := http.Post(kongURL, "application/json", strings.NewReader(string(postBody)))
			if err != nil {
				return err
			}
			defer postResp.Body.Close()

			if postResp.StatusCode != http.StatusCreated {
				return fmt.Errorf("create target failed,unexpected status code: %d", postResp.StatusCode)
			}
			fmt.Printf("create target %s success\n", instanceIP)
		}
	}

	return nil
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
		ReqMethod:  "POST",
		ReqTimeout: 50,
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
