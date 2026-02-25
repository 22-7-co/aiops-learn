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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devopsgeektimev1 "devops-advanced-camp/spotpool/api/v1"
	v1 "devops-advanced-camp/spotpool/api/v1"

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

	// Get the Spotpool instance from the API server.
	spotpool := &v1.Spotpool{}
	err := r.Get(ctx, req.NamespacedName, spotpool)
	if err != nil {
		log.Error(err, "unable to fetch Spotpool")
		return ctrl.Result{}, err
	}

	// check Kong API Gateway
	err = r.checkKongAiProxy(spotpool)

	// 获取 Running 实例数量，这里需要一直检查，因为 IP 地址是异步分配的
	_, err = r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Get the current number of running CVM instances.
	runningCount, err := r.countRunningInstances(spotpool)
	if err != nil {
		log.Error(err, "failed to get running CVM instance count")
		return ctrl.Result{}, err
	}

	// Take actions based on the desired state (Minimum and Maximum) and current state (runningCount).
	switch {
	case runningCount < spotpool.Spec.Minimum:
		// Need to start more CVMs.
		delta := spotpool.Spec.Minimum - runningCount
		logger.Info("Starting ", delta, " CVM instances")
		err = r.runInstances(spotpool, delta)
		if err != nil {
			log.Info(err.Error(), "failed to start CVM instances")
			// 40s 后重试
			return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
		}
	case runningCount > spotpool.Spec.Maximum:
		// Need to terminate CVMs.
		delta := runningCount - spotpool.Spec.Maximum
		logger.Info("Terminating ", delta, " CVM instances")
		err = r.terminateInstances(spotpool, delta)
		if err != nil {
			log.Info(err.Error(), "failed to terminate CVM instances")
			return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
		}
	}

	// Update the Spotpool status with the current number of running instances.
	spotpool.Status.Size = runningCount
	err = r.Status().Update(ctx, spotpool)
	if err != nil {
		log.Error(err, "failed to update Spotpool status")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled Spotpool ", spotpool.Name)
	// 同步 Kong Upstream
	err = r.syncKongUpstream(spotpool)
	if err != nil {
		log.Error(err, "failed to sync Kong Upstream")
	}
	return ctrl.Result{RequeueAfter: 40 * time.Second}, nil
}

func (r *SpotpoolReconciler) syncKongUpstream(spotpool *v1.Spotpool) error {
	kongURL := fmt.Sprintf("http://%s:8001/upstreams/llama2-upstream/targets", spotpool.Spec.KongGatewayIP)

	// 获取当前 targets
	resp, err := http.Get(kongURL)
	if err != nil {
		return fmt.Errorf("failed to fetch targets from Kong: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status when fetching targets: %d", resp.StatusCode)
	}

	// 读取并解析响应数据
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// 使用 map 解析 JSON 数据
	var targetResponse map[string]interface{}
	err = json.Unmarshal(body, &targetResponse)
	if err != nil {
		return fmt.Errorf("failed to unmarshal targets response: %v", err)
	}

	// 获取当前 targets 数据
	currentTargets, ok := targetResponse["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format, 'data' field not found")
	}

	// 从 spotpool 中获取实例列表
	instances := spotpool.Status.Instances

	// 创建一个目标 IP 的集合，以便于后续比较
	instanceIPs := make(map[string]bool)
	for _, instance := range instances {
		instanceIPs[instance.PublicIp] = true
	}

	// 如果当前 targets 为空，直接添加所有实例为新的 targets
	if len(currentTargets) == 0 {
		for _, instance := range instances {
			instanceIP := fmt.Sprintf("%s:8080", instance.PublicIp)
			postData := map[string]string{
				"target": instanceIP,
			}
			postBody, err := json.Marshal(postData)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON for new target: %v", err)
			}

			postResp, err := http.Post(
				kongURL,
				"application/json",
				strings.NewReader(string(postBody)),
			)
			if err != nil {
				return fmt.Errorf("failed to create new target for %s: %v", instanceIP, err)
			}
			defer postResp.Body.Close()

			if postResp.StatusCode != http.StatusCreated {
				return fmt.Errorf("failed to create target for %s, response status: %d", instanceIP, postResp.StatusCode)
			}

			fmt.Printf("Created target for %s\n", instanceIP)
		}
		return nil // 处理完所有实例后返回
	}

	// 遍历当前 targets，检查与实例的匹配
	for _, target := range currentTargets {
		targetMap, ok := target.(map[string]interface{})
		if !ok {
			continue
		}

		// 获取目标 IP
		targetIP := targetMap["target"].(string)      // 格式: "43.156.25.202:8080"
		targetHost := strings.Split(targetIP, ":")[0] // 获取 IP 部分

		// 如果在实例列表中找到匹配的 IP
		if _, exists := instanceIPs[targetHost]; !exists {
			// 如果目标 IP 不在实例列表中，则删除该 target
			targetID := targetMap["target"].(string)
			deleteURL := fmt.Sprintf("http://%s:8001/upstreams/llama2-upstream/targets/%s", spotpool.Spec.KongGatewayIP, targetID)
			req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create delete request: %v", err)
			}

			delResp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to delete target %s: %v", targetID, err)
			}
			delResp.Body.Close()

			if delResp.StatusCode != http.StatusNoContent {
				return fmt.Errorf("failed to delete target %s, response status: %d", targetID, delResp.StatusCode)
			}

			fmt.Printf("Deleted target %s\n", targetID)
		}
	}

	// 检查实例列表中是否有目标不存在的情况
	for _, instance := range instances {
		instanceIP := fmt.Sprintf("%s:8080", instance.PublicIp) // 使用格式化的 target 字符串
		found := false

		// 检查当前目标是否存在
		for _, target := range currentTargets {
			targetMap, ok := target.(map[string]interface{})
			if !ok {
				continue
			}
			if targetMap["target"] == instanceIP {
				found = true
				break
			}
		}

		// 如果实例在目标列表中不存在，则添加新的 target
		if !found {
			postData := map[string]string{
				"target": instanceIP,
			}
			postBody, err := json.Marshal(postData)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON for new target: %v", err)
			}

			postResp, err := http.Post(
				kongURL,
				"application/json",
				strings.NewReader(string(postBody)),
			)
			if err != nil {
				return fmt.Errorf("failed to create new target for %s: %v", instanceIP, err)
			}
			defer postResp.Body.Close()

			if postResp.StatusCode != http.StatusCreated {
				return fmt.Errorf("failed to create target for %s, response status: %d", instanceIP, postResp.StatusCode)
			}

			fmt.Printf("Created target for %s\n", instanceIP)
		}
	}

	return nil
}

func (r *SpotpoolReconciler) checkKongAiProxy(spotpool *v1.Spotpool) error {
	// 获取 KongGatewayIP
	kongGatewayIP := spotpool.Spec.KongGatewayIP
	kongURL := fmt.Sprintf("http://%s:8001", kongGatewayIP)

	// 使用 http.Client 发起 GET 请求
	client := &http.Client{}
	resp, err := client.Get(kongURL + "/services")
	if err != nil {
		return fmt.Errorf("failed to fetch services from Kong: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	// 解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	services, ok := result["data"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format, 'data' field not found")
	}

	// 检查是否存在 name=ai-proxy 的服务
	for _, service := range services {
		if serviceMap, ok := service.(map[string]interface{}); ok {
			if name, exists := serviceMap["name"].(string); exists && name == "ai-proxy" {
				fmt.Println("ai-proxy service found.")
				return nil
			}
		}
	}

	// 创建新的 ai-proxy 服务
	fmt.Println("ai-proxy service not found, creating a new one.")
	if err := r.createKongServiceAndRouteAndUpstream(spotpool); err != nil {
		return err
	}

	fmt.Println("ai-proxy service and route created successfully.")
	return nil
}

func (r *SpotpoolReconciler) createKongServiceAndRouteAndUpstream(spotpool *v1.Spotpool) error {
	kongGatewayIP := spotpool.Spec.KongGatewayIP
	kongURL := fmt.Sprintf("http://%s:8001", kongGatewayIP)

	client := &http.Client{}

	// 创建 Upstream
	upstreamURL := fmt.Sprintf("%s/upstreams", kongURL)
	upstreamData := map[string]string{
		"name": "llama2-upstream",
	}
	upstreamBody, _ := json.Marshal(upstreamData)
	// 请求
	upstreamResp, err := client.Post(upstreamURL, "application/json", bytes.NewReader(upstreamBody))
	if err != nil {
		return fmt.Errorf("failed to create upstream: %v", err)
	}
	defer upstreamResp.Body.Close()

	// 构造创建服务的 POST 请求数据
	serviceData := map[string]string{
		"name":     "ai-proxy",
		"host":     "llama2-upstream",
		"protocol": "http",
		"path":     "/api/chat",
	}

	serviceBody, _ := json.Marshal(serviceData)

	// 发起创建服务的请求
	serviceResp, err := client.Post(kongURL+"/services", "application/json", bytes.NewReader(serviceBody))
	if err != nil {
		return fmt.Errorf("failed to create ai-proxy service: %v", err)
	}
	defer serviceResp.Body.Close()

	if serviceResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create ai-proxy service, status: %d", serviceResp.StatusCode)
	}

	// 创建 route
	routeURL := fmt.Sprintf("%s/services/ai-proxy/routes", kongURL)
	routeData := map[string]interface{}{
		"name":  "ollama-chat",
		"paths": []string{"~/ollama-chat$"},
	}
	routeBody, _ := json.Marshal(routeData)

	// 发起创建 route 的请求
	routeResp, err := client.Post(routeURL, "application/json", bytes.NewReader(routeBody))
	if err != nil {
		return fmt.Errorf("failed to create ai-proxy route: %v", err)
	}
	defer routeResp.Body.Close()

	if routeResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create ai-proxy route, status: %d", routeResp.StatusCode)
	}

	return nil
}

// runInstances starts the specified number of CVMs based on the Spotpool configuration.
func (r *SpotpoolReconciler) runInstances(spotpool *v1.Spotpool, count int32) error {
	// Implement your logic here to call the Tencent Cloud CVM API to start new CVM instances based on the Spotpool spec.
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return err
	}

	request := cvm.NewRunInstancesRequest()
	request.ImageId = common.StringPtr(spotpool.Spec.ImageId)
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(spotpool.Spec.AvailabilityZone),
	}

	request.InstanceChargeType = common.StringPtr(spotpool.Spec.InstanceChargeType)
	request.InstanceCount = common.Int64Ptr(int64(count))
	request.InstanceName = common.StringPtr("spotpool-instance-" + time.Now().Format("20060102150405"))
	request.InstanceType = common.StringPtr(spotpool.Spec.InstanceType)
	request.InternetAccessible = &cvm.InternetAccessible{
		InternetChargeType:      common.StringPtr("BANDWIDTH_POSTPAID_BY_HOUR"),
		InternetMaxBandwidthOut: common.Int64Ptr(1),
		PublicIpAssigned:        common.BoolPtr(true),
	}
	request.LoginSettings = &cvm.LoginSettings{
		Password: common.StringPtr("Password123"),
	}
	request.SecurityGroupIds = common.StringPtrs(spotpool.Spec.SecurityGroupIds)
	request.SystemDisk = &cvm.SystemDisk{
		DiskSize: common.Int64Ptr(100),
		DiskType: common.StringPtr("CLOUD_BSSD"),
	}
	request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{
		SubnetId: common.StringPtr(spotpool.Spec.SubnetId),
		VpcId:    common.StringPtr(spotpool.Spec.VpcId),
	}
	// cvm instance number
	request.InstanceCount = common.Int64Ptr(int64(count))

	// get response structure
	response, err := client.RunInstances(request)
	// API errors
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return err
	}
	// unexpected errors
	if err != nil {
		panic(err)
	}
	// Extract instance IDs from the response
	instanceIds := make([]string, 0, len(response.Response.InstanceIdSet))
	for _, instanceId := range response.Response.InstanceIdSet {
		instanceIds = append(instanceIds, *instanceId)
	}
	logger.Info("Started ", len(instanceIds), " CVM instances: ", instanceIds)
	// Get the instance IDs to terminate, for update status
	_, err = r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return err
	}
	return nil
}

// terminateInstances terminates the specified number of CVMs based on the Spotpool configuration.
func (r *SpotpoolReconciler) terminateInstances(spotpool *v1.Spotpool, count int32) error {
	// Implement your logic here to call the Tencent Cloud CVM API to terminate existing CVM instances based on the Spotpool spec.
	// This example code shows a basic invocation but you would need to fill in the details based on the actual API call.
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return err
	}

	// Get the instance IDs to terminate
	runningInstances, err := r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return err
	}
	// Select the instance IDs to terminate based on count
	instanceIds := runningInstances[:count]
	logger.Info("Terminating", len(instanceIds), "CVM instances:", instanceIds)

	request := cvm.NewTerminateInstancesRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)

	// get response structure
	response, err := client.TerminateInstances(request)
	// API errors
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		log.Log.Error(err, "failed to terminate CVM instances")
		return err
	}
	// unexpected errors
	if err != nil {
		log.Log.Error(err, "failed to terminate CVM instances")
	}
	logger.Info("Response", response)
	logger.Info("Terminated", len(instanceIds), "CVM instances")

	// Get the instance IDs to terminate, for update status
	_, err = r.getRunningInstanceIDs(spotpool)
	if err != nil {
		return err
	}

	return nil
}

// createCVMClient creates a Tencent Cloud CVM client based on the Spotpool spec.
func (r *SpotpoolReconciler) createCVMClient(spec v1.SpotpoolSpec) (*cvm.Client, error) {
	credential := common.NewCredential(spec.SecretId, spec.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "POST"
	cpf.HttpProfile.ReqTimeout = 5
	//cpf.HttpProfile.Endpoint = "cvm.ap-guangzhou.tencentcloudapi.com"
	cpf.SignMethod = "HmacSHA1"

	client, err := cvm.NewClient(credential, spec.Region, cpf)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// getRunningInstanceIDs gets the IDs of running CVM instances that match the Spotpool configuration.
func (r *SpotpoolReconciler) getRunningInstanceIDs(spotpool *v1.Spotpool) ([]string, error) {
	// Implement your logic here to query the Tencent Cloud CVM API to get the IDs of running CVM instances that match the Spotpool configuration.
	// This example code assumes you have a functional CVM client and implement a basic search based on InstanceIds from the Spotpool spec. You might need to adjust the logic based on your actual implementation.
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeInstancesRequest()
	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	var instances []v1.Instances
	var runningInstanceIDs []string
	for _, instance := range response.Response.InstanceSet {
		if *instance.InstanceState == "RUNNING" || *instance.InstanceState == "PENDING" || *instance.InstanceState == "STARTING" {
			runningInstanceIDs = append(runningInstanceIDs, *instance.InstanceId)
		}
		// 检查实例的公网 IP，如果不存在公网 IP 则返回错误，下次重试
		if len(instance.PublicIpAddresses) == 0 {
			return nil, fmt.Errorf("instance %s does not have a public IP", *instance.InstanceId)
		}
		instances = append(instances, v1.Instances{
			InstanceId: *instance.InstanceId,
			PublicIp:   *instance.PublicIpAddresses[0],
		})
	}
	// update status with runningInstanceIDs
	spotpool.Status.Instances = instances
	err = r.Status().Update(context.Background(), spotpool)
	if err != nil {
		return nil, err
	}
	return runningInstanceIDs, nil
}

// countRunningInstances gets the number of CVMs running in the specified Spotpool's configuration.
func (r *SpotpoolReconciler) countRunningInstances(spotpool *v1.Spotpool) (int32, error) {
	// Implement your logic here to query the Tencent Cloud CVM API and return the number of running instances that match the Spotpool configuration (e.g., Region, InstanceType, etc.)
	// This example code assumes you have a functional CVM client and implement a basic search based on InstanceIds from the Spotpool spec. You might need to adjust the logic based on your actual implementation.
	client, err := r.createCVMClient(spotpool.Spec)
	if err != nil {
		return 0, err
	}
	request := cvm.NewDescribeInstancesRequest()
	response, err := client.DescribeInstances(request)
	if err != nil {
		return 0, err
	}
	var runningCount int32
	for _, instance := range response.Response.InstanceSet {
		logger.Info("Instance ", *instance.InstanceId, " State ", *instance.InstanceState)
		if *instance.InstanceState == "RUNNING" || *instance.InstanceState == "PENDING" || *instance.InstanceState == "STARTING" {
			runningCount++
		}
	}
	logger.Info("Found ", runningCount, " running instances")
	return runningCount, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotpoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsgeektimev1.Spotpool{}).
		Complete(r)
}
