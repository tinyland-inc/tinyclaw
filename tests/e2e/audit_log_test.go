package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

// AuditEntry mirrors the F* PicoClaw.AuditLog audit_entry type.
type AuditEntry struct {
	Sequence   int64
	Timestamp  int64
	EventType  string
	AgentID    string
	SessionKey string
	Details    string
	PrevHash   string
	Hash       string
}

// computeHash creates the hash for an audit entry.
func computeHash(entry *AuditEntry) string {
	data := fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s",
		entry.Sequence, entry.Timestamp, entry.EventType,
		entry.AgentID, entry.SessionKey, entry.Details, entry.PrevHash)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// appendAuditEntry creates a new audit entry chained to the previous.
func appendAuditEntry(log []AuditEntry, eventType, agentID, sessionKey, details string) []AuditEntry {
	prevHash := ""
	seq := int64(0)
	if len(log) > 0 {
		prevHash = log[len(log)-1].Hash
		seq = log[len(log)-1].Sequence + 1
	}

	entry := AuditEntry{
		Sequence:   seq,
		Timestamp:  time.Now().UnixMilli(),
		EventType:  eventType,
		AgentID:    agentID,
		SessionKey: sessionKey,
		Details:    details,
		PrevHash:   prevHash,
	}
	entry.Hash = computeHash(&entry)

	return append(log, entry)
}

// validateChain verifies the hash chain integrity.
func validateChain(log []AuditEntry) error {
	for i, entry := range log {
		if i == 0 {
			if entry.PrevHash != "" {
				return fmt.Errorf("entry[0] should have empty prev_hash, got %q", entry.PrevHash)
			}
		} else {
			if entry.PrevHash != log[i-1].Hash {
				return fmt.Errorf("entry[%d] prev_hash mismatch: got %q, want %q",
					i, entry.PrevHash, log[i-1].Hash)
			}
		}

		computed := computeHash(&entry)
		if computed != entry.Hash {
			return fmt.Errorf("entry[%d] hash mismatch: computed %q, stored %q", i, computed, entry.Hash)
		}

		if entry.Sequence != int64(i) {
			return fmt.Errorf("entry[%d] sequence mismatch: got %d", i, entry.Sequence)
		}
	}
	return nil
}

func TestAuditLog_EmptyChainValid(t *testing.T) {
	var log []AuditEntry
	if err := validateChain(log); err != nil {
		t.Errorf("empty chain should be valid: %v", err)
	}
}

func TestAuditLog_SingleEntry(t *testing.T) {
	var log []AuditEntry
	log = appendAuditEntry(log, "route_resolved", "agent-1", "session-1", "matched channel binding")

	if err := validateChain(log); err != nil {
		t.Errorf("single entry chain invalid: %v", err)
	}

	if log[0].PrevHash != "" {
		t.Error("first entry should have empty prev_hash")
	}
	if log[0].Hash == "" {
		t.Error("entry should have non-empty hash")
	}
}

func TestAuditLog_ChainIntegrity(t *testing.T) {
	var log []AuditEntry

	events := []struct {
		eventType, agent, session, details string
	}{
		{"route_resolved", "agent-1", "s1", "channel match"},
		{"tool_authorized", "agent-1", "s1", "tool: web_search, grant: always_allowed"},
		{"tool_executed", "agent-1", "s1", "web_search completed in 120ms"},
		{"llm_call_started", "agent-1", "s1", "model: claude-sonnet-4.6"},
		{"llm_call_completed", "agent-1", "s1", "tokens: 1200"},
		{"message_processed", "agent-1", "s1", "response sent"},
	}

	for _, e := range events {
		log = appendAuditEntry(log, e.eventType, e.agent, e.session, e.details)
	}

	if err := validateChain(log); err != nil {
		t.Errorf("chain integrity check failed: %v", err)
	}

	if len(log) != 6 {
		t.Errorf("expected 6 entries, got %d", len(log))
	}
}

func TestAuditLog_TamperDetection(t *testing.T) {
	var log []AuditEntry
	log = appendAuditEntry(log, "route_resolved", "agent-1", "s1", "channel match")
	log = appendAuditEntry(log, "tool_authorized", "agent-1", "s1", "always_allowed")
	log = appendAuditEntry(log, "tool_executed", "agent-1", "s1", "completed")

	if err := validateChain(log); err != nil {
		t.Fatalf("chain should be valid before tampering: %v", err)
	}

	// Tamper with the middle entry
	log[1].Details = "TAMPERED"

	if err := validateChain(log); err == nil {
		t.Error("expected chain validation to fail after tampering")
	}
}

func TestAuditLog_MonotonicSequence(t *testing.T) {
	var log []AuditEntry
	for i := range 100 {
		log = appendAuditEntry(log, "event", "agent-1", "s1", fmt.Sprintf("event %d", i))
	}

	for i, entry := range log {
		if entry.Sequence != int64(i) {
			t.Errorf("entry[%d] sequence: got %d, want %d", i, entry.Sequence, i)
		}
	}

	if err := validateChain(log); err != nil {
		t.Errorf("100-entry chain invalid: %v", err)
	}
}

func TestAuditLog_AppendMonotonic(t *testing.T) {
	var log []AuditEntry
	log = appendAuditEntry(log, "event_a", "agent-1", "s1", "first")

	prevLen := len(log)
	log = appendAuditEntry(log, "event_b", "agent-1", "s1", "second")

	if len(log) != prevLen+1 {
		t.Errorf("append should increase length by 1: got %d, want %d", len(log), prevLen+1)
	}

	// Verify monotonicity: new entry's sequence > previous
	if log[len(log)-1].Sequence <= log[len(log)-2].Sequence {
		t.Error("sequence should be monotonically increasing")
	}
}
