/* PicoClaw FFI header for KaRaMeL-extracted C code.
 *
 * This header provides the C interface between:
 * - KaRaMeL-extracted F* code (verified core logic)
 * - Futhark-compiled C code (parallel compute kernels)
 * - System libraries (TLS, networking, I/O)
 *
 * The combined binary (picoclaw-verified) links all three together
 * into a single executable with no Go or OCaml runtime dependency.
 */

#ifndef PICOCLAW_FFI_H
#define PICOCLAW_FFI_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

/* ─── String type ───────────────────────────────────────────────── */

/* KaRaMeL extracts F* strings as C strings (null-terminated).
 * These helpers provide safe string operations. */

typedef const char* picoclaw_string;

/* Concatenate strings with separator. Caller must free result. */
char* picoclaw_string_concat(const char* sep, const char** parts, size_t count);

/* Hash a string (SHA-256). Returns hex-encoded hash. Caller must free. */
char* picoclaw_string_hash(const char* input);

/* ─── Audit log ─────────────────────────────────────────────────── */

/* Audit event types (matching F* audit_event) */
typedef enum {
    PICOCLAW_EVENT_ROUTE_RESOLVED = 0,
    PICOCLAW_EVENT_TOOL_AUTHORIZED = 1,
    PICOCLAW_EVENT_TOOL_DENIED = 2,
    PICOCLAW_EVENT_TOOL_EXECUTED = 3,
    PICOCLAW_EVENT_LLM_CALL_STARTED = 4,
    PICOCLAW_EVENT_LLM_CALL_COMPLETED = 5,
    PICOCLAW_EVENT_SESSION_CREATED = 6,
    PICOCLAW_EVENT_MESSAGE_PROCESSED = 7,
    PICOCLAW_EVENT_APERTURE_METERING = 8,
    PICOCLAW_EVENT_CERBOS_DECISION = 9,
} picoclaw_event_type;

/* Audit entry (matching F* audit_entry) */
typedef struct {
    uint32_t sequence;
    uint64_t timestamp;
    picoclaw_event_type event_type;
    const char* event_detail;
    const char* agent_id;
    const char* session_key;
    const char* prev_hash;
    const char* request_id;
} picoclaw_audit_entry;

/* ─── Routing ───────────────────────────────────────────────────── */

/* Match reason (matching F* match_reason) */
typedef enum {
    PICOCLAW_MATCH_PEER = 0,
    PICOCLAW_MATCH_PARENT_PEER = 1,
    PICOCLAW_MATCH_GUILD = 2,
    PICOCLAW_MATCH_TEAM = 3,
    PICOCLAW_MATCH_ACCOUNT = 4,
    PICOCLAW_MATCH_CHANNEL_WILDCARD = 5,
    PICOCLAW_MATCH_DEFAULT = 6,
} picoclaw_match_reason;

/* Resolved route (matching F* resolved_route) */
typedef struct {
    const char* agent_id;
    const char* channel;
    const char* account_id;
    const char* session_key;
    const char* main_session_key;
    picoclaw_match_reason matched_by;
} picoclaw_resolved_route;

/* ─── Tool authorization ────────────────────────────────────────── */

typedef enum {
    PICOCLAW_AUTH_ALWAYS_ALLOWED = 0,
    PICOCLAW_AUTH_REQUIRES_GRANT = 1,
    PICOCLAW_AUTH_ALWAYS_DENIED = 2,
} picoclaw_auth_level;

typedef struct {
    bool authorized;
    const char* reason;  /* NULL if authorized, reason string if denied */
} picoclaw_auth_decision;

/* ─── Futhark kernel interface ──────────────────────────────────── */

/* Batch cosine similarity: query vs candidates matrix.
 * Returns array of similarity scores. Caller must free result. */
float* picoclaw_futhark_batch_similarity(
    const float* query, size_t query_len,
    const float* candidates, size_t num_candidates, size_t candidate_len);

/* Top-k similar: returns indices of top k most similar candidates.
 * Caller must free result. */
int32_t* picoclaw_futhark_top_k_similar(
    const float* query, size_t query_len,
    const float* candidates, size_t num_candidates, size_t candidate_len,
    size_t k);

/* Batch token estimation. Returns array of token counts.
 * Caller must free result. */
int32_t* picoclaw_futhark_batch_estimate_tokens(
    const char** texts, size_t count);

/* ─── JSON-RPC I/O ──────────────────────────────────────────────── */

/* Read a Content-Length framed message from fd.
 * Returns malloc'd buffer. Caller must free. */
char* picoclaw_read_message(int fd);

/* Write a Content-Length framed message to fd. */
int picoclaw_write_message(int fd, const char* content, size_t len);

/* ─── Main entry point ──────────────────────────────────────────── */

/* Initialize the verified core. Returns 0 on success. */
int picoclaw_core_init(void);

/* Run the JSON-RPC main loop. Returns exit code. */
int picoclaw_core_run(void);

/* Shutdown the verified core. */
void picoclaw_core_shutdown(void);

#endif /* PICOCLAW_FFI_H */
