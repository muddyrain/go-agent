package tool

import "go-agent/internal/llm"

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册一个工具
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get 按名称查找工具
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) ToLLMTools() []llm.Tool {
	result := make([]llm.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return result
}
