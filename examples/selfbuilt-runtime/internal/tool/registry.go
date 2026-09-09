package tool

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Registry 并发安全地维护工具及其已编译 Schema；读多写少场景使用 RWMutex。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]registeredTool
}

// registeredTool 将执行实现和注册时编译好的 Schema 绑定，避免每次调用重复编译。
type registeredTool struct {
	tool   Tool
	schema *jsonschema.Schema
}

// preparedTool 保存已经完成名称校验和 Schema 编译、
// 但尚未写入 Registry 的工具。
type preparedTool struct {
	name       string
	registered registeredTool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]registeredTool),
	}
}

func (r *Registry) Register(candidate Tool) error {
	return r.RegisterBatch(candidate)
}

func (r *Registry) Get(name string) (Tool, bool) {
	name = strings.TrimSpace(name)

	r.mu.RLock()
	defer r.mu.RUnlock()

	registered, ok := r.tools[name]
	if !ok {
		return nil, false
	}
	return registered.tool, true
}

func (r *Registry) Definitions() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]Definition, 0, len(r.tools))

	for _, registered := range r.tools {
		definition := registered.tool.Definition()
		// 返回副本，避免调用者通过 Parameters 修改 Registry 持有的定义。
		definition.Parameters = cloneRawMessage(
			definition.Parameters,
		)

		definitions = append(
			definitions,
			definition,
		)
	}

	// map 遍历顺序不稳定，按名称排序以保证模型请求和测试结果可复现。
	sort.Slice(
		definitions,
		func(i, j int) bool {
			return definitions[i].Name < definitions[j].Name
		},
	)

	return definitions
}

func cloneRawMessage(message json.RawMessage) json.RawMessage {
	if message == nil {
		return nil
	}

	cloned := make(json.RawMessage, len(message))
	copy(cloned, message)

	return cloned
}

func compileSchema(
	name string,
	parameters json.RawMessage,
) (*jsonschema.Schema, error) {
	var schemaDocument any

	if err := json.Unmarshal(
		parameters,
		&schemaDocument,
	); err != nil {
		return nil, fmt.Errorf(
			"decode tool %q parameter schema: %w",
			name,
			err,
		)
	}

	compiler := jsonschema.NewCompiler()

	schemaURL := fmt.Sprintf(
		"urn:agenthub:tool:%s",
		name,
	)

	if err := compiler.AddResource(
		schemaURL,
		schemaDocument,
	); err != nil {
		return nil, fmt.Errorf(
			"add tool %q parameter schema: %w",
			name,
			err,
		)
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf(
			"compile tool %q parameter schema: %w",
			name,
			err,
		)
	}

	return schema, nil
}

func (r *Registry) Validate(
	name string,
	arguments json.RawMessage,
) error {
	name = strings.TrimSpace(name)

	r.mu.RLock()
	registered, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf(
			"tool %q is not registered",
			name,
		)
	}

	var value any

	if err := json.Unmarshal(arguments, &value); err != nil {
		return fmt.Errorf(
			"decode tool %q arguments: %w",
			name,
			err,
		)
	}

	if err := registered.schema.Validate(value); err != nil {
		return fmt.Errorf(
			"tool %q arguments do not match schema: %w",
			name,
			err,
		)
	}

	return nil
}

func (r *Registry) RegisterBatch(candidates ...Tool) error {
	if len(candidates) == 0 {
		return fmt.Errorf(
			"at least one tool is required",
		)
	}

	prepared := make(
		[]preparedTool,
		0,
		len(candidates),
	)

	batchNames := make(
		map[string]struct{},
		len(candidates),
	)

	// 第一阶段：校验并准备所有工具。
	for _, candidate := range candidates {
		if candidate == nil {
			return fmt.Errorf(
				"tool is required",
			)
		}

		definition := candidate.Definition()
		name := strings.TrimSpace(
			definition.Name,
		)

		if name == "" {
			return fmt.Errorf(
				"tool name is required",
			)
		}

		if _, exists := batchNames[name]; exists {
			return fmt.Errorf(
				"tool %q is duplicated in batch",
				name,
			)
		}

		schema, err := compileSchema(
			name,
			definition.Parameters,
		)
		if err != nil {
			return err
		}

		batchNames[name] = struct{}{}

		prepared = append(
			prepared,
			preparedTool{
				name: name,
				registered: registeredTool{
					tool:   candidate,
					schema: schema,
				},
			},
		)
	}
	// 第二阶段：获取写锁，检查 Registry 中是否已存在同名工具。
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range prepared {
		if _, exists := r.tools[item.name]; exists {
			return fmt.Errorf(
				"tool %q is already registered",
				item.name,
			)
		}
	}
	// 第三阶段：所有检查通过后，一次性写入。
	for _, item := range prepared {
		r.tools[item.name] = item.registered
	}
	return nil
}
