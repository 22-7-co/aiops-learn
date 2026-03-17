/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"cobra-demo/cmd/utils"
	"context"
	"fmt"

	"bytes"

	"github.com/cloudwego/eino/schema"
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
			fmt.Printf("Failed to get pod events and logs: %v\n", err)
			return
		}
		fmt.Println(eventLog)
	},
}

func getPodEventsAndLogs() (map[string][]string, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)

	// 获取 Warning 级别事件
	events, err := clientGo.Clientset.CoreV1().Events("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, event := range events.Items {
		podName := event.InvolvedObject.Name
		podNamespace := event.InvolvedObject.Namespace
		message := event.Message
		if event.Type == "Pod" {
			logOption := &corev1.PodLogOptions{}
			req := clientGo.Clientset.CoreV1().Pods(podNamespace).GetLogs(podName, logOption)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				return nil, err
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(podLogs)
			if err != nil {
				continue
			}
			result[podName] = append(result[podName], fmt.Sprintf("Event: %s", message))
			result[podName] = append(result[podName], fmt.Sprintf("Namespace: %s", namespace))
			result[podName] = append(result[podName], fmt.Sprintf("Logs: %s", buf.String()))
		}
	}
	return result, nil
}

// 把日志发过ai，给出建议
func sendLogsToAI(logs map[string][]string) (string, error) {
	client := utils.NewDoubao()

	// 拼接所有 Pod 事件和日志
	combinedInfo := "找到以下 Pod Warning 事件和日志：\n"
	for podName, logs := range logs {
		combinedInfo += fmt.Sprintf("Pod: %s\n", podName)
		for _, log := range logs {
			combinedInfo += fmt.Sprintf("Logs: %s\n", log)
		}
		combinedInfo += "\n"
	}

	fmt.Println(combinedInfo)

	// 构造 ai 请求信息
	message := []*schema.Message{
		{Role: schema.System, Content: "你是一个 Kubernetes 专家，请根据以下日志给出建议"},
		{Role: schema.User, Content: fmt.Sprintf("以下是多个Pod Event 事件和对应日志: \n%s, 请针对Pod Log 给出实质性，可操作的建议。优先选择高效不需要编写Yaml的解决办法", combinedInfo)},
	}
	response, err := client.SendMessage(message)
	if err != nil {
		return "", err
	}
	return response, nil
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
