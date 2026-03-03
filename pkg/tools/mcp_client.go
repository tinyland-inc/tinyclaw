package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tinyland-inc/tinyclaw/pkg/logger"
)

// MCPClientTool bridges an MCP server subprocess into TinyClaw's tool registry.
// It spawns the MCP server process, discovers tools via tools/list, and proxies
// Execute calls through JSON-RPC 2.0 stdin/stdout.
type MCPClientTool struct {
	name        string
	description string
	params      map[string]any

	// Subprocess management
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex

	// JSON-RPC request ID counter
	nextID atomic.Int64

	// Discovered MCP tools cached after initialize
	mcpTools map[string]MCPToolInfo
}

// MCPToolInfo describes a single tool discovered from the MCP server.
type MCPToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// jsonRPCRequest is the JSON-RPC 2.0 request envelope.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCResponse is the JSON-RPC 2.0 response envelope.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewMCPClientTool creates a meta-tool that proxies calls to an MCP server.
// The tool_name argument selects which MCP sub-tool to invoke.
func NewMCPClientTool(name, description string) *MCPClientTool {
	return &MCPClientTool{
		name:        name,
		description: description,
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tool_name": map[string]any{
					"type":        "string",
					"description": "Name of the MCP tool to call",
				},
				"arguments": map[string]any{
					"type":        "object",
					"description": "Arguments for the MCP tool",
				},
			},
			"required": []string{"tool_name"},
		},
		mcpTools: make(map[string]MCPToolInfo),
	}
}

func (t *MCPClientTool) Name() string               { return t.name }
func (t *MCPClientTool) Description() string        { return t.description }
func (t *MCPClientTool) Parameters() map[string]any { return t.params }

// Start spawns the MCP server process and performs the initialize handshake.
func (t *MCPClientTool) Start(ctx context.Context, command string, args []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd := exec.CommandContext(ctx, command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcp stdin pipe: %w", err)
	}
	stdout, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return fmt.Errorf("mcp stdout pipe: %w", pipeErr)
	}

	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("mcp start: %w", startErr)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.stdout = bufio.NewReader(stdout)

	// Initialize handshake
	initResp, err := t.call("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "tinyclaw",
			"version": "1.0.0",
		},
	})
	if err != nil {
		t.stop()
		return fmt.Errorf("mcp initialize: %w", err)
	}

	logger.InfoCF("mcp", "MCP server initialized", map[string]any{
		"server": t.name,
		"result": string(initResp),
	})

	// Send initialized notification (no response expected)
	_ = t.notify("notifications/initialized", nil)

	// Discover tools
	if err := t.discoverTools(); err != nil {
		t.stop()
		return fmt.Errorf("mcp discover tools: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the MCP server subprocess.
func (t *MCPClientTool) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stop()
}

func (t *MCPClientTool) stop() {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_ = t.cmd.Wait()
	}
	t.cmd = nil
	t.stdin = nil
	t.stdout = nil
}

// DiscoveredTools returns the list of tools discovered from the MCP server.
func (t *MCPClientTool) DiscoveredTools() map[string]MCPToolInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make(map[string]MCPToolInfo, len(t.mcpTools))
	maps.Copy(result, t.mcpTools)
	return result
}

func (t *MCPClientTool) discoverTools() error {
	result, err := t.call("tools/list", nil)
	if err != nil {
		return err
	}

	var toolsResp struct {
		Tools []MCPToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsResp); err != nil {
		return fmt.Errorf("parse tools/list: %w", err)
	}

	t.mcpTools = make(map[string]MCPToolInfo, len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		t.mcpTools[tool.Name] = tool
	}

	logger.InfoCF("mcp", "Discovered MCP tools", map[string]any{
		"server": t.name,
		"count":  len(t.mcpTools),
	})

	return nil
}

// Execute proxies a tools/call to the MCP server.
func (t *MCPClientTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	toolName, ok := args["tool_name"].(string)
	if !ok || toolName == "" {
		return ErrorResult("tool_name is required and must be a non-empty string")
	}

	t.mu.Lock()
	_, known := t.mcpTools[toolName]
	t.mu.Unlock()

	if !known {
		available := t.listToolNames()
		return ErrorResult(fmt.Sprintf(
			"MCP tool %q not found. Available: %s",
			toolName, strings.Join(available, ", "),
		))
	}

	var toolArgs map[string]any
	if raw, ok := args["arguments"]; ok {
		if a, ok := raw.(map[string]any); ok {
			toolArgs = a
		}
	}

	t.mu.Lock()
	result, err := t.call("tools/call", map[string]any{
		"name":      toolName,
		"arguments": toolArgs,
	})
	t.mu.Unlock()

	if err != nil {
		return ErrorResult(fmt.Sprintf("MCP call %q failed: %v", toolName, err))
	}

	// Parse MCP tool result
	var mcpResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &mcpResult); err != nil {
		// Return raw JSON if can't parse structured content
		return NewToolResult(string(result))
	}

	// Concatenate text content blocks
	var sb strings.Builder
	for _, block := range mcpResult.Content {
		if block.Type == "text" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(block.Text)
		}
	}

	text := sb.String()
	if text == "" {
		text = string(result)
	}

	if mcpResult.IsError {
		return ErrorResult(text)
	}

	return NewToolResult(text)
}

func (t *MCPClientTool) listToolNames() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	names := make([]string, 0, len(t.mcpTools))
	for name := range t.mcpTools {
		names = append(names, name)
	}
	return names
}

// call sends a JSON-RPC 2.0 request and reads the response.
// Caller must hold t.mu.
func (t *MCPClientTool) call(method string, params any) (json.RawMessage, error) {
	if t.stdin == nil || t.stdout == nil {
		return nil, errors.New("MCP server not running")
	}

	id := t.nextID.Add(1)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, writeErr := t.stdin.Write(data); writeErr != nil {
		return nil, fmt.Errorf("write request: %w", writeErr)
	}

	// Read response line
	line, readErr := t.stdout.ReadBytes('\n')
	if readErr != nil {
		return nil, fmt.Errorf("read response: %w", readErr)
	}

	var resp jsonRPCResponse
	if unmarshalErr := json.Unmarshal(line, &resp); unmarshalErr != nil {
		return nil, fmt.Errorf("parse response: %w", unmarshalErr)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

// notify sends a JSON-RPC 2.0 notification (no id, no response expected).
// Caller must hold t.mu.
func (t *MCPClientTool) notify(method string, params any) error {
	if t.stdin == nil {
		return errors.New("MCP server not running")
	}

	type notification struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}

	data, err := json.Marshal(notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	data = append(data, '\n')

	_, err = t.stdin.Write(data)
	return err
}
