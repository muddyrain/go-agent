package tool

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]registeredTool
}

type registeredTool struct {
	tool   Tool
	schema *jsonschema.Schema
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]registeredTool),
	}
}

func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool is required")
	}

	definition := tool.Definition()
	name := strings.TrimSpace(definition.Name)

	if name == "" {
		return fmt.Errorf("tool name is required")
	}

	schema, err := compileSchema(
		name,
		definition.Parameters,
	)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf(
			"tool %q is already registered",
			name,
		)
	}

	r.tools[name] = registeredTool{
		tool:   tool,
		schema: schema,
	}

	return nil
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
		definition.Parameters = cloneRawMessage(
			definition.Parameters,
		)

		definitions = append(
			definitions,
			definition,
		)
	}

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
