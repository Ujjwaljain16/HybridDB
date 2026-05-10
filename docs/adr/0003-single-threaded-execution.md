# ADR 0003: Single-Threaded Execution Model

## Status
Accepted

## Context
Concurrency management adds significant complexity to database internals (e.g., latches, locks, thread-safe buffer pools, MVCC).

## Decision
V1 will use a single-threaded execution model.

## Consequences
- **Pros**: Drastically reduces implementation time and debugging complexity. B+ tree splits, Buffer Pool replacement, and WAL interactions don't require latch coupling.
- **Cons**: Limits throughput. Concurrency and connection pooling are explicitly out of scope for V1.
