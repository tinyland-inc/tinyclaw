-- Batch cosine similarity for skills search and memory retrieval.
--
-- Given a query vector and a matrix of candidate vectors,
-- compute cosine similarity for all candidates in parallel.
-- Used for skill matching, memory retrieval, and semantic search.

import "../lib/linalg"

-- Compute cosine similarity between a query and each row of a matrix.
-- Returns an array of similarity scores, one per candidate.
entry batch_cosine_similarity [m][n]
  (query: [n]f32)
  (candidates: [m][n]f32)
  : [m]f32 =
  map (\c -> cosine_similarity query c) candidates

-- Find the top-k most similar candidates.
-- Returns indices sorted by descending similarity.
-- Uses a simple parallel sort approach.
entry top_k_similar [m][n]
  (query: [n]f32)
  (candidates: [m][n]f32)
  (k: i64)
  : []i64 =
  let scores = batch_cosine_similarity query candidates
  -- Create (score, index) pairs
  let indexed = zip scores (iota m)
  -- Sort by descending score using a bitonic-style approach
  let sorted = merge_sort (\(a, _) (b, _) -> a > b) indexed
  -- Take top k indices
  let k' = i64.min k m
  in map (.1) (take k' sorted)

-- Batch similarity: compute pairwise similarities for multiple queries.
-- Returns an [q][m] matrix where result[i][j] = similarity(queries[i], candidates[j]).
entry batch_pairwise_similarity [q][m][n]
  (queries: [q][n]f32)
  (candidates: [m][n]f32)
  : [q][m]f32 =
  map (\query -> batch_cosine_similarity query candidates) queries

-- Merge sort implementation for Futhark (bitonic sort variant).
-- Stable sort by predicate.
let merge_sort 't [n] (leq: t -> t -> bool) (xs: [n]t) : [n]t =
  -- Use radix-like approach: flatten sort via reduce_by_index
  -- For simplicity, use Futhark's built-in merge sort via segmented scan
  let iota_n = iota n
  -- Compute ranks: for each element, count how many are "before" it
  let ranks = map (\i ->
    let xi = xs[i]
    in reduce (+) 0i64 (map (\j ->
      if j == i then 0
      else if leq (xs[j]) xi then
        if leq xi (xs[j]) then
          -- Equal elements: break tie by index
          if j < i then 1 else 0
        else 1
      else 0
    ) iota_n)
  ) iota_n
  -- Scatter into sorted positions
  let result = scatter (copy xs) ranks xs
  in result
