package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

type Doubao struct {
	model *ark.ChatModel
	ctx   context.Context
}

func NewDoubao() *Doubao {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	model, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("MODEL"),
	})
	if err != nil {
		panic(err)
	}
	return &Doubao{
		model: model,
		ctx:   ctx,
	}
}

func (o *Doubao) SendMessage(messages []*schema.Message) (string, error) {

	input := []*schema.Message{
		schema.SystemMessage(messages[0].Content),
		schema.UserMessage(messages[1].Content),
	}
	content, err := generate(o.ctx, o.model, input)
	if err != nil {
		panic(err)
	}

	// stream(o.ctx, o.model, input)
	return content, nil
}

func stream(ctx context.Context, model *ark.ChatModel, input []*schema.Message) { //流式输出
	reader, err := model.Stream(ctx, input)
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	// 处理流式内容
	for {
		chunk, err := reader.Recv()
		if err != nil {
			break
		}
		fmt.Print(chunk.Content)
	}
}

func generate(ctx context.Context, model *ark.ChatModel, input []*schema.Message) (string, error) {
	resp, err := model.Generate(ctx, input)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// 暴露原模型给 Agent 使用
func (d *Doubao) GetModel() model.ToolCallingChatModel {
	return d.model
}
