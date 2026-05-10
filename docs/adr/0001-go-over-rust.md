# ADR 0001: Go over Rust

## Status
Accepted

## Context
We need to select the core language for implementing a single-threaded educational relational and vector database.

## Decision
We chose Go 1.22+. 

## Consequences
- **Pros**: Fast compilation, robust standard library, excellent garbage collection for simplified development, built-in tooling (gofmt, `testing` package).
- **Cons**: GC pauses could impact strict performance predictability, but this is an acceptable tradeoff for an educational database vs the learning curve of Rust borrow checker in this context.
