package tools

import (
	"fmt"
)

type ToolType string

type Tool struct {
	Type     string   `json:"type"` // always "function"
	Function Function `json:"function"`
}

type ToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"` // JSON-encoded string
}

type Function struct {
	Name        ToolType           `json:"name"`
	Description string             `json:"description"`
	Parameters  FunctionParameters `json:"parameters"`
}

type FunctionParameters struct {
	Type       string                       `json:"type"` // e.g., "object"
	Properties map[string]FunctionParameter `json:"properties"`
	Required   []string                     `json:"required,omitempty"`
}

type FunctionParameter struct {
	Type        string `json:"type"` // e.g., "string", "integer"
	Description string `json:"description"`
}

type ToolCallEvent struct {
	Name   string
	Args   string
	Result string // filled after execution
}

type ITool interface {
	Name() string
	Execute(name, argsJSON string) (string, error)
	Tool() Tool
}

/// Registry and execution

var toolsRegistry = map[ToolType]ITool{}

func AvailableTools() []Tool {
	tools := make([]Tool, 0, len(toolsRegistry))
	for _, tool := range toolsRegistry {
		tools = append(tools, tool.Tool())
	}
	return tools
}

func registerTool(tool ITool) {
	toolsRegistry[ToolType(tool.Name())] = tool
}

func ExecuteTool(name, argsJSON string) (string, error) {
	if tool, ok := toolsRegistry[ToolType(name)]; ok {
		return tool.Execute(name, argsJSON)
	}
	return "", fmt.Errorf("tool '%s' not found", name)
}
