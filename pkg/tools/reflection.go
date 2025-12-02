package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/barekit/talos/pkg/llm"
)

// Tool represents a tool that can be used by the agent.
type Tool struct {
	Name        string
	Description string
	Function    interface{}
	Definition  llm.ToolDefinition
}

// New creates a new Tool from a function.
// The function must take exactly one argument, which must be a struct (or pointer to struct).
// The struct fields should have `json` tags for names and `description` tags for descriptions.
// The function must return (string, error) or just error.
func New(name string, description string, fn interface{}) (*Tool, error) {
	def, err := generateDefinition(name, description, fn)
	if err != nil {
		return nil, err
	}

	return &Tool{
		Name:        name,
		Description: description,
		Function:    fn,
		Definition:  *def,
	}, nil
}

// Call executes the tool with the given arguments (JSON string).
func (t *Tool) Call(argsJSON string) (string, error) {
	fnVal := reflect.ValueOf(t.Function)
	fnType := fnVal.Type()

	// Create the argument struct
	argType := fnType.In(0)
	isPtr := false
	if argType.Kind() == reflect.Ptr {
		argType = argType.Elem()
		isPtr = true
	}

	argVal := reflect.New(argType)

	// Unmarshal JSON into the struct
	if err := json.Unmarshal([]byte(argsJSON), argVal.Interface()); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Call the function
	var args []reflect.Value
	if isPtr {
		args = []reflect.Value{argVal}
	} else {
		args = []reflect.Value{argVal.Elem()}
	}

	results := fnVal.Call(args)

	// Handle return values
	// Expected: (string, error) or (error)
	var output string
	var err error

	if len(results) == 1 {
		// (error)
		if !results[0].IsNil() {
			err = results[0].Interface().(error)
		}
	} else if len(results) == 2 {
		// (string, error)
		output = results[0].String()
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}
	} else {
		return "", fmt.Errorf("unexpected number of return values: %d", len(results))
	}

	return output, err
}

func generateDefinition(name, description string, fn interface{}) (*llm.ToolDefinition, error) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a function, got %s", t.Kind())
	}

	if t.NumIn() != 1 {
		return nil, fmt.Errorf("function must have exactly one argument")
	}

	argType := t.In(0)
	if argType.Kind() == reflect.Ptr {
		argType = argType.Elem()
	}
	if argType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("function argument must be a struct or pointer to struct")
	}

	properties := make(map[string]interface{})
	required := []string{}

	for i := 0; i < argType.NumField(); i++ {
		field := argType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}
		// Handle "name,omitempty"
		parts := strings.Split(jsonTag, ",")
		fieldName := parts[0]

		descTag := field.Tag.Get("description")

		prop := map[string]interface{}{
			"type": goTypeToJSONType(field.Type),
		}
		if descTag != "" {
			prop["description"] = descTag
		}

		properties[fieldName] = prop
		required = append(required, fieldName)
	}

	params := map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}

	return &llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
	}, nil
}

func goTypeToJSONType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		return "string" // Fallback
	}
}
