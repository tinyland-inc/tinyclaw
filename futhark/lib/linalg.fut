-- Linear algebra primitives for PicoClaw compute kernels.

-- Dot product of two vectors.
let dotprod [n] (a: [n]f32) (b: [n]f32) : f32 =
  reduce (+) 0f32 (map2 (*) a b)

-- L2 norm of a vector.
let norm [n] (v: [n]f32) : f32 =
  f32.sqrt (dotprod v v)

-- Cosine similarity between two vectors.
-- Returns 0 if either vector is zero.
let cosine_similarity [n] (a: [n]f32) (b: [n]f32) : f32 =
  let na = norm a
  let nb = norm b
  in if na == 0f32 || nb == 0f32
     then 0f32
     else dotprod a b / (na * nb)
