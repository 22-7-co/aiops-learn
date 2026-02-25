/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"cobra-demo/cmd/utils"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	skillsmw "github.com/dyike/eino-skills/pkg/middleware"
	skillpkg "github.com/dyike/eino-skills/pkg/skill"
	skilltools "github.com/dyike/eino-skills/pkg/tools"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	schema2 "k8s.io/apimachinery/pkg/runtime/schema"
	scheme3 "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
)

// chataiCmd represents the chatai command
var chataiCmd = &cobra.Command{
	Use:   "chatai",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		startChat()
	},
}

func startChat() {
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()
	// 1. 初始化 Skills (加载器 & 注册表)
	loader := skillpkg.NewLoader(
		skillpkg.WithGlobalSkillsDir("./cmd/skills"), // 指向实际的 skills 目录
	)
	registry := skillpkg.NewRegistry(loader)
	if err := registry.Initialize(ctx); err != nil {
		panic(err)
	}
	// 2. 创建 Skills 中间件
	skillsMiddleware := skillsmw.NewSkillsMiddleware(registry)

	// 3. 准备 Tools (基础 Skill 工具 + 终端执行能力的工具)
	tools := skilltools.NewSkillTools(registry) // 包含 list_skills, view_skill
	cwd, _ := os.Getwd()
	tools = append(tools, skilltools.NewRunTerminalCommandTool(cwd))
	tools = append(tools, functionCalling()...)

	fmt.Printf("=== Tools 数量: %d ===\n", len(tools))
	for i, t := range tools {
		info, _ := t.Info(ctx)
		fmt.Printf("%d. %s: %s\n", i+1, info.Name, info.Desc)
	}

	// 4. 注入 System Prompt (包含 Skills 使用规范)
	basePrompt := `你现在是一个K8s助手,你要帮用户完成任务`
	systemPrompt := skillsMiddleware.InjectPrompt(basePrompt)
	client := utils.NewDoubao()

	// 5. 创建 Agent
	myAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: client.GetModel(),
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: 50, // 增加步数限制以支持多步骤 Skill
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("你是 K8s Copilot, 请问有什么可以帮你?")
	for {
		fmt.Print(">")
		if scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				fmt.Println("再见!")
				break
			}
			if input == "" {
				continue
			}
			// 7. 运行 Agent
			// 实际使用建议使用 Stream 模式
			resp, err := myAgent.Generate(ctx, []*schema.Message{
				{Role: schema.System, Content: systemPrompt},
				{Role: schema.User, Content: input},
			})

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			fmt.Println(resp.Content)
		}
	}
}

func functionCalling() []tool.BaseTool {
	// 根据 K8s YAML 部署资源
	f1 := einoutils.NewTool(
		&schema.ToolInfo{
			Name: "DeployResource",
			Desc: "根据 K8s YAML 部署资源",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"yamlContent": {
					Desc:     "K8s YAML",
					Type:     schema.String,
					Required: true,
				},
			}),
		},
		func(ctx context.Context, input *struct {
			YamlContent string `json:"yamlContent"`
		}) (string, error) {
			// fmt.Println("【F1 被调用了】")
			clientGo, err := utils.NewClientGo("~/.kube/config")
			if err != nil {
				return "", err
			}
			resources, err := restmapper.GetAPIGroupResources(clientGo.DiscoveryClient)
			if err != nil {
				return "", err
			}
			//  把 Yaml 转换为 Unstructured 对象
			unstructuredObj := &unstructured.Unstructured{}
			_, _, err = scheme3.Codecs.UniversalDeserializer().Decode([]byte(input.YamlContent), nil, unstructuredObj)
			if err != nil {
				return "", err
			}
			// 创建 mapper
			mapper := restmapper.NewDiscoveryRESTMapper(resources)
			// 从 unstructured 对象中获取 API 组和版本
			gvk := unstructuredObj.GroupVersionKind()
			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return "", err
			}
			namespace := unstructuredObj.GetNamespace()
			if namespace == "" {
				namespace = "default"
			}
			// 使用 dynamic client 创建资源
			dynamicClient := clientGo.DynamicClient.Resource(mapping.Resource).Namespace(namespace)
			_, err = dynamicClient.Create(ctx, unstructuredObj, metav1.CreateOptions{})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("资源 %s/%s 创建成功", namespace, unstructuredObj.GetName()), nil
		},
	)
	// 查询 K8s 资源
	f2 := einoutils.NewTool(
		&schema.ToolInfo{
			Name: "queryK8sResource",
			Desc: "查询 K8s 资源",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"namespace": {
					Desc:     "所查询资源所在的命名空间",
					Type:     schema.String,
					Required: true,
				},
				"resourceType": {
					Desc:     "所查询K8s资源的标准类型, 如 Deployment、Pod、Service 等",
					Type:     schema.String,
					Required: true,
				},
			}),
		},
		func(ctx context.Context, input *struct {
			Namespace    string `json:"namespace"`
			ResourceType string `json:"resourceType"`
		}) (string, error) {
			clientGo, err := utils.NewClientGo("~/.kube/config")
			if err != nil {
				return "", err
			}
			resourceType := strings.ToLower(input.ResourceType)
			var gvr schema2.GroupVersionResource
			switch resourceType {
			case "deployment":
				gvr = schema2.GroupVersionResource{
					Group:    "apps",
					Version:  "v1",
					Resource: "deployments",
				}
			case "service":
				gvr = schema2.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "services",
				}
			case "pod":
				gvr = schema2.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				}
			case "ingress":
				gvr = schema2.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "ingresses",
				}
			default:
				return "", fmt.Errorf("不支持的资源类型: %s", resourceType)
			}
			dynamicClient := clientGo.DynamicClient.Resource(gvr).Namespace(input.Namespace)
			resources, err := dynamicClient.List(ctx, metav1.ListOptions{})
			if err != nil {
				return "", err
			}
			result := ""
			for _, resource := range resources.Items {
				result += fmt.Sprintf("%s/%s\n", resourceType, resource.GetName())
			}
			return result, nil
		},
	)
	//删除 K8s 资源
	f3 := einoutils.NewTool(
		&schema.ToolInfo{
			Name: "deleteK8sResource",
			Desc: "删除 K8s 资源",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"namespace": {
					Desc:     "所删除资源所在的命名空间",
					Type:     schema.String,
					Required: true,
				},
				"resourceType": {
					Desc:     "所删除K8s资源的标准类型, 如 Deployment、Pod、Service 等",
					Type:     schema.String,
					Required: true,
				},
				"resourceName": {
					Desc:     "所删除K8s资源的名称",
					Type:     schema.String,
					Required: true,
				},
			}),
		},
		func(ctx context.Context, input *struct {
			Namespace    string `json:"namespace"`
			ResourceType string `json:"resourceType"`
			ResourceName string `json:"resourceName"`
		}) (string, error) {
			// fmt.Println("【F3 被调用了】")
			clientGo, err := utils.NewClientGo("~/.kube/config")
			if err != nil {
				return "", err
			}
			resourceType := strings.ToLower(input.ResourceType)
			var gvr schema2.GroupVersionResource
			switch resourceType {
			case "deployment":
				gvr = schema2.GroupVersionResource{
					Group:    "apps",
					Version:  "v1",
					Resource: "deployments",
				}
			case "service":
				gvr = schema2.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "services",
				}
			case "pod":
				gvr = schema2.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				}
			case "ingress":
				gvr = schema2.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "ingresses",
				}
			default:
				return "", fmt.Errorf("不支持的资源类型: %s", resourceType)
			}
			dynamicClient := clientGo.DynamicClient.Resource(gvr).Namespace(input.Namespace)
			err = dynamicClient.Delete(ctx, input.ResourceName, metav1.DeleteOptions{})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("资源 %s/%s/%s 删除成功", input.Namespace, resourceType, input.ResourceName), nil
		},
	)
	return []tool.BaseTool{f1, f2, f3}
}
func init() {
	askCmd.AddCommand(chataiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// chataiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// chataiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
