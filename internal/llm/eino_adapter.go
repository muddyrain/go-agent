package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EinoModelAdapter 把 Eino 的 OpenAI ChatModel 包装成我们自己的 llm.Model 接口。
// 这是适配器模式：Eino 的模型接口和我们自己的不一样，用适配器统一成我们的接口。
type EinoModelAdapter struct {
	einoModel *openai.ChatModel
}

// NewEinoModelAdapter 创建适配器，内部用 Eino 的 OpenAI 模型。
func NewEinoModelAdapter(ctx context.Context, baseURL, apiKey, modelName string) (*EinoModelAdapter, error) {
	einoModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create eino model: %w", err)
	}
	return &EinoModelAdapter{einoModel: einoModel}, nil
}

// Chat 实现我们自己的 llm.Model 接口。
// 把我们的消息转成 Eino 的消息，调用 Eino 的 Generate，再把结果转回来。
func (a *EinoModelAdapter) Chat(ctx context.Context, messages []Message, tools []Tool) (Message, error) {
	// 1. 我们的消息 → Eino 的消息
	einoMsgs := toEinoMessages(messages)

	// 2. 构造 options：如果有工具，加上 WithTools
	var opts []model.Option
	if len(tools) > 0 {
		einoTools := toEinoTools(tools)
		opts = append(opts, model.WithTools(einoTools))
	}

	// 3. 调用 Eino 的 Generate，传入 options
	einoMsg, err := a.einoModel.Generate(ctx, einoMsgs, opts...)
	if err != nil {
		return Message{}, fmt.Errorf("eino generate: %w", err)
	}

	// 4. Eino 的消息 → 我们的消息
	return toOurMessage(einoMsg), nil
}

// ChatStream 实现流式接口（llm.Model 接口要求）。
// 暂时不实现流式，返回错误。后续可以用 Eino 的 Stream 方法实现。
func (a *EinoModelAdapter) ChatStream(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("stream not implemented in eino adapter yet")
}

// toEinoMessages 把我们的 []Message 转成 Eino 的 []*schema.Message。
func toEinoMessages(msgs []Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(msgs))
	for _, m := range msgs {
		einoMsg := &schema.Message{
			Role:       schema.RoleType(m.Role), // string → RoleType（本质也是 string）
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		// 转换 ToolCalls
		for _, tc := range m.ToolCalls {
			einoMsg.ToolCalls = append(einoMsg.ToolCalls, schema.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: schema.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		result = append(result, einoMsg)
	}
	return result
}

// toOurMessage 把 Eino 的 *schema.Message 转成我们的 Message。
func toOurMessage(m *schema.Message) Message {
	ourMsg := Message{
		Role:       string(m.Role), // RoleType → string
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
	}
	for _, tc := range m.ToolCalls {
		ourMsg.ToolCalls = append(ourMsg.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return ourMsg
}

// toEinoTools 把我们的 []Tool 转成 Eino 的 []*schema.ToolInfo。
func toEinoTools(tools []Tool) []*schema.ToolInfo {
	result := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		toolInfo := &schema.ToolInfo{
			Name: t.Function.Name,
			Desc: t.Function.Description,
		}
		// 转换参数 schema
		if len(t.Function.Parameters) > 0 {
			toolInfo.ParamsOneOf = jsonSchemaToParams(t.Function.Parameters)
		}
		result = append(result, toolInfo)
	}
	return result
}

// jsonSchemaToParams 把 JSON Schema 原始字节转成 Eino 的 *schema.ParamsOneOf。
// 只处理简单的 JSON Schema（type、properties、description、required），
// 我们的工具参数都比较简单，这个转换足够用了。
func jsonSchemaToParams(raw json.RawMessage) *schema.ParamsOneOf {
	var js struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(raw, &js); err != nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range js.Required {
		requiredSet[r] = true
	}

	params := make(map[string]*schema.ParameterInfo)
	for name, propRaw := range js.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			continue
		}
		params[name] = &schema.ParameterInfo{
			Type:     schema.DataType(prop.Type),
			Desc:     prop.Description,
			Required: requiredSet[name],
		}
	}

	return schema.NewParamsOneOfByParams(params)
}
