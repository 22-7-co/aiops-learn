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
	"net/url"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	logv1 "github.com/lyzhang1999/llm-log-operator/api/v1"
	openai "github.com/sashabaranov/go-openai"
)

// LogPilotReconciler reconciles a LogPilot object
type LogPilotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=log.aiops.com,resources=logpilots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=log.aiops.com,resources=logpilots/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=log.aiops.com,resources=logpilots/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LogPilot object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *LogPilotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var logPilot logv1.LogPilot
	if err := r.Get(ctx, req.NamespacedName, &logPilot); err != nil {
		logger.Error(err, "unable to fetch LogPilot")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 计算查询的时间范围
	currentTime := time.Now().Unix()
	preTimeStamp := logPilot.Status.PreTimeStamp
	// print preTimeStamp for debugging
	fmt.Printf("preTimeStamp: %s\n", preTimeStamp)
	var preTime int64
	if preTimeStamp == "" {
		preTime = currentTime - 5
	} else {
		preTime, _ = strconv.ParseInt(preTimeStamp, 10, 64)
	}

	// Loki 查询
	lokiQuery := logPilot.Spec.LokiPromQL
	// 纳秒级时间戳
	endTime := currentTime * 1000000000     // 当前时间的纳秒级时间戳
	startTime := (preTime - 5) * 1000000000 // 上次的时间戳

	fmt.Printf("startTime: %d, endTime: %d\n", startTime, endTime)

	if startTime >= endTime {
		logger.Info("startTime is greater than or equal to endTime, skipping log query")
		// print startTime and endTime for debugging
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	startTimeForUpdate := currentTime
	lokiURL := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&start=%d&end=%d",
		logPilot.Spec.LokiURL, url.QueryEscape(lokiQuery), startTime, endTime)
	fmt.Println("lokiURL: ", lokiURL)
	lokiLogs, err := r.queryLoki(lokiURL)
	fmt.Println(lokiLogs)
	if err != nil {
		logger.Error(err, "unable to query Loki")
		return ctrl.Result{}, err
	}

	// 如果有日志结果，调用 LLM 进行分析
	if lokiLogs != "" {
		fmt.Println("send logs to LLM")
		analysisResult, err := r.analyzeLogsWithLLM(logPilot.Spec.LLMEndpoint, logPilot.Spec.LLMToken, logPilot.Spec.LLMModel, lokiLogs)
		if err != nil {
			logger.Error(err, "unable to analyze logs with LLM")
			return ctrl.Result{}, err
		}

		// 如果 LLM 返回的结果表明日志有问题，发送飞书告警
		if analysisResult.HasErrors {
			err := r.sendFeishuAlert(logPilot.Spec.FeishuWebhook, analysisResult.Analysis)
			if err != nil {
				logger.Error(err, "unable to send alert to Feishu")
				return ctrl.Result{}, err
			}
		}
	}

	// 更新状态中的 PreTimeStamp
	logPilot.Status.PreTimeStamp = fmt.Sprintf("%d", startTimeForUpdate)
	if err := r.Status().Update(ctx, &logPilot); err != nil {
		logger.Error(err, "unable to update LogPilot status")
		return ctrl.Result{}, err
	}

	// 10 秒后再次 Reconcile
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// queryLoki 从 Loki 获取日志
func (r *LogPilotReconciler) queryLoki(lokiURL string) (string, error) {
	resp, err := http.Get(lokiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println(string(body))

	var lokiResponse map[string]interface{}
	err = json.Unmarshal(body, &lokiResponse)
	if err != nil {
		return "", err
	}

	data, ok := lokiResponse["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid Loki response data format")
	}

	// 检查 result 是否为空
	result, ok := data["result"].([]interface{})
	if !ok || len(result) == 0 {
		return "", nil
	}

	return string(body), nil
}

// LLMAnalysisResult 用于存储 LLM 分析的结果
type LLMAnalysisResult struct {
	HasErrors bool   // 是否有错误日志
	Analysis  string // LLM 返回的分析结果
}

// analyzeLogsWithLLM 调用 LLM 接口分析日志
func (r *LogPilotReconciler) analyzeLogsWithLLM(endpoint, token, model, logs string) (*LLMAnalysisResult, error) {
	config := openai.DefaultConfig(token)
	config.BaseURL = endpoint
	client := openai.NewClientWithConfig(config)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("你现在是一名运维专家，以下日志是从日志系统里获取的日志，请分析日志的错误等级，如果遇到严重的问题，例如请求外部系统失败、外部系统故障、致命故障、数据库连接错误等严重问题时，请给出简短的建议，对于你认为严重需要通知运营人员的，请在返回内容里增加[feishu]标识:\n%s", logs),
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return nil, err
	}

	return parseLLMResponse(&resp), nil
}

// parseLLMResponse 解析 LLM API 的响应
func parseLLMResponse(resp *openai.ChatCompletionResponse) *LLMAnalysisResult {
	result := &LLMAnalysisResult{
		Analysis: resp.Choices[0].Message.Content, // 从 LLM 返回的文本中获取分析结果
	}

	// 简单判断分析结果是否包含错误的标识符
	if strings.Contains(strings.ToLower(result.Analysis), "feishu") {
		result.HasErrors = true
	} else {
		result.HasErrors = false
	}

	return result
}

// sendFeishuAlert 发送飞书告警
func (r *LogPilotReconciler) sendFeishuAlert(webhook, analysis string) error {
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
func (r *LogPilotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&logv1.LogPilot{}).
		Complete(r)
}
