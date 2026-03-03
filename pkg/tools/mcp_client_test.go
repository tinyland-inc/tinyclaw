package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinyland-inc/tinyclaw/pkg/tools"
)

// TestNewMCPClientTool verifies the constructor and interface compliance.
func TestNewMCPClientTool(t *testing.T) {
	client := tools.NewMCPClientTool("test-mcp", "Test MCP server")

	if client.Name() != "test-mcp" {
		t.Errorf("Name() = %q, want %q", client.Name(), "test-mcp")
	}
	if client.Description() != "Test MCP server" {
		t.Errorf("Description() = %q, want %q", client.Description(), "Test MCP server")
	}

	params := client.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters type = %v, want 'object'", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters properties not a map")
	}
	if _, ok := props["tool_name"]; !ok {
		t.Error("Parameters missing 'tool_name' property")
	}
}

// TestMCPClientToolExecuteWithoutStart verifies error when server not started.
func TestMCPClientToolExecuteWithoutStart(t *testing.T) {
	client := tools.NewMCPClientTool("test", "Test")

	result := client.Execute(context.Background(), map[string]any{
		"tool_name": "some_tool",
	})

	if !result.IsError {
		t.Error("Execute without Start should return error")
	}
	if !strings.Contains(result.ForLLM, "not found") {
		t.Errorf("Error message should mention 'not found', got: %s", result.ForLLM)
	}
}

// TestMCPClientToolExecuteMissingToolName verifies required field validation.
func TestMCPClientToolExecuteMissingToolName(t *testing.T) {
	client := tools.NewMCPClientTool("test", "Test")

	result := client.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("Execute without tool_name should return error")
	}
	if !strings.Contains(result.ForLLM, "tool_name") {
		t.Errorf("Error should mention 'tool_name', got: %s", result.ForLLM)
	}
}

// TestMCPClientToolExecuteEmptyToolName verifies empty string rejection.
func TestMCPClientToolExecuteEmptyToolName(t *testing.T) {
	client := tools.NewMCPClientTool("test", "Test")

	result := client.Execute(context.Background(), map[string]any{
		"tool_name": "",
	})

	if !result.IsError {
		t.Error("Execute with empty tool_name should return error")
	}
}

// TestMCPProxyToolInterface verifies MCPProxyTool satisfies Tool interface.
func TestMCPProxyToolInterface(t *testing.T) {
	var _ tools.Tool = (*tools.MCPProxyTool)(nil)
}

// TestMCPRegistryWithMockServer tests the full lifecycle with a mock MCP server.
func TestMCPRegistryWithMockServer(t *testing.T) {
	mockScript := createMockMCPServer(t)

	ctx := context.Background()
	registry := tools.NewToolRegistry()

	count, err := tools.RegisterMCPServer(
		ctx, registry,
		"mock-gnucash", mockScript, nil, "",
	)
	if err != nil {
		t.Fatalf("RegisterMCPServer failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 tools registered, got %d", count)
	}

	// Verify tools are in registry
	toolNames := registry.List()
	found := map[string]bool{}
	for _, name := range toolNames {
		found[name] = true
	}

	if !found["get_accounts"] {
		t.Error("Expected 'get_accounts' tool in registry")
	}
	if !found["get_balance"] {
		t.Error("Expected 'get_balance' tool in registry")
	}

	// Execute a mock tool
	result := registry.Execute(ctx, "get_accounts", map[string]any{})
	if result.IsError {
		t.Errorf("get_accounts returned error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Checking") {
		t.Errorf("Expected 'Checking' in result, got: %s", result.ForLLM)
	}
}

// TestMCPRegistryToolExecution tests calling individual proxy tools.
func TestMCPRegistryToolExecution(t *testing.T) {
	mockScript := createMockMCPServer(t)

	ctx := context.Background()
	registry := tools.NewToolRegistry()

	_, err := tools.RegisterMCPServer(ctx, registry, "mock", mockScript, nil, "")
	if err != nil {
		t.Fatalf("RegisterMCPServer: %v", err)
	}

	result := registry.Execute(ctx, "get_balance", map[string]any{
		"account": "checking",
	})

	if result.IsError {
		t.Errorf("get_balance returned error: %s", result.ForLLM)
	}
}

// TestMCPRegistryWithPrefix tests tool name prefixing.
func TestMCPRegistryWithPrefix(t *testing.T) {
	mockScript := createMockMCPServer(t)

	ctx := context.Background()
	registry := tools.NewToolRegistry()

	count, err := tools.RegisterMCPServer(ctx, registry, "mock", mockScript, nil, "gnucash_")
	if err != nil {
		t.Fatalf("RegisterMCPServer: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 tools, got %d", count)
	}

	toolNames := registry.List()
	for _, name := range toolNames {
		if !strings.HasPrefix(name, "gnucash_") {
			t.Errorf("Tool %q should have 'gnucash_' prefix", name)
		}
	}
}

// TestMCPClientIntegrationWithBridge tests with the real gnucash-bridge binary.
func TestMCPClientIntegrationWithBridge(t *testing.T) {
	bridge, err := exec.LookPath("gnucash-bridge")
	if err != nil {
		t.Skip("gnucash-bridge not in PATH; skipping integration test")
	}

	ctx := context.Background()
	client := tools.NewMCPClientTool("gnucash", "GnuCash bridge")

	if err := client.Start(ctx, bridge, []string{"--mcp"}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer client.Stop()

	discovered := client.DiscoveredTools()
	if len(discovered) == 0 {
		t.Error("Expected discovered tools from gnucash-bridge")
	}

	t.Logf("Discovered %d tools from gnucash-bridge", len(discovered))
	for name := range discovered {
		t.Logf("  - %s", name)
	}
}

// createMockMCPServer creates a temporary bash script that acts as an MCP server.
func createMockMCPServer(t *testing.T) string {
	t.Helper()

	script := `#!/bin/bash
# Mock MCP server for testing
while IFS= read -r line; do
  method=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('method',''))" 2>/dev/null)
  id=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',0))" 2>/dev/null)

  case "$method" in
    initialize)
      echo '{"jsonrpc":"2.0","id":'"$id"',"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"mock-gnucash","version":"1.0.0"}}}'
      ;;
    notifications/initialized)
      # No response for notifications
      ;;
    tools/list)
      echo '{"jsonrpc":"2.0","id":'"$id"',"result":{"tools":[{"name":"get_accounts","description":"Get all accounts","inputSchema":{"type":"object","properties":{}}},{"name":"get_balance","description":"Get account balance","inputSchema":{"type":"object","properties":{"account":{"type":"string"}},"required":["account"]}}]}}'
      ;;
    tools/call)
      tool_name=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('params',{}).get('name',''))" 2>/dev/null)
      case "$tool_name" in
        get_accounts)
          echo '{"jsonrpc":"2.0","id":'"$id"',"result":{"content":[{"type":"text","text":"Accounts: Checking, Savings, Credit Card"}]}}'
          ;;
        get_balance)
          echo '{"jsonrpc":"2.0","id":'"$id"',"result":{"content":[{"type":"text","text":"Balance: $1,500.00"}]}}'
          ;;
        *)
          echo '{"jsonrpc":"2.0","id":'"$id"',"error":{"code":-32601,"message":"Unknown tool"}}'
          ;;
      esac
      ;;
    *)
      echo '{"jsonrpc":"2.0","id":'"$id"',"error":{"code":-32601,"message":"Method not found"}}'
      ;;
  esac
done
`

	// Check if python3 is available (needed for the mock)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available; skipping mock MCP server tests")
	}

	scriptPath := filepath.Join(t.TempDir(), "mock-mcp.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	return scriptPath
}

// TestJSONRPCTypes verifies JSON serialization of request/response types.
func TestJSONRPCTypes(t *testing.T) {
	// Verify we can marshal a request
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_accounts",
			"arguments": map[string]any{},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(data), "tools/call") {
		t.Error("Marshaled request should contain method")
	}

	// Verify we can unmarshal a response
	respJSON := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"hello"}]}}`
	var resp map[string]any
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp["id"] != float64(1) {
		t.Errorf("id = %v, want 1", resp["id"])
	}
}
