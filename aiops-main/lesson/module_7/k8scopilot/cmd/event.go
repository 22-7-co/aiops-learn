/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lyzhang1999/k8scopilot/utils"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// eventCmd represents the event command
var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		eventLog, err := getPodEventsAndLogs()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		result, err := sendToChatGPT(eventLog)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(result)
	},
}

// sendToChatGPT 函数接受 podInfo map，逐个元素发送给 OpenAI 的 ChatGPT 获取建议
func sendToChatGPT(podInfo map[string][]string) (string, error) {
	client, err := utils.NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("error creating OpenAI client: %v", err)
	}

	// 拼接所有 Pod 的事件和日志信息
	combinedInfo := "找到以下 Pod Waring 事件及其日志：\n\n"
	for podName, info := range podInfo {
		combinedInfo += fmt.Sprintf("Pod 名称: %s\n", podName)
		// 每个 Pod 的 event 和日志拼接成一个字符串
		for _, line := range info {
			combinedInfo += line + "\n"
		}
		combinedInfo += "\n" // 每个 Pod 信息之间加一个空行
	}

	fmt.Println(combinedInfo)

	//构造 ChatGPT 请求消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "您是一位 Kubernetes 专家，你要帮助用户诊断多个 Pod 问题。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("以下是多个 Pod Event 事件和对应的日志:\n%s\n请主要针对 Pod Log 给出实质性、可操作的建议", combinedInfo),
			// 更好的提示语
			//Content: fmt.Sprintf("以下是多个 Pod Event 事件和对应的日志:\n%s\n请主要针对 Pod Log 给出实质性、可操作的建议，优先给出不需要编写 YAML 的纯命令行操作方法，如果无法单纯通过命令行操作，再用 YAML 来解决问题。禁止废话，Event 事件仅做参考。", combinedInfo),
		},
	}

	// 请求 ChatGPT 获取建议
	resp, err := client.Client.CreateChatCompletion(
		context.TODO(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4oMini,
			Messages: messages,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error calling OpenAI API: %v", err)
	}

	responseText := resp.Choices[0].Message.Content
	return responseText, nil
}

func getPodEventsAndLogs() (map[string][]string, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	// map[string] 切片，切片用来存储每个 Pod 的事件和日志信息
	result := make(map[string][]string)
	// 获取 Warning 级别的事件
	events, err := clientGo.Clientset.CoreV1().Events("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, fmt.Errorf("error getting events: %v", err)
	}

	for _, event := range events.Items {
		podName := event.InvolvedObject.Name
		namespace := event.InvolvedObject.Namespace
		message := event.Message

		// 获取 Pod 的日志
		if event.InvolvedObject.Kind == "Pod" {
			logOptions := &corev1.PodLogOptions{}
			req := clientGo.Clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				continue
			}
			defer podLogs.Close()

			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(podLogs)
			if err != nil {
				continue
			}
			// 只存有日志的 Pod，否则单纯靠 event 信息无法给出建议
			// 将事件信息存入 map
			result[podName] = append(result[podName], fmt.Sprintf("Event Message: %s", message))
			result[podName] = append(result[podName], fmt.Sprintf("Namespace: %s", namespace))
			// 将日志信息存入 map
			result[podName] = append(result[podName], fmt.Sprintf("Logs:\n%s", buf.String()))
		}
	}

	return result, nil
}

func init() {
	analyzeCmd.AddCommand(eventCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// eventCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// eventCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
