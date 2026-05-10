# ADR 0002: Brute-Force Vector Search for v1

## Status
Accepted

## Context
Implementing vector similarity search requires choosing between exact (brute-force) or approximate nearest neighbor (ANN) algorithms.

## Decision
We will use a brute-force cosine similarity scan for V1.

## Consequences
- **Pros**: 100% correct recall, simplified implementation without complex graph data structures. O(N*d) scan is perfectly acceptable for the target of 10,000 documents.
- **Cons**: Slows down query latency for larger datasets. We may implement HNSW (Hierarchical Navigable Small World) in V2.
