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
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/uuid"
	logv1 "github.com/lyzhang1999/rag-log-operator/api/v1"
)

// RagLogPilotReconciler reconciles a RagLogPilot object
type RagLogPilotReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	KubeClient *kubernetes.Clientset
}

//+kubebuilder:rbac:groups=log.aiops.com,resources=raglogpilots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=log.aiops.com,resources=raglogpilots/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=log.aiops.com,resources=raglogpilots/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RagLogPilot object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *RagLogPilotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ragLogPilot logv1.RagLogPilot
	if err := r.Get(ctx, req.NamespacedName, &ragLogPilot); err != nil {
		logger.Error(err, "unable to fetch ragLogPilot")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 检查是否有 ConversationId
	if ragLogPilot.Status.ConversationId == "" {
		// 如果没有则创建新的对话
		conversationId, err := r.createNewConversation(ragLogPilot)
		if err != nil {
			logger.Error(err, "failed to create new conversation")
			return ctrl.Result{}, err
		}

		// 保存 ConversationId 到 Status
		ragLogPilot.Status.ConversationId = conversationId
		if err := r.Status().Update(ctx, &ragLogPilot); err != nil {
			logger.Error(err, "failed to update status with ConversationId")
			return ctrl.Result{}, err
		}
	}

	// 获取目标 namespace 下的所有 Pod
	var pods corev1.PodList
	if err := r.List(ctx, &pods, &client.ListOptions{Namespace: req.Namespace}); err != nil {
		logger.Error(err, "failed to list pods")
		return ctrl.Result{}, err
	}

	// 遍历 Pod 并增量获取日志
	for _, pod := range pods.Items {
		logString, err := r.getPodLogs(pod)
		if err != nil {
			logger.Error(err, "failed to get pod logs", "pod", pod.Name)
			continue
		}
		var errorLog []string
		logLines := strings.Split(logString, "\n")
		for _, line := range logLines {
			if strings.Contains(line, "ERROR") {
				errorLog = append(errorLog, line)
			}
		}
		if len(errorLog) > 0 {
			combinedErrorLog := strings.Join(errorLog, "\n")
			fmt.Println("combinedErrorLog: ", combinedErrorLog)
			// 调用 RAG 系统解决方案
			answer, err := r.queryRagSystem(combinedErrorLog, ragLogPilot)
			if err != nil {
				logger.Error(err, "failed to query RAG system")
				return ctrl.Result{}, err
			}

			err = r.sendFeishuAlert("https://open.feishu.cn/open-apis/bot/v2/hook/d5e267dc-a92f-43d3-bc45-106b5e718c49", answer)
			if err != nil {
				fmt.Println(err)
			}
			logger.Info("RAG system response", "answer", answer)
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// 获取 Pod 日志
func (r *RagLogPilotReconciler) getPodLogs(pod corev1.Pod) (string, error) {
	tailLines := int64(20)
	logOptions := &corev1.PodLogOptions{TailLines: &tailLines}
	req := r.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, logOptions)
	// Stream the logs
	logStream, err := req.Stream(context.TODO())
	if err != nil {
		return "", err // Return the error if Stream fails
	}
	defer logStream.Close() // Ensure the stream is closed after reading

	// Read the logs from the stream
	var logBuffer bytes.Buffer
	if _, err := logBuffer.ReadFrom(logStream); err != nil {
		return "", err // Return the error if reading from the stream fails
	}

	return logBuffer.String(), nil // Return the logs as a string
}

// 创建新的 conversation
func (r *RagLogPilotReconciler) createNewConversation(ragLogPilot logv1.RagLogPilot) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/new_conversation?user_id=%s", ragLogPilot.Spec.RagFlowEndpoint, uuid.NewString()), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ragLogPilot.Spec.RagFlowToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result["data"].(map[string]interface{})["id"].(string), nil
}

// 查询 RAG 系统
func (r *RagLogPilotReconciler) queryRagSystem(podLog string, ragLogPilot logv1.RagLogPilot) (string, error) {
	payload := map[string]interface{}{
		"conversation_id": ragLogPilot.Status.ConversationId,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": fmt.Sprintf("以下是获取到的日志：%s，请基于运维知识库进行解答，如果你不知道，就说不知道", podLog),
			},
		},
		"stream": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/completion", ragLogPilot.Spec.RagFlowEndpoint), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ragLogPilot.Spec.RagFlowToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result["data"].(map[string]interface{})["answer"].(string), nil
}

// sendFeishuAlert 发送飞书告警
func (r *RagLogPilotReconciler) sendFeishuAlert(webhook, analysis string) error {
	// 飞书消息内容
	message := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": analysis,
		},
	}

	// 将消息内容序列化为 JSON
	messageBody, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", webhook, bytes.NewBuffer(messageBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// 发出请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Feishu alert, status code: %d", resp.StatusCode)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RagLogPilotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var kubeConfig *string
	var config *rest.Config

	if home := homedir.HomeDir(); home != "" {
		kubeConfig = flag.String("kubeConfig", filepath.Join(home, ".kube", "config"), "[可选] kubeconfig 绝对路径")
	}

	// Initialize the KubeClient
	config, err := rest.InClusterConfig()
	if err != nil {
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeConfig); err != nil {
			return err
		}
	}

	r.KubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&logv1.RagLogPilot{}).
		Complete(r)
}
