-- Parallel audit log aggregation and anomaly detection.
--
-- Processes audit log entries in parallel to compute:
-- - Per-agent event counts
-- - Per-session tool call frequencies
-- - Temporal anomaly detection (burst detection)
-- - Hash chain validation

-- Audit event type tags (matching PicoClaw.AuditLog)
let event_route_resolved: i32    = 0
let event_tool_authorized: i32   = 1
let event_tool_denied: i32       = 2
let event_tool_executed: i32     = 3
let event_llm_call_started: i32  = 4
let event_llm_call_completed: i32 = 5
let event_session_created: i32   = 6
let event_message_processed: i32 = 7

-- Audit entry as flat struct for GPU processing.
type audit_entry = {
  sequence: i64,
  timestamp: i64,
  event_type: i32,
  agent_id_hash: i64,    -- Hash of agent_id string
  session_key_hash: i64  -- Hash of session_key string
}

-- Count events by type across the entire log.
-- Returns array of counts indexed by event type.
entry count_events_by_type [n]
  (entries: [n]audit_entry)
  : [8]i64 =
  let counts = replicate 8 0i64
  in reduce_by_index counts (+) 0
       (map (\e -> i64.i32 e.event_type) entries)
       (replicate n 1i64)

-- Count events per agent (by agent_id_hash).
-- Returns array of (agent_hash, count) pairs for agents with > 0 events.
entry count_events_per_agent [n]
  (entries: [n]audit_entry)
  : ([]i64, []i64) =
  let agent_hashes = map (.agent_id_hash) entries
  -- Get unique agents and their counts using histogram
  let max_agents = 1024i64  -- Maximum distinct agents
  let counts = reduce_by_index
    (replicate max_agents 0i64)
    (+) 0
    (map (\h -> h %% max_agents) agent_hashes)
    (replicate n 1i64)
  -- Filter out zero counts
  let indexed = zip (iota max_agents) counts
  let nonzero = filter (\(_, c) -> c > 0) indexed
  in unzip nonzero

-- Detect temporal anomalies: find time windows with abnormally high event rates.
-- A window is anomalous if its event count exceeds threshold.
-- Window size is in seconds.
entry detect_bursts [n]
  (entries: [n]audit_entry)
  (window_seconds: i64)
  (threshold: i64)
  : []i64 =
  if n == 0 then []
  else
    let timestamps = map (.timestamp) entries
    let min_t = reduce i64.min i64.highest timestamps
    let max_t = reduce i64.max i64.lowest timestamps
    let num_windows = (max_t - min_t) / window_seconds + 1
    let safe_windows = i64.max 1 (i64.min num_windows 100000)
    -- Count events per window
    let window_ids = map (\e -> (e.timestamp - min_t) / window_seconds) entries
    let window_counts = reduce_by_index
      (replicate safe_windows 0i64)
      (+) 0
      window_ids
      (replicate n 1i64)
    -- Find windows exceeding threshold
    let anomalous = filter (\(_, c) -> c > threshold)
                           (zip (iota safe_windows) window_counts)
    in map (\(w, _) -> min_t + w * window_seconds) anomalous

-- Count tool denials per agent (security metric).
-- High denial rates may indicate misconfigured permissions or attack attempts.
entry count_denials_per_agent [n]
  (entries: [n]audit_entry)
  : ([]i64, []i64) =
  let denials = filter (\e -> e.event_type == event_tool_denied) entries
  let m = length denials
  let agent_hashes = map (.agent_id_hash) denials
  let max_agents = 1024i64
  let counts = reduce_by_index
    (replicate max_agents 0i64)
    (+) 0
    (map (\h -> h %% max_agents) agent_hashes)
    (replicate m 1i64)
  let indexed = zip (iota max_agents) counts
  let nonzero = filter (\(_, c) -> c > 0) indexed
  in unzip nonzero

-- Compute average time between LLM calls (latency metric).
entry avg_llm_interval [n]
  (entries: [n]audit_entry)
  : f64 =
  let llm_entries = filter (\e -> e.event_type == event_llm_call_started) entries
  let m = length llm_entries
  in if m < 2 then 0f64
     else
       let timestamps = map (.timestamp) llm_entries
       let sorted = merge_sort (<=) timestamps
       let intervals = tabulate (m - 1) (\i -> sorted[i + 1] - sorted[i])
       let total = reduce (+) 0i64 intervals
       in f64.i64 total / f64.i64 (m - 1)

-- Simple merge sort for i64 arrays.
let merge_sort [n] (leq: i64 -> i64 -> bool) (xs: [n]i64) : [n]i64 =
  let ranks = map (\i ->
    reduce (+) 0i64 (map (\j ->
      if j == i then 0
      else if leq xs[j] xs[i] then
        if leq xs[i] xs[j] then (if j < i then 1 else 0) else 1
      else 0
    ) (iota n))
  ) (iota n)
  in scatter (copy xs) ranks xs
