// Package core provides the CoreProxy that spawns the F*-extracted verified
// core binary and communicates with it via JSON-RPC over STDIO.
//
// The proxy implements the same message processing interface as the Go agent
// loop, allowing the gateway to switch between legacy (Go) and verified (F*)
// modes transparently.
package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tinyland-inc/picoclaw/pkg/logger"
)

// CoreProxy manages the lifecycle of the F*-extracted core binary
// and handles JSON-RPC communication over STDIO.
type CoreProxy struct {
	binaryPath string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	mu         sync.Mutex
	nextID     atomic.Uint64
	callbacks  map[uint64]chan json.RawMessage
	callbackMu sync.Mutex
	running    bool
}

// RPCRequest is a JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// RPCResponse is a JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ProcessMessageParams are the parameters for the process_message RPC call.
type ProcessMessageParams struct {
	Channel       string `json:"channel"`
	AccountID     string `json:"account_id"`
	SenderID      string `json:"sender_id"`
	ChatID        string `json:"chat_id"`
	Content       string `json:"content"`
	RequestID     string `json:"request_id"`
	MaxIterations int    `json:"max_iterations"`
}

// ProcessMessageResult is the result of the process_message RPC call.
type ProcessMessageResult struct {
	Content    string          `json:"content"`
	AgentID    string          `json:"agent_id"`
	SessionKey string          `json:"session_key"`
	AuditLog   json.RawMessage `json:"audit_log"`
}

// AuditEntry is a single entry in the verified core's audit log.
type AuditEntry struct {
	Sequence   int    `json:"sequence"`
	Timestamp  int64  `json:"timestamp"`
	Event      string `json:"event"`
	AgentID    string `json:"agent_id"`
	SessionKey string `json:"session_key"`
	PrevHash   string `json:"prev_hash"`
	RequestID  string `json:"request_id"`
}

// NewCoreProxy creates a new CoreProxy for the given binary path.
func NewCoreProxy(binaryPath string) *CoreProxy {
	return &CoreProxy{
		binaryPath: binaryPath,
		callbacks:  make(map[uint64]chan json.RawMessage),
	}
}

// Start spawns the core binary as a subprocess.
func (p *CoreProxy) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("core proxy already running")
	}

	p.cmd = exec.CommandContext(ctx, p.binaryPath)
	var err error

	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	p.stdout = bufio.NewReader(stdout)

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start core binary: %w", err)
	}

	p.running = true

	// Start response reader goroutine
	go p.readResponses()

	logger.InfoC("core", "Verified core started: "+p.binaryPath)
	return nil
}

// Stop terminates the core binary.
func (p *CoreProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.running = false
	if p.stdin != nil {
		p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Wait()
	}
	return nil
}

// IsRunning returns whether the core binary is running.
func (p *CoreProxy) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// ProcessMessage sends a message to the verified core for processing.
func (p *CoreProxy) ProcessMessage(ctx context.Context, params ProcessMessageParams) (*ProcessMessageResult, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	resultJSON, err := p.call(ctx, "process_message", paramsJSON)
	if err != nil {
		return nil, err
	}

	var result ProcessMessageResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// call sends a JSON-RPC request and waits for the response.
func (p *CoreProxy) call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	id := p.nextID.Add(1)

	req := RPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Register callback channel
	ch := make(chan json.RawMessage, 1)
	p.callbackMu.Lock()
	p.callbacks[id] = ch
	p.callbackMu.Unlock()

	defer func() {
		p.callbackMu.Lock()
		delete(p.callbacks, id)
		p.callbackMu.Unlock()
	}()

	// Send request
	if err := p.sendRequest(req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendRequest writes a JSON-RPC request to the core's stdin.
func (p *CoreProxy) sendRequest(req RPCRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(p.stdin, header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := p.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

// readResponses continuously reads JSON-RPC responses from the core's stdout.
func (p *CoreProxy) readResponses() {
	for {
		p.mu.Lock()
		running := p.running
		p.mu.Unlock()
		if !running {
			return
		}

		// Read Content-Length header
		header, err := p.stdout.ReadString('\n')
		if err != nil {
			if p.running {
				logger.ErrorCF("core", "Failed to read header", map[string]any{"error": err.Error()})
			}
			return
		}

		header = strings.TrimSpace(header)
		if !strings.HasPrefix(header, "Content-Length:") {
			continue
		}

		lengthStr := strings.TrimSpace(strings.TrimPrefix(header, "Content-Length:"))
		contentLength, err := strconv.Atoi(lengthStr)
		if err != nil {
			logger.ErrorCF("core", "Invalid content length", map[string]any{"header": header})
			continue
		}

		// Skip blank line
		if _, err := p.stdout.ReadString('\n'); err != nil {
			return
		}

		// Read content
		buf := make([]byte, contentLength)
		if _, err := io.ReadFull(p.stdout, buf); err != nil {
			if p.running {
				logger.ErrorCF("core", "Failed to read content", map[string]any{"error": err.Error()})
			}
			return
		}

		// Parse response
		var resp RPCResponse
		if err := json.Unmarshal(buf, &resp); err != nil {
			logger.ErrorCF("core", "Failed to parse response", map[string]any{"error": err.Error()})
			continue
		}

		// Dispatch to callback
		p.callbackMu.Lock()
		if ch, ok := p.callbacks[resp.ID]; ok {
			if resp.Error != nil {
				// Convert error to nil result (caller sees error via channel close)
				close(ch)
			} else {
				ch <- resp.Result
			}
		}
		p.callbackMu.Unlock()
	}
}
