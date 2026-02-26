-- Parallel token estimation for context window management.
--
-- Estimates token counts for batches of text using the heuristic
-- from the Go codebase: totalChars * 2 / 5 (2.5 chars per token),
-- with CJK character detection for better accuracy.

-- Check if a character code is in the CJK Unified Ideographs range.
let is_cjk (c: i32) : bool =
  (c >= 0x4E00 && c <= 0x9FFF) ||    -- CJK Unified Ideographs
  (c >= 0x3400 && c <= 0x4DBF) ||    -- CJK Unified Ideographs Extension A
  (c >= 0xF900 && c <= 0xFAFF) ||    -- CJK Compatibility Ideographs
  (c >= 0x3040 && c <= 0x309F) ||    -- Hiragana
  (c >= 0x30A0 && c <= 0x30FF) ||    -- Katakana
  (c >= 0xAC00 && c <= 0xD7AF)       -- Hangul Syllables

-- Estimate token count for a single text (as array of char codes).
-- CJK characters count as ~1.5 tokens each, ASCII as ~0.4 tokens each.
let estimate_tokens_single [n] (chars: [n]i32) : i64 =
  let cjk_count = i64.sum (map (\c -> if is_cjk c then 1i64 else 0i64) chars)
  let ascii_count = i64.i64 n - cjk_count
  -- CJK: ~1.5 tokens per char, ASCII: ~0.4 tokens per char
  -- Equivalent to: cjk * 3/2 + ascii * 2/5
  let cjk_tokens = cjk_count * 3 / 2
  let ascii_tokens = ascii_count * 2 / 5
  in cjk_tokens + ascii_tokens

-- Batch token estimation for multiple texts.
-- Each text is represented as a flat array of char codes with offsets/lengths.
entry batch_estimate_tokens [total][m]
  (all_chars: [total]i32)    -- Flat concatenated char codes
  (offsets: [m]i64)          -- Start offset of each text
  (lengths: [m]i64)          -- Length of each text
  : [m]i64 =
  map2 (\offset len ->
    let chars = all_chars[offset : offset + len]
    in estimate_tokens_single chars
  ) offsets lengths

-- Estimate total tokens for a conversation (multiple messages).
-- Returns per-message estimates and the total.
entry conversation_tokens [total][m]
  (all_chars: [total]i32)
  (offsets: [m]i64)
  (lengths: [m]i64)
  : (i64, [m]i64) =
  let per_message = batch_estimate_tokens all_chars offsets lengths
  let total_tokens = i64.sum per_message
  in (total_tokens, per_message)

-- Check if conversation exceeds context window percentage.
-- Returns true if estimated tokens > max_tokens * threshold_pct / 100.
entry exceeds_context_threshold [total][m]
  (all_chars: [total]i32)
  (offsets: [m]i64)
  (lengths: [m]i64)
  (max_tokens: i64)
  (threshold_pct: i64)
  : bool =
  let (total_tokens, _) = conversation_tokens all_chars offsets lengths
  in total_tokens * 100 > max_tokens * threshold_pct
