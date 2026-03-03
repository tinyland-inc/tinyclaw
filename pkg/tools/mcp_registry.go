package tools

import (
	"context"
	"fmt"

	"github.com/tinyland-inc/tinyclaw/pkg/logger"
)

// MCPProxyTool wraps a single MCP tool as a TinyClaw Tool.
// Each discovered MCP tool becomes one MCPProxyTool in the registry.
type MCPProxyTool struct {
	info   MCPToolInfo
	client *MCPClientTool
}

func (t *MCPProxyTool) Name() string        { return t.info.Name }
func (t *MCPProxyTool) Description() string  { return t.info.Description }

func (t *MCPProxyTool) Parameters() map[string]any {
	if t.info.InputSchema != nil {
		return t.info.InputSchema
	}
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *MCPProxyTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return t.client.Execute(ctx, map[string]any{
		"tool_name": t.info.Name,
		"arguments": args,
	})
}

// RegisterMCPServer starts an MCP server subprocess, discovers its tools,
// and registers each one individually in the TinyClaw tool registry.
//
// Each MCP tool is registered under its own name (e.g., "gnucash_get_accounts").
// The optional prefix prepends to tool names to avoid collisions.
//
// Returns the number of tools registered and any error.
func RegisterMCPServer(
	ctx context.Context,
	registry *ToolRegistry,
	name string,
	command string,
	args []string,
	prefix string,
) (int, error) {
	client := NewMCPClientTool(name, "MCP server: "+name)

	if err := client.Start(ctx, command, args); err != nil {
		return 0, fmt.Errorf("start MCP server %q: %w", name, err)
	}

	discovered := client.DiscoveredTools()
	count := 0

	for _, info := range discovered {
		toolName := info.Name
		if prefix != "" {
			toolName = prefix + info.Name
		}

		// Override the name with the prefixed version
		proxiedInfo := info
		proxiedInfo.Name = toolName

		proxy := &MCPProxyTool{
			info:   proxiedInfo,
			client: client,
		}
		registry.Register(proxy)
		count++

		logger.DebugCF("mcp", "Registered MCP tool", map[string]any{
			"server": name,
			"tool":   toolName,
		})
	}

	logger.InfoCF("mcp", "MCP server tools registered", map[string]any{
		"server": name,
		"count":  count,
		"prefix": prefix,
	})

	return count, nil
}
