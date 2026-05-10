# HybridDB — Complete Build Execution Plan

> **Semester-Long Implementation Playbook for a Hybrid Relational-Vector Educational Database Engine**
> *Principal Systems Architect Edition — Implementation-First, Correctness-First*

---

## Table of Contents

1. [Development Philosophy](#1-development-philosophy)
2. [Engineering Principles](#2-engineering-principles)
3. [Repository Structure](#3-repository-structure)
4. [Development Environment](#4-development-environment)
5. [Module Dependency Graph](#5-module-dependency-graph)
6. [Phase-Wise Execution Plan](#6-phase-wise-execution-plan)
7. [Week-by-Week Roadmap](#7-week-by-week-roadmap)
8. [Testing Strategy](#8-testing-strategy)
9. [Debugging Strategy](#9-debugging-strategy)
10. [Observability Strategy](#10-observability-strategy)
11. [Benchmarking Strategy](#11-benchmarking-strategy)
12. [Risk Analysis](#12-risk-analysis)
13. [Technical Debt Strategy](#13-technical-debt-strategy)
14. [Final Demo Strategy](#14-final-demo-strategy)
15. [Viva / Interview Preparation](#15-viva--interview-preparation)

---

## 1. Development Philosophy

### 1.1 Why Implementation Order Matters in Database Systems

Database systems are uniquely order-sensitive. Unlike a web application where you can build any endpoint independently, a database engine is a strict dependency stack. Every layer relies on the correctness of the layer beneath it.

If you implement the query executor before the buffer pool is correct, every query test becomes a debugging session that conflates query logic bugs with caching bugs. If you implement the B+ tree before the WAL is in place, a crash mid-split leaves the tree in an unknown state and you cannot reason about whether your split logic or your recovery logic is wrong.

The order is not a preference — it is a constraint imposed by the architecture.

```
IMPLEMENTATION ORDER CONSTRAINT CHAIN:

  OS File I/O
      │
      ▼
  Pager (disk abstraction)
      │
      ▼
  Slotted Pages + Tuple Serialization
      │
      ▼
  Buffer Pool (in-memory cache of pages)
      │                │
      ▼                ▼
   WAL Manager     B+ Tree Index
      │                │
      └──────┬──────────┘
             ▼
         SQL Parser + Catalog
             │
             ▼
         Execution Engine (Volcano Model)
             │
             ▼
         Vector Engine (brute-force cosine)
             │
             ▼
         Hybrid Query Engine (rank fusion)
             │
             ▼
         Observability UI
             │
             ▼
         Benchmarking + Polish
```

**Rule:** Never start implementing Layer N until Layer N-1 has passing tests.

### 1.2 Why Storage Must Come Before Query Execution

The query executor is a consumer of the storage layer. It reads pages through the buffer pool, traverses B+ tree nodes, and deserializes tuples from slotted pages. If any of those foundations have bugs — a page read returns stale data, a slot offset calculation is wrong, a tuple deserialization drops a column — every query result is suspect.

By the time you write your first `Filter` operator, the contract below it must be rock solid: "give me a tuple from this page, it will be correctly deserialized, every time." Testing the executor in isolation is possible with mocks, but end-to-end correctness requires a correct storage foundation.

### 1.3 Why Observability Must Be Added Early

The most common failure mode in systems projects is building 80% of the system before realizing you cannot debug the remaining 20%. By week 8, if you have no logging, no metrics, and no visibility into what the buffer pool is doing or which WAL records are being replayed, you will spend more time debugging than building.

Observability infrastructure must be established in Phase 0 (logging framework, structured log format, metrics collection interfaces) and extended in every subsequent phase. Each new module adds its own metrics and log events at the time it is built — not as an afterthought at the end.

**Principle:** The cheapest time to add a log statement is when you write the code. The most expensive time is three weeks later when you are chasing a bug.

### 1.4 Why Vector Search Should Be Delayed

Vector search is computationally independent of the relational engine — it does not require the B+ tree, the WAL, or even the buffer pool in its simplest form. It is tempting to build it early because it feels modern and impressive.

The danger is that vector search introduces a new code path (embedding client, float32 serialization, cosine arithmetic) that, if built before the storage foundation is stable, will interact with unfixed bugs in the storage layer in confusing ways. Additionally, building vector search early creates pressure to integrate it before the query planner is ready, leading to spaghetti code that is difficult to refactor.

**Rule:** Vector search is built in Phase 6, after the relational query executor is demonstrably correct on end-to-end queries.

### 1.5 Correctness Over Features

A database that inserts 5 data types correctly and recovers from crashes reliably is worth more than one that claims to support 20 data types but occasionally loses data. For an educational system, correctness is the only metric that matters. An instructor can see a bug. An instructor cannot unsee one.

**Every phase has a correctness gate.** No phase begins until the previous phase's tests pass. This is non-negotiable.

### 1.6 Iterative Systems Development

Each phase is a complete vertical slice: build, test, instrument, document, then freeze. The code written in each phase is not thrown away — it is the foundation of the next. Refactoring is done at designated refactor checkpoints, not ad hoc during feature development.

---

## 2. Engineering Principles

### P1 — Correctness Before Optimization

Do not add caching, batching, or any performance optimization until the naive version is proven correct by tests. The LRU eviction policy is not interesting until basic page reads and writes work. SIMD cosine similarity is not interesting until scalar cosine similarity returns correct results.

### P2 — Observability Before Performance Tuning

You cannot optimize what you cannot measure. Before claiming a component is slow, instrument it. Before claiming an optimization helps, measure before and after with the same workload.

### P3 — No Hidden State

Every piece of state that affects behavior must be either:
- Explicitly passed as a parameter
- Stored in a named, documented field
- Part of a named context object

Global variables are forbidden except for: compile-time constants, default configuration values, and the top-level logger.

### P4 — Every Subsystem Independently Testable

Each module must be testable without starting the entire database. The pager must be testable with just a temp file. The B+ tree must be testable with an in-memory pager mock. The parser must be testable with a string input. No module may require a running embedding service, a running HTTP server, or a specific file path to run its unit tests.

### P5 — Explicit Interfaces

Every module boundary is defined by a Go interface. The concrete implementation satisfies that interface. Tests use the interface, not the concrete type. This allows mock substitution and makes the dependency structure explicit and auditable.

```go
// GOOD: Explicit interface boundary
type Pager interface {
    ReadPage(pageID uint32, buf []byte) error
    WritePage(pageID uint32, buf []byte) error
    AllocatePage() (uint32, error)
    NumPages() uint32
    Close() error
}

// BAD: Direct struct dependency
type BufferPool struct {
    pager *DiskPager  // concrete type, hard to mock
}
```

### P6 — Deterministic Behavior

Given the same input and the same initial state, every component must produce the same output. This means:
- No use of `time.Now()` inside core logic (pass it as a parameter if needed)
- No use of `rand` without a fixed seed in tests
- No goroutines in core components (v1 is single-threaded)
- No reading from environment variables inside core logic

### P7 — Fail Loudly, Not Silently

If a WAL record fails its checksum check, panic or return a hard error — do not silently skip it and continue. If a B+ tree node has more keys than its maximum, return an error immediately — do not try to "fix" it in place. Silent corruption is the most dangerous class of database bug because it may not surface until the data is already unrecoverable.

```go
// GOOD: Loud failure
if actual != expected {
    return fmt.Errorf("WAL checksum mismatch: LSN=%d, expected=%08x, got=%08x", lsn, expected, actual)
}

// BAD: Silent skip
if actual != expected {
    continue  // silently skip corrupted record
}
```

### P8 — Educational Clarity Over Cleverness

If there is a choice between an elegant but opaque implementation and a verbose but obvious one, choose the obvious one. This is an educational system. Variable names must be descriptive. Every non-obvious algorithm must have a comment explaining the invariant it maintains. Abbreviations are forbidden in exported names.

### P9 — Freeze Before Extend

Before adding a new feature, confirm that existing features still pass all tests. The test suite is the definition of "working." If adding a feature breaks existing tests, fix the regression before proceeding.

### P10 — Document as You Build

Each module gets a `README.md` in its package directory at the time it is built. Not after the semester. Not at demo time. At build time. The README explains: what this module does, what its interface contract is, what its invariants are, and how to run its tests.

---

## 3. Repository Structure

```
hybriddb/
│
├── cmd/
│   ├── hybriddb/
│   │   └── main.go              # Database server entrypoint (HTTP + REPL)
│   ├── hybriddb-cli/
│   │   └── main.go              # Interactive SQL REPL client
│   └── hybriddb-bench/
│       └── main.go              # Benchmark runner entrypoint
│
├── internal/                    # Private packages (not importable externally)
│   │
│   ├── storage/
│   │   ├── pager/
│   │   │   ├── pager.go         # Pager interface + DiskPager implementation
│   │   │   ├── pager_test.go
│   │   │   └── README.md
│   │   ├── page/
│   │   │   ├── slotted.go       # Slotted page layout: header, slot dir, tuples
│   │   │   ├── header.go        # Page header serialization
│   │   │   ├── slotted_test.go
│   │   │   └── README.md
│   │   └── tuple/
│   │       ├── tuple.go         # Tuple struct, schema, serialization
│   │       ├── types.go         # TypeTag constants, type descriptors
│   │       ├── tuple_test.go
│   │       └── README.md
│   │
│   ├── buffer/
│   │   ├── pool.go              # BufferPool: frame management, pin/unpin
│   │   ├── lru.go               # LRU replacement policy
│   │   ├── frame.go             # Frame struct: pageID, data, pinCount, isDirty
│   │   ├── pool_test.go
│   │   └── README.md
│   │
│   ├── wal/
│   │   ├── wal.go               # WALManager: append, flush, checkpoint
│   │   ├── record.go            # WAL record format, serialization, checksum
│   │   ├── recovery.go          # Startup recovery: redo + undo phases
│   │   ├── wal_test.go
│   │   ├── recovery_test.go
│   │   └── README.md
│   │
│   ├── index/
│   │   ├── btree/
│   │   │   ├── btree.go         # BPlusTree: Insert, Search, RangeScan, Delete
│   │   │   ├── node.go          # Node struct, internal/leaf serialization
│   │   │   ├── split.go         # Split logic (leaf + internal)
│   │   │   ├── merge.go         # Merge + borrow logic
│   │   │   ├── cursor.go        # Range scan cursor (leaf chain traversal)
│   │   │   ├── btree_test.go
│   │   │   ├── invariants.go    # assertInvariants() for testing
│   │   │   └── README.md
│   │   └── vectorindex/
│   │       ├── bruteforce.go    # Brute-force cosine similarity, top-K heap
│   │       ├── bruteforce_test.go
│   │       └── README.md
│   │
│   ├── catalog/
│   │   ├── catalog.go           # In-memory schema registry, backed by Page 0
│   │   ├── schema.go            # TableSchema, ColumnDef, IndexDef
│   │   ├── catalog_test.go
│   │   └── README.md
│   │
│   ├── sql/
│   │   ├── lexer/
│   │   │   ├── lexer.go         # Tokenizer: SQL string → token stream
│   │   │   ├── token.go         # Token types and values
│   │   │   ├── lexer_test.go
│   │   │   └── README.md
│   │   ├── parser/
│   │   │   ├── parser.go        # Recursive descent parser: tokens → AST
│   │   │   ├── ast.go           # AST node types
│   │   │   ├── parser_test.go
│   │   │   └── README.md
│   │   └── planner/
│   │       ├── planner.go       # Rule-based logical→physical plan builder
│   │       ├── rules.go         # Planning rules (indexed scan, hybrid detect)
│   │       ├── stats.go         # Selectivity estimation from catalog stats
│   │       ├── planner_test.go
│   │       └── README.md
│   │
│   ├── executor/
│   │   ├── context.go           # ExecContext: buffer pool, WAL, txn, metrics
│   │   ├── operator.go          # Operator interface: Open, Next, Close, Schema
│   │   ├── seqscan.go           # SeqScan operator
│   │   ├── indexscan.go         # IndexScan operator
│   │   ├── filter.go            # Filter operator + predicate evaluators
│   │   ├── projection.go        # Projection operator
│   │   ├── sort.go              # In-memory sort operator
│   │   ├── limit.go             # Limit operator
│   │   ├── vectorscan.go        # VectorScan operator (brute-force cosine)
│   │   ├── hybridscan.go        # HybridScan: VF/FF strategies + rank fusion
│   │   ├── fusion.go            # RRF + WLC rank fusion implementations
│   │   ├── executor_test.go
│   │   └── README.md
│   │
│   ├── vector/
│   │   ├── client.go            # EmbeddingClient: HTTP calls to Python service
│   │   ├── cosine.go            # Cosine similarity, dot product, L2 distance
│   │   ├── cosine_test.go
│   │   └── README.md
│   │
│   ├── metrics/
│   │   ├── collector.go         # MetricsCollector: per-query, per-operator stats
│   │   ├── buffer_metrics.go    # Buffer pool metrics: hit rate, evictions
│   │   ├── query_metrics.go     # Query metrics: latency, rows, pages
│   │   └── README.md
│   │
│   ├── trace/
│   │   ├── tracer.go            # ExecutionTracer: records operator events
│   │   ├── event.go             # TraceEvent types (OperatorOpen, RowEmitted, etc.)
│   │   └── README.md
│   │
│   └── server/
│       ├── server.go            # HTTP server: query endpoint, metrics endpoint
│       ├── ws.go                # WebSocket: live metrics streaming
│       ├── handlers.go          # Request handlers
│       └── README.md
│
├── pkg/                         # Public packages (importable by external tools)
│   └── hybriddbclient/
│       ├── client.go            # Go client library for hybriddb HTTP API
│       └── README.md
│
├── web/                         # React + TypeScript visualization UI
│   ├── src/
│   │   ├── components/
│   │   │   ├── QueryEditor.tsx       # SQL input with syntax highlighting
│   │   │   ├── ExplainPlanTree.tsx   # Operator tree (React Flow)
│   │   │   ├── BTreeVisualizer.tsx   # Interactive B+ tree diagram
│   │   │   ├── BufferPoolHeatmap.tsx # Frame grid: clean/dirty/pinned
│   │   │   ├── WALLogViewer.tsx      # WAL record stream with color coding
│   │   │   ├── VectorSpaceView.tsx   # 2D PCA projection of embedding space
│   │   │   ├── HybridTimeline.tsx    # Gantt-style execution timeline
│   │   │   ├── MetricsDashboard.tsx  # Recharts: latency, hit rate, candidates
│   │   │   └── ResultTable.tsx       # Query result table with score column
│   │   ├── hooks/
│   │   │   ├── useMetrics.ts         # WebSocket live metrics hook
│   │   │   ├── useQueryTrace.ts      # Execution trace state management
│   │   │   └── useWebSocket.ts       # WS connection management
│   │   ├── types/
│   │   │   ├── plan.ts               # ExplainPlan TypeScript types
│   │   │   ├── metrics.ts            # Metrics payload types
│   │   │   └── trace.ts              # Execution trace types
│   │   ├── api/
│   │   │   └── hybriddb.ts           # Typed API client for Go HTTP server
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   └── vite.config.ts
│
├── embedding-service/           # Python FastAPI embedding microservice
│   ├── main.py                  # FastAPI app: /embed endpoint
│   ├── model.py                 # sentence-transformers model loader + cache
│   ├── requirements.txt
│   └── README.md
│
├── benchmarks/
│   ├── storage_bench_test.go    # Go benchmark: pager, buffer pool
│   ├── btree_bench_test.go      # Go benchmark: B+ tree insert/search
│   ├── query_bench_test.go      # Go benchmark: relational queries
│   ├── vector_bench_test.go     # Go benchmark: cosine similarity at scale
│   ├── hybrid_bench_test.go     # Go benchmark: full hybrid queries
│   └── README.md
│
├── datasets/
│   ├── generate.py              # Synthetic dataset generator
│   ├── research_papers_1k.json  # 1k papers with precomputed embeddings
│   ├── research_papers_10k.json # 10k papers with precomputed embeddings
│   ├── load.go                  # Go tool: bulk-load JSON dataset into HybridDB
│   └── README.md
│
├── docs/
│   ├── architecture.md          # System architecture narrative
│   ├── storage-format.md        # Page layout, tuple format, WAL format (binary diagrams)
│   ├── query-lifecycle.md       # End-to-end query trace
│   ├── hybrid-execution.md      # Hybrid planner: VF/FF strategies, RRF explanation
│   ├── btree-internals.md       # B+ tree: node format, split/merge walkthrough
│   ├── recovery.md              # WAL invariant, ARIES-simplified recovery steps
│   └── api-reference.md         # HTTP API reference
│
├── diagrams/
│   ├── architecture.excalidraw  # Full system architecture diagram
│   ├── module-dependencies.png  # Rendered dependency DAG
│   ├── btree-split.png          # Annotated B+ tree split example
│   ├── hybrid-execution.png     # VF vs FF execution flow
│   └── wal-recovery.png         # WAL record lifecycle
│
├── scripts/
│   ├── setup.sh                 # One-command dev environment setup
│   ├── build.sh                 # Build all binaries
│   ├── test.sh                  # Run full test suite
│   ├── bench.sh                 # Run benchmark suite
│   ├── demo.sh                  # Load demo dataset + start server + open UI
│   └── crash-test.sh            # Kill server mid-insert, verify recovery
│
├── test/
│   ├── integration/
│   │   ├── e2e_query_test.go    # End-to-end: insert → query → verify
│   │   ├── recovery_test.go     # Crash → restart → consistency check
│   │   ├── hybrid_test.go       # Hybrid query ground truth comparison
│   │   └── btree_stress_test.go # 10k random insert/delete invariant check
│   └── fixtures/
│       ├── queries.sql           # Canonical test queries
│       └── expected_results/     # JSON ground truth for test queries
│
├── .golangci.yml                # Linter config (golangci-lint)
├── go.mod
├── go.sum
├── Makefile                     # Top-level make targets
└── README.md                    # Project overview, quick start, architecture summary
```

### Folder Ownership and Dependency Rules

| Folder | Owns | May Import | Must Not Import |
|--------|------|------------|-----------------|
| `internal/storage/` | Disk I/O, page layout, tuple format | OS stdlib only | buffer, wal, index, executor, sql |
| `internal/buffer/` | Page cache, LRU | storage/pager | wal, index, executor, sql |
| `internal/wal/` | Durability log, recovery | storage/pager | buffer, index, executor, sql |
| `internal/index/` | B+ tree, vector index | storage, buffer, wal | executor, sql, server |
| `internal/catalog/` | Schema registry | storage, buffer | executor, sql (parser ok) |
| `internal/sql/` | Lexer, parser, planner | catalog | executor, storage internals |
| `internal/executor/` | Query operators | storage, buffer, index, catalog, vector, metrics | server, web |
| `internal/vector/` | Embedding client, similarity | (http stdlib) | executor, storage internals |
| `internal/metrics/` | Metric structs, collection | (no imports) | everything |
| `internal/server/` | HTTP handlers, WS | executor, metrics, trace | storage internals directly |
| `web/` | UI components | (TypeScript only) | Go code |
| `embedding-service/` | Python model | (Python only) | Go code |

---

## 4. Development Environment

### 4.1 Core Tooling

```bash
# Go (core engine)
go version >= 1.22
# Install: https://go.dev/dl/

# Node.js (React UI)
node version >= 20 LTS
npm version >= 10

# Python (embedding service)
python version >= 3.11
pip install -r embedding-service/requirements.txt

# golangci-lint (Go linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# dlv (Go debugger)
go install github.com/go-delve/delve/cmd/dlv@latest
```

### 4.2 IDE Recommendation

**VS Code** with these extensions:
- `golang.go` — Go language server (gopls), debugging, test runner
- `esbenp.prettier-vscode` — TypeScript/React formatting
- `ms-python.python` — Python language server
- `usernamehw.errorlens` — Inline error display (critical for debugging)
- `eamodio.gitlens` — Git blame, history

**Why VS Code over GoLand:**
- Free, runs everywhere
- The Go extension uses gopls which is the official Go language server
- Integrated terminal is essential for running tests and the server simultaneously

### 4.3 Debugging Tools

```bash
# Delve: Go debugger (for stepping through B+ tree splits, WAL replay)
dlv debug ./cmd/hybriddb -- --db /tmp/test.hdb

# pprof: Go profiler (for benchmarking phases 9)
go tool pprof cpu.prof
go tool pprof mem.prof

# xxd: hex dump of raw page files (essential for storage debugging)
xxd /tmp/test.hdb | head -100

# Custom page dump tool (build this in Phase 1):
go run ./cmd/hybriddb-cli dump-page --db /tmp/test.hdb --page 3
```

### 4.4 Testing Tools

```bash
# Standard Go test runner
go test ./... -v -race

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Property-based testing
go get pgregory.net/rapid  # fast property-based testing for Go

# Benchmarks
go test -bench=. -benchmem ./benchmarks/...
```

### 4.5 Formatter and Linter Config

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt          # standard formatting
    - govet          # suspicious constructs
    - errcheck       # unchecked errors
    - staticcheck    # correctness checks
    - unused         # unused code
    - gocritic       # code quality
    - exhaustive     # exhaustive switch statements (critical for TypeTag handling)

linters-settings:
  govet:
    check-shadowing: true
  errcheck:
    check-type-assertions: true  # panic on failed type assertions
```

### 4.6 Makefile Targets

```makefile
.PHONY: build test bench lint fmt clean demo setup

build:
	go build ./cmd/...

test:
	go test ./... -v -race -count=1

test-integration:
	go test ./test/integration/... -v -timeout 60s

bench:
	go test -bench=. -benchmem -count=3 ./benchmarks/...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	cd web && npm run format

clean:
	rm -rf /tmp/hybriddb-*.hdb /tmp/hybriddb-*.wal
	go clean ./...

demo:
	./scripts/demo.sh

setup:
	./scripts/setup.sh
```

### 4.7 Local Setup (scripts/setup.sh)

```bash
#!/bin/bash
set -e

echo "==> Installing Go dependencies"
go mod download

echo "==> Installing Node.js dependencies"
cd web && npm install && cd ..

echo "==> Setting up Python embedding service"
cd embedding-service
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
cd ..

echo "==> Verifying embedding model download"
python embedding-service/model.py --download-only

echo "==> Running initial test suite"
go test ./... -count=1

echo "==> Setup complete. Run 'make demo' to start HybridDB."
```

### 4.8 Docker Strategy

Docker is used only for the embedding service in development, to isolate Python dependencies:

```dockerfile
# embedding-service/Dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8001
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8001"]
```

The Go engine and React UI run natively (not in Docker) to keep the development loop fast. Debugging a containerized Go binary is significantly slower than using `dlv` directly.

---

## 5. Module Dependency Graph

### 5.1 Full Dependency DAG

```
                    ┌─────────────────────────────────────────┐
                    │           External Interfaces            │
                    │  (OS file I/O, HTTP stdlib, time stdlib) │
                    └──────────────────┬──────────────────────┘
                                       │
                    ┌──────────────────▼──────────────────────┐
                    │              storage/pager               │
                    │  Pager interface, DiskPager impl          │
                    └──────┬───────────────────────────────────┘
                           │
              ┌────────────┼────────────────────┐
              │            │                    │
    ┌─────────▼──────┐ ┌───▼─────────┐  ┌──────▼──────────┐
    │  storage/page  │ │ buffer/pool  │  │   wal/manager   │
    │  SlottedPage   │ │ BufferPool   │  │   WALManager    │
    │  PageHeader    │ │ LRU policy   │  │   Recovery      │
    └─────────┬──────┘ └───┬─────────┘  └──────┬──────────┘
              │             │                    │
    ┌─────────▼──────┐      │                    │
    │ storage/tuple  │      │                    │
    │ Tuple, Schema  │      │                    │
    └─────────┬──────┘      │                    │
              │             │                    │
              └──────┬──────┘                    │
                     │                           │
          ┌──────────▼───────────────────────────▼─────────┐
          │                  index/btree                    │
          │         BPlusTree, Node, Split, Merge           │
          └──────────────────────┬──────────────────────────┘
                                 │
          ┌──────────────────────▼──────────────────────────┐
          │                    catalog                       │
          │          TableSchema, ColumnDef, IndexDef        │
          └──────────────────────┬──────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────────┐
              │                  │                       │
    ┌─────────▼──────┐  ┌────────▼────────┐  ┌──────────▼─────┐
    │   sql/lexer    │  │   sql/parser     │  │  sql/planner   │
    │   Tokenizer    │  │   AST Builder    │  │  Rule Engine   │
    └─────────┬──────┘  └────────┬─────────┘  └──────────┬─────┘
              └─────────────────┬┘                        │
                                │               ┌─────────▼──────────┐
                                │               │  vector/client      │
                                │               │  EmbeddingClient    │
                                │               │  CosineSimiliarity  │
                                │               └─────────┬──────────┘
                                │                         │
                    ┌───────────▼─────────────────────────▼──────────┐
                    │                   executor                       │
                    │  SeqScan, IndexScan, Filter, Projection,        │
                    │  Sort, Limit, VectorScan, HybridScan, Fusion    │
                    └──────────────────────┬──────────────────────────┘
                                           │
                    ┌──────────────────────▼──────────────────────────┐
                    │                metrics + trace                   │
                    │          MetricsCollector, ExecutionTracer       │
                    └──────────────────────┬──────────────────────────┘
                                           │
                    ┌──────────────────────▼──────────────────────────┐
                    │                  server (HTTP + WS)              │
                    └──────────────────────┬──────────────────────────┘
                                           │
                    ┌──────────────────────▼──────────────────────────┐
                    │                  web (React UI)                  │
                    └─────────────────────────────────────────────────┘
```

### 5.2 Interface Contracts

```go
// storage/pager — Pager interface
type Pager interface {
    ReadPage(pageID uint32, buf []byte) error
    WritePage(pageID uint32, buf []byte) error
    AllocatePage() (uint32, error)
    FreePage(pageID uint32) error
    NumPages() uint32
    Sync() error
    Close() error
}

// buffer — BufferPool interface
type BufferPool interface {
    FetchPage(pageID uint32) (*Frame, error)
    UnpinPage(pageID uint32, isDirty bool) error
    FlushPage(pageID uint32) error
    NewPage() (*Frame, error)
    Metrics() BufferMetrics
}

// wal — WALManager interface
type WALManager interface {
    AppendRecord(rec *Record) (LSN, error)
    Flush(upToLSN LSN) error
    Checkpoint() error
    RecoverFrom(checkpointLSN LSN) error
    LastLSN() LSN
}

// executor — Operator interface (Volcano model)
type Operator interface {
    Open(ctx *ExecContext) error
    Next() (*tuple.Tuple, error)
    Close() error
    Schema() *tuple.Schema
    Metrics() OperatorMetrics
}

// vector — EmbeddingClient interface
type EmbeddingClient interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Dimensions() int
}
```

### 5.3 Anti-Patterns to Avoid

| Anti-Pattern | Why Dangerous | Enforcement |
|-------------|---------------|-------------|
| `executor` importing `storage/pager` directly | Executor should see only the buffer pool abstraction, never raw pages | golangci-lint `depguard` rule |
| `storage` importing `wal` | WAL is a caller of storage, not the other way | Code review |
| Any module importing `server` | Server is the top-level consumer, never a dependency | Module structure |
| Calling `EmbeddingClient` from inside `storage` or `buffer` | Embeddings are a query-time concern, not a storage concern | Architecture doc |
| Global mutable state in any `internal/` package | Breaks independent testability | P3 principle + linter |

---

## 6. Phase-Wise Execution Plan

---

### Phase 0 — Architecture & Setup (Week 1)

#### Goal
Establish the technical foundation: repository, tooling, interfaces, logging, metrics infrastructure, and coding standards. No storage code yet.

#### Why This Phase Exists
Starting to write storage code on Day 1 without interfaces defined leads to a codebase where every module is tightly coupled to every other. The first three days of a database project should be spent on architecture, not implementation.

#### Deliverables Checklist

- [ ] Repository created with full folder structure (Section 3)
- [ ] `go.mod` initialized: `module github.com/<user>/hybriddb`
- [ ] `Makefile` with all targets (`build`, `test`, `lint`, `fmt`, `demo`)
- [ ] `.golangci.yml` configured (Section 4.5)
- [ ] All interface files created (empty implementations, compilable)
- [ ] `internal/metrics/collector.go` — MetricsCollector struct with no-op methods
- [ ] `internal/trace/tracer.go` — ExecutionTracer with no-op event recording
- [ ] `internal/storage/tuple/types.go` — TypeTag constants defined
- [ ] Structured logging setup: `slog` (Go 1.21+ stdlib), JSON format
- [ ] `docs/architecture.md` — architecture diagram + module descriptions
- [ ] CI: `go build ./...` and `go test ./...` pass (trivially, on empty stubs)

#### Interfaces to Define (all files, all stubs)

```go
// All interface files created in this phase. Implementations are stubs:
// func (d *DiskPager) ReadPage(...) error { return nil }

internal/storage/pager/pager.go     → Pager interface
internal/buffer/pool.go             → BufferPool interface
internal/wal/wal.go                 → WALManager interface
internal/executor/operator.go       → Operator interface
internal/vector/client.go           → EmbeddingClient interface
```

#### Logging Setup

```go
// pkg/log/log.go
package log

import "log/slog"

var Logger *slog.Logger

func Init(level slog.Level) {
    Logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: level,
    }))
}

// Usage in every module:
log.Logger.Info("page allocated", "pageID", newID, "numPages", pager.NumPages())
log.Logger.Error("WAL checksum mismatch", "lsn", lsn, "expected", expected, "got", actual)
```

#### Metrics Infrastructure Setup

```go
// internal/metrics/collector.go
type QueryMetrics struct {
    QueryID        string
    TotalTimeMs    float64
    PagesRead      uint64
    PagesWritten   uint64
    RowsEmitted    uint64
    CacheHitRate   float64
    OperatorStats  []OperatorStats
}

type OperatorStats struct {
    OperatorName string
    TimeMs       float64
    RowsIn       uint64
    RowsOut      uint64
    PagesRead    uint64
}

// MetricsCollector is passed in ExecContext. Operators call it.
// In Phase 0 all methods are no-ops. Filled in Phase 8.
type MetricsCollector struct { ... }
func (m *MetricsCollector) RecordOperatorStats(stats OperatorStats) {}
```

#### TypeTag Constants (critical: define once, never change)

```go
// internal/storage/tuple/types.go
type TypeTag uint8

const (
    TypeINT32   TypeTag = 0x01
    TypeINT64   TypeTag = 0x02
    TypeFLOAT32 TypeTag = 0x03
    TypeVARCHAR TypeTag = 0x04
    TypeVECTOR  TypeTag = 0x05
    TypeNULL    TypeTag = 0x06
    TypeBOOL    TypeTag = 0x07
)
```

**Rationale for defining TypeTags in Phase 0:** These constants appear in both serialization (Phase 1) and the parser (Phase 4). Defining them once in Phase 0 prevents the two modules from inventing incompatible constants independently.

#### Do NOT Implement in Phase 0
- Any actual page I/O
- Any actual data structures (B+ tree nodes, WAL records)
- The SQL parser
- Any UI code

#### Demo Checkpoint
```
$ make build
→ Compiles with zero errors

$ make test
→ All tests pass (trivially, stubs)

$ make lint
→ Zero lint errors
```

---

### Phase 1 — Pager & Storage Foundation (Weeks 1-2)

#### Goal
Implement correct, persistent, page-based storage: the `Pager`, `SlottedPage`, and `Tuple` serialization. By the end of this phase, you can write a tuple to disk and read it back correctly, including after process restart.

#### Why This Phase Comes First
Every other component is a consumer of the storage layer. The buffer pool caches pages from the pager. The WAL writes to the pager. The B+ tree stores nodes as pages. You cannot build any of these until you know that page reads and writes are correct.

#### Modules to Build

**`internal/storage/pager/pager.go` — DiskPager**

```go
type DiskPager struct {
    file      *os.File
    pageSize  uint32
    numPages  uint32
    freePages []uint32   // free page list (reuse freed pages)
}

// File layout:
//   Byte 0..pageSize-1: DB header page (pageID = 0)
//   Byte N*pageSize .. (N+1)*pageSize-1: Page N
//
// Read/Write use pread/pwrite equivalents (os.File.ReadAt/WriteAt)
// for correct behavior under concurrent use (future) and no seek state.

func (p *DiskPager) ReadPage(pageID uint32, buf []byte) error {
    if uint32(len(buf)) != p.pageSize {
        return fmt.Errorf("buffer size %d != page size %d", len(buf), p.pageSize)
    }
    offset := int64(pageID) * int64(p.pageSize)
    _, err := p.file.ReadAt(buf, offset)
    return err
}
```

**`internal/storage/page/slotted.go` — SlottedPage**

```
Binary Layout (4096 bytes):
┌──────────────────────────────────┐ offset 0
│ PageHeader (32 bytes)            │
│   pageID      uint32  (4B)       │
│   pageType    uint8   (1B)       │
│   numSlots    uint16  (2B)       │
│   freeStart   uint16  (2B)       │  <- start of free space
│   freeEnd     uint16  (2B)       │  <- end of free space
│   pageLSN     uint64  (8B)       │
│   _reserved   [13]byte           │
├──────────────────────────────────┤ offset 32
│ Slot Directory                   │
│   [slotID: uint16]               │
│   [offset: uint16, len: uint16,  │
│    flags: uint8]    (5 bytes each)│
│   grows DOWN ↓                   │
├──────────────────────────────────┤
│                                  │
│         Free Space               │
│                                  │
├──────────────────────────────────┤
│   grows UP ↑                     │
│   Tuple Data (variable-length)   │
└──────────────────────────────────┘ offset 4095
```

```go
// Slot flags
const (
    SlotFlagActive  uint8 = 0x01
    SlotFlagDeleted uint8 = 0x02
)

func (p *SlottedPage) InsertTuple(data []byte) (slotID uint16, err error)
func (p *SlottedPage) GetTuple(slotID uint16) ([]byte, error)
func (p *SlottedPage) DeleteTuple(slotID uint16) error
func (p *SlottedPage) FreeSpace() uint16
func (p *SlottedPage) Compact() error   // reclaim fragmented space
```

**`internal/storage/tuple/tuple.go` — Tuple Serialization**

```go
type Schema struct {
    Columns []ColumnDef
}

type ColumnDef struct {
    Name      string
    Type      TypeTag
    VecDim    uint16  // only for TypeVECTOR
    Nullable  bool
}

type Tuple struct {
    Schema *Schema
    Values []Value   // len == len(Schema.Columns)
}

type Value struct {
    IsNull bool
    Int32  int32
    Int64  int64
    Float  float32
    Str    string
    Vec    []float32
    Bool   bool
}

// Serialization:
func (t *Tuple) Serialize() ([]byte, error)
func DeserializeTuple(data []byte, schema *Schema) (*Tuple, error)
```

**Serialization Wire Format:**
```
[numCols: uint8]
[nullBitmap: ceil(numCols/8) bytes]
for each non-null col:
  [typeTag: uint8]
  switch typeTag:
    INT32:   [val: int32 little-endian]
    INT64:   [val: int64 little-endian]
    FLOAT32: [val: float32 IEEE754 little-endian]
    VARCHAR: [len: uint16][bytes: len bytes]
    VECTOR:  [dims: uint16][vals: dims*4 bytes float32 little-endian]
    BOOL:    [val: uint8 (0 or 1)]
```

#### Tests to Write

```go
// pager_test.go
func TestPagerReadWriteRoundTrip(t *testing.T)
func TestPagerAllocateSequential(t *testing.T)
func TestPagerPersistsAcrossReopen(t *testing.T)   // reopen file, read same page
func TestPagerFreePage(t *testing.T)                // freed page is reallocated

// slotted_test.go
func TestInsertAndReadTuple(t *testing.T)
func TestDeleteTupleMarksInactive(t *testing.T)
func TestFreeSpaceAccountingIsAccurate(t *testing.T)
func TestInsertUntilFull(t *testing.T)             // verify overflow error
func TestCompactReclainsFragmentedSpace(t *testing.T)
func TestSlottedPageRoundTrip(t *testing.T)        // serialize page to bytes, deserialize

// tuple_test.go
func TestSerializeDeserializeAllTypes(t *testing.T)
func TestNullableColumns(t *testing.T)
func TestVectorColumnRoundTrip(t *testing.T)
func TestSerializationLengthDeterministic(t *testing.T)
```

**Property test for tuple serialization:**
```go
func TestTupleSerializationProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random tuple with random column types and values
        tuple := generateRandomTuple(t)
        data, err := tuple.Serialize()
        require.NoError(t, err)
        roundTripped, err := DeserializeTuple(data, tuple.Schema)
        require.NoError(t, err)
        require.Equal(t, tuple, roundTripped)
    })
}
```

#### Debugging Tools to Build in This Phase

```go
// cmd/hybriddb-cli: 'dump-page' subcommand
// Usage: hybriddb-cli dump-page --db test.hdb --page 2
// Output:
//   Page 2 (4096 bytes):
//   Header: pageID=2 numSlots=3 freeStart=92 freeEnd=3950 pageLSN=0
//   Slot 0: offset=3900 len=50 flags=ACTIVE
//   Slot 1: offset=3840 len=60 flags=ACTIVE
//   Slot 2: offset=3780 len=60 flags=DELETED
//   Tuples:
//     Slot 0: [INT32:1, VARCHAR:"HybridDB", INT32:2024]
//     Slot 1: [INT32:2, VARCHAR:"Vector DBs", INT32:2025]
```

#### Do NOT Implement in Phase 1
- Buffer pool (Phase 2)
- WAL (Phase 2)
- B+ tree (Phase 3)
- Any query layer

#### Demo Checkpoint
```
$ go test ./internal/storage/... -v
→ All tests pass

$ go run ./cmd/hybriddb-cli dump-page --db /tmp/test.hdb --page 0
→ Shows human-readable page dump
```

---

### Phase 2 — Buffer Pool & WAL (Week 3)

#### Goal
Implement the buffer pool (in-memory page cache with LRU eviction) and the WAL manager (append-only durability log with redo/undo recovery). By the end of this phase, you can demonstrate crash recovery.

#### Why Buffer Pool Before WAL
The WAL manager needs to flush dirty pages through the buffer pool when checkpointing. The buffer pool needs to know the current LSN (from the WAL) when marking pages dirty. Both depend on each other via their interfaces — but the buffer pool's interface is simpler and should be implemented first as a stub, then the WAL completed, then the buffer pool's WAL integration finalized.

**Build order within this phase:**
1. BufferPool (LRU, pin/unpin, dirty tracking) — no WAL integration yet
2. WALManager (append, flush, checkpoint)
3. BufferPool WAL integration (call WAL.Flush before evicting dirty pages)
4. Recovery logic (redo + undo)

#### Buffer Pool Implementation

```go
// internal/buffer/pool.go
type Frame struct {
    PageID   uint32
    Data     []byte          // PAGE_SIZE bytes
    PinCount int32
    IsDirty  bool
    LastUsed time.Time
}

type BufferPool struct {
    frames   []*Frame
    pageMap  map[uint32]*Frame
    lruList  *list.List              // *list.Element values are *Frame
    lruMap   map[uint32]*list.Element
    pager    pager.Pager
    wal      wal.WALManager          // for LSN-before-evict check
    mu       sync.Mutex
    metrics  BufferMetrics
}

// FetchPage: return pinned frame for pageID (from cache or disk)
func (bp *BufferPool) FetchPage(pageID uint32) (*Frame, error)

// UnpinPage: decrement pin count; mark dirty if modified
func (bp *BufferPool) UnpinPage(pageID uint32, isDirty bool) error

// evict: internal — find LRU unpinned frame, flush if dirty, load new page
func (bp *BufferPool) evict(newPageID uint32) (*Frame, error)
```

**LRU Eviction Detail:**
```go
func (bp *BufferPool) evict(newPageID uint32) (*Frame, error) {
    // Walk LRU list from back (least recently used) to front
    for e := bp.lruList.Back(); e != nil; e = e.Prev() {
        frame := e.Value.(*Frame)
        if frame.PinCount > 0 {
            continue  // cannot evict pinned frame
        }
        if frame.IsDirty {
            // WAL invariant: ensure WAL is flushed before writing dirty page
            if err := bp.wal.Flush(frame.PageLSN); err != nil {
                return nil, err
            }
            if err := bp.pager.WritePage(frame.PageID, frame.Data); err != nil {
                return nil, err
            }
            bp.metrics.DirtyEvictions++
        }
        // Reuse frame for new page
        delete(bp.pageMap, frame.PageID)
        bp.lruList.Remove(e)
        if err := bp.pager.ReadPage(newPageID, frame.Data); err != nil {
            return nil, err
        }
        frame.PageID = newPageID
        frame.IsDirty = false
        frame.PinCount = 1
        // ... add to front of LRU
        return frame, nil
    }
    return nil, fmt.Errorf("buffer pool exhausted: all %d frames are pinned", len(bp.frames))
}
```

#### WAL Manager Implementation

```go
// internal/wal/record.go
type RecordType uint8
const (
    RecordBEGIN    RecordType = 0x01
    RecordINSERT   RecordType = 0x02
    RecordUPDATE   RecordType = 0x03
    RecordDELETE   RecordType = 0x04
    RecordCOMMIT   RecordType = 0x05
    RecordABORT    RecordType = 0x06
    RecordCHECKPT  RecordType = 0x07
)

type Record struct {
    LSN         uint64
    TxnID       uint32
    Type        RecordType
    TableID     uint32
    PageID      uint32
    SlotID      uint16
    BeforeImage []byte
    AfterImage  []byte
    Checksum    uint32    // CRC32 of all preceding fields
}

// WAL file format:
// [FileHeader: 32 bytes]
// [RecordHeader: 4B length prefix][RecordBody: variable][RecordHeader][RecordBody]...
// Length prefix allows scanning forward and backward.
```

#### Recovery Algorithm

```go
// internal/wal/recovery.go
func Recover(walFile string, pager pager.Pager, checkpointLSN uint64) error {
    records := scanWALFrom(walFile, checkpointLSN)

    // REDO phase: apply all ops from committed txns
    committedTxns := findCommittedTxns(records)
    for _, rec := range records {
        if rec.Type == RecordINSERT || rec.Type == RecordUPDATE || rec.Type == RecordDELETE {
            if committedTxns[rec.TxnID] {
                page := readPage(pager, rec.PageID)
                if page.LSN < rec.LSN {  // effect not yet on disk
                    applyAfterImage(page, rec)
                    writePage(pager, rec.PageID, page)
                }
            }
        }
    }

    // UNDO phase: reverse uncommitted txns
    uncommittedTxns := findUncommittedTxns(records, committedTxns)
    for txnID := range uncommittedTxns {
        undoTxn(records, pager, txnID)
    }

    return nil
}
```

#### Tests to Write

```go
// pool_test.go
func TestFetchPageCacheHit(t *testing.T)
func TestFetchPageCacheMiss(t *testing.T)
func TestLRUEvictionOrder(t *testing.T)          // verify LRU frame evicted, not MRU
func TestDirtyPageFlushedBeforeEviction(t *testing.T)
func TestPinCountPreventsEviction(t *testing.T)
func TestAllFramesPinnedReturnsError(t *testing.T)
func TestBufferMetricsHitRate(t *testing.T)

// wal_test.go
func TestWALRecordRoundTrip(t *testing.T)
func TestWALRecordsAreMonotonicallyIncreasing(t *testing.T)
func TestChecksumDetectsCorruption(t *testing.T)
func TestFlushForcesDataToDisk(t *testing.T)

// recovery_test.go
func TestRecoveryReplaysMissingCommittedOps(t *testing.T)
func TestRecoveryUndoesUncommittedOps(t *testing.T)
func TestRecoveryIdempotent(t *testing.T)        // run recovery twice, same result
func TestCrashAfterWALBeforePageWrite(t *testing.T)  // simulate crash, verify redo
```

**Crash simulation test pattern:**
```go
func TestCrashDuringInsert(t *testing.T) {
    db := openTestDB(t)

    // Insert 50 rows and commit
    insertRows(db, 50, true)  // committed

    // Insert 30 more rows but CRASH before commit
    // Simulate by: write WAL records, mark pages dirty, do NOT write COMMIT record
    insertRowsPartial(db, 30)  // no commit

    // Reopen db (simulates restart)
    db.Close()
    db2 := openTestDB(t)  // triggers recovery

    // Verify: exactly 50 rows present
    count := countRows(db2, "test_table")
    assert.Equal(t, 50, count)
}
```

#### Demo Checkpoint
```
$ ./scripts/crash-test.sh
→ Inserts 50 rows, kills process, restarts
→ Recovery log shows: "REDO phase: 3 ops applied, UNDO phase: 2 txns rolled back"
→ Row count: 50 ✓
```

---

### Phase 3 — B+ Tree Index (Weeks 4-5)

#### Goal
Implement a correct, disk-persistent B+ tree index with insert, search, range scan, delete, leaf chaining, and WAL integration.

#### Why This Phase Is The Hardest

The B+ tree is the most implementation-complex component in HybridDB. It requires correctly handling:
- Node splits that propagate upward (potentially to the root)
- Node merges that pull down separator keys from the parent
- Leaf chain maintenance during splits and merges
- Page allocation and persistence for every new node
- WAL logging before every node modification

A bug in the split logic may not surface until a specific key count triggers a root split. A bug in the merge logic may only appear after hundreds of deletions. This is why the B+ tree gets two weeks and an aggressive test strategy.

#### Node Structure

```go
// internal/index/btree/node.go

const (
    NodeTypeInternal uint8 = 0x01
    NodeTypeLeaf     uint8 = 0x02
)

// Binary serialization of a node (stored in one 4096-byte page):
//
// [NodeHeader: 24 bytes]
//   nodeType:   uint8
//   numKeys:    uint16
//   parentID:   uint32     (0 = root/no parent)
//   nextLeafID: uint32     (leaves only; 0 = no next)
//   prevLeafID: uint32     (leaves only; 0 = no prev)
//   _reserved:  [9]byte
//
// [Keys: numKeys * keySize bytes]
//   For INT32 keys: 4 bytes each → max keys = (4096-24) / (4+8) ≈ 339 (internal)
//   For INT32 keys leaf:          → max keys = (4096-24) / (4+6) ≈ 407 (leaf)
//
// [Pointers (internal): (numKeys+1) * 4 bytes]  (child pageIDs)
// [Records (leaf): numKeys * 6 bytes]            (pageID: uint32 + slotID: uint16)

type RecordID struct {
    PageID uint32
    SlotID uint16
}

type InternalNode struct {
    PageID   uint32
    NumKeys  uint16
    ParentID uint32
    Keys     []int64   // use int64 to support both INT32 and INT64 keys
    Children []uint32  // pageIDs of child nodes; len = NumKeys+1
}

type LeafNode struct {
    PageID     uint32
    NumKeys    uint16
    ParentID   uint32
    NextLeafID uint32
    PrevLeafID uint32
    Keys       []int64
    Records    []RecordID
}
```

#### Split Algorithm

```go
// internal/index/btree/split.go

// splitLeaf handles the case when a leaf node is full.
// Returns the new leaf and the key to push up to the parent.
func (bt *BPlusTree) splitLeaf(leaf *LeafNode, newKey int64, newRID RecordID) (*LeafNode, int64, error) {
    // 1. Create new leaf page
    newPageID, err := bt.bufPool.NewPage()
    newLeaf := &LeafNode{PageID: newPageID.PageID}

    // 2. Combine existing keys + new key into a sorted slice
    allKeys, allRIDs := insertSorted(leaf.Keys, leaf.Records, newKey, newRID)

    // 3. Split at midpoint: lower half stays, upper half goes to newLeaf
    mid := len(allKeys) / 2
    leaf.Keys = allKeys[:mid]
    leaf.Records = allRIDs[:mid]
    newLeaf.Keys = allKeys[mid:]
    newLeaf.Records = allRIDs[mid:]

    // 4. Maintain leaf chain
    newLeaf.NextLeafID = leaf.NextLeafID
    newLeaf.PrevLeafID = leaf.PageID
    leaf.NextLeafID = newLeaf.PageID
    if newLeaf.NextLeafID != 0 {
        // Update the next leaf's PrevLeafID pointer
        bt.updatePrevLeafPointer(newLeaf.NextLeafID, newLeaf.PageID)
    }

    // 5. WAL log both modified pages before marking dirty
    bt.wal.AppendRecord(&wal.Record{Type: wal.RecordUPDATE, PageID: leaf.PageID, AfterImage: leaf.Serialize()})
    bt.wal.AppendRecord(&wal.Record{Type: wal.RecordINSERT, PageID: newLeaf.PageID, AfterImage: newLeaf.Serialize()})

    // 6. The key to push up = first key of the new leaf (copy up, not push up)
    return newLeaf, newLeaf.Keys[0], nil
}
```

#### Invariant Checker

```go
// internal/index/btree/invariants.go
// Call this after EVERY operation during testing. Never in production.

func (bt *BPlusTree) AssertInvariants(t testing.TB) {
    t.Helper()

    // 1. All leaf keys are in sorted ascending order
    // 2. Leaf chain traversal yields all keys in order
    // 3. All internal node keys are correct separators (left child max < key <= right child min)
    // 4. All keys findable by Search() are in the leaf chain
    // 5. No node has more than maxKeys keys
    // 6. No non-root node has fewer than minKeys keys
    // 7. All parent pointers are correct
    // 8. Leaf chain nextLeafID/prevLeafID are consistent
}
```

#### Tests to Write

```go
// btree_test.go
func TestInsertSequentialKeys(t *testing.T)          // 1..1000 in order
func TestInsertReverseKeys(t *testing.T)             // 1000..1 in reverse
func TestInsertRandomKeys(t *testing.T)              // 10000 random with invariant check
func TestLeafSplitCorrect(t *testing.T)              // force split, verify structure
func TestRootSplitCreatesNewRoot(t *testing.T)       // verify new root after root splits
func TestRangeScanReturnsCorrectKeys(t *testing.T)   // range [50, 100] on 0..200 insert
func TestDeleteLeafKey(t *testing.T)                 // delete from leaf, no underflow
func TestDeleteCausesBorrow(t *testing.T)            // delete causing borrow from sibling
func TestDeleteCausesMerge(t *testing.T)             // delete causing merge + parent update
func TestSearchMissingKeyReturnsNil(t *testing.T)
func TestPersistenceAcrossReopen(t *testing.T)       // close, reopen, search still works

// Property test
func TestBTreePropertyRandomOps(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        ops := rapid.SliceOf(rapid.SampledFrom([]string{"insert","delete","search"}))(t)
        // ... execute random ops, after each: AssertInvariants()
    })
}
```

#### B+ Tree Visualization Data Export

```go
// In btree.go: export tree structure as JSON for UI rendering
type NodeJSON struct {
    PageID   uint32     `json:"pageID"`
    IsLeaf   bool       `json:"isLeaf"`
    Keys     []int64    `json:"keys"`
    Children []uint32   `json:"children,omitempty"` // internal nodes
    Records  []string   `json:"records,omitempty"`  // leaf: "pageID:slotID"
    NextLeaf uint32     `json:"nextLeaf,omitempty"`
}

func (bt *BPlusTree) ExportJSON() []NodeJSON
```

#### Demo Checkpoint
```
$ go test ./internal/index/btree/... -v -run TestInsertRandom
→ 10000 random inserts, invariants checked after each: PASS

$ go run ./cmd/hybriddb-cli btree-dump --db /tmp/test.hdb --index category_idx
→ Renders ASCII B+ tree:
    [Internal: 250, 500, 750]
      [Leaf: 1,2..249] → [Leaf: 250,251..499] → [Leaf: 500..749] → [Leaf: 750..1000]
```

---

### Phase 4 — SQL Parser & Catalog (Weeks 6-7)

#### Goal
Implement the SQL lexer, recursive descent parser producing a typed AST, and the catalog (schema registry). By the end of this phase, you can parse any supported SQL statement and inspect table schemas.

#### Why Parser After Storage

The parser does not need the storage layer to function — it operates on strings. However, the parser is built in Phase 4 (not Phase 0) because:
1. The TypeTag constants (Phase 0) are needed for `VECTOR(384)` column type parsing.
2. The catalog schema (Phase 4) is needed by the query planner and executor.
3. Writing the parser first (before the execution engine) would leave you unable to test it end-to-end, increasing the risk of discovering late-breaking grammar bugs.

#### Lexer Implementation

```go
// internal/sql/lexer/token.go
type TokenType int

const (
    // Keywords
    TOKEN_SELECT  TokenType = iota
    TOKEN_FROM
    TOKEN_WHERE
    TOKEN_INSERT
    TOKEN_INTO
    TOKEN_VALUES
    TOKEN_CREATE
    TOKEN_TABLE
    TOKEN_INDEX
    TOKEN_ON
    TOKEN_DELETE
    TOKEN_UPDATE
    TOKEN_SET
    TOKEN_LIMIT
    TOKEN_ORDER
    TOKEN_BY
    TOKEN_ASC
    TOKEN_DESC
    TOKEN_AND
    TOKEN_OR
    TOKEN_NOT
    TOKEN_NULL
    TOKEN_SIMILAR   // for SIMILAR TO
    TOKEN_TO
    TOKEN_EXPLAIN
    TOKEN_VECTOR    // for VECTOR(384) type
    TOKEN_BETWEEN
    TOKEN_LIKE

    // Literals
    TOKEN_IDENT
    TOKEN_INT_LIT
    TOKEN_FLOAT_LIT
    TOKEN_STRING_LIT

    // Operators
    TOKEN_EQ       // =
    TOKEN_NEQ      // !=
    TOKEN_LT       // <
    TOKEN_GT       // >
    TOKEN_LTE      // <=
    TOKEN_GTE      // >=

    // Punctuation
    TOKEN_LPAREN   // (
    TOKEN_RPAREN   // )
    TOKEN_COMMA    // ,
    TOKEN_SEMICOLON
    TOKEN_STAR     // *
    TOKEN_DOT      // .

    TOKEN_EOF
)
```

**Lexer design:** Hand-written character-by-character scanner. Do NOT use `regexp` for tokenization — it is slower and harder to debug. A simple switch statement on the current character is sufficient and transparent.

#### Parser Grammar (implemented subset)

```
stmt         := select_stmt | insert_stmt | create_stmt | delete_stmt | update_stmt | explain_stmt
select_stmt  := SELECT col_list FROM ident [WHERE where_clause] [ORDER BY ident [ASC|DESC]] [LIMIT int]
col_list     := STAR | ident (',' ident)*
where_clause := predicate (AND predicate)*
predicate    := ident op literal
              | ident BETWEEN literal AND literal
              | ident SIMILAR TO string_lit
op           := = | != | < | > | <= | >=
create_stmt  := CREATE TABLE ident '(' col_def_list ')'
              | CREATE INDEX ON ident '(' ident ')'
col_def      := ident type_tag [NOT NULL]
type_tag     := INT | BIGINT | FLOAT | VARCHAR | VECTOR '(' int ')'
insert_stmt  := INSERT INTO ident VALUES '(' value_list ')'
```

#### Catalog Implementation

```go
// internal/catalog/catalog.go

// The catalog is stored in Page 0 of the database file.
// It maintains an in-memory copy, loaded at startup, persisted on every DDL.

type Catalog struct {
    tables  map[string]*TableSchema
    indexes map[string][]*IndexDef   // table name → indexes
    bufPool buffer.BufferPool
    mu      sync.RWMutex
}

type TableSchema struct {
    TableID   uint32
    Name      string
    Columns   []ColumnDef
    RootPage  uint32   // first data page (or 0 if empty)
    RowCount  uint64   // approximate, for selectivity estimation
}

type IndexDef struct {
    IndexID    uint32
    TableName  string
    ColumnName string
    Type       IndexType   // BTree or Vector (future)
    RootPageID uint32
}
```

#### Tests to Write

```go
// lexer_test.go
func TestLexerKeywords(t *testing.T)
func TestLexerIdentifiers(t *testing.T)
func TestLexerIntLiterals(t *testing.T)
func TestLexerStringLiterals(t *testing.T)
func TestLexerOperators(t *testing.T)
func TestLexerSIMILARTO(t *testing.T)       // multi-word keyword

// parser_test.go
func TestParseSimpleSelect(t *testing.T)
func TestParseSelectWithWhere(t *testing.T)
func TestParseSelectWithLimit(t *testing.T)
func TestParseHybridQuery(t *testing.T)     // WHERE year > 2024 AND embedding SIMILAR TO '...'
func TestParseCreateTable(t *testing.T)
func TestParseCreateIndex(t *testing.T)
func TestParseInsert(t *testing.T)
func TestParseInvalidSQL(t *testing.T)      // verify structured errors
func TestExplainPrefix(t *testing.T)

// catalog_test.go
func TestCreateTablePersists(t *testing.T)  // close, reopen, table still exists
func TestCreateIndex(t *testing.T)
func TestGetSchema(t *testing.T)
```

#### Do NOT Implement in Phase 4
- Query execution (Phase 5)
- Vector parsing (already handled by `TOKEN_VECTOR` + `TypeVECTOR`)
- JOINs — not in scope for v1

#### Demo Checkpoint
```
$ echo "SELECT title, year FROM research_papers WHERE year > 2024 LIMIT 5;" | go run ./cmd/hybriddb-cli parse
→ AST output:
  SelectStmt {
    Columns: [title, year]
    From: research_papers
    Where: AndPredicate {
      Left: CompPredicate { Col: year, Op: >, Val: 2024 }
    }
    Limit: 5
  }
```

---

### Phase 5 — Query Execution Engine (Weeks 6-7)

#### Goal
Implement the Volcano iterator model with standard relational operators: SeqScan, IndexScan, Filter, Projection, Sort, Limit, and the ExecContext.

#### Execution Context

```go
// internal/executor/context.go
type ExecContext struct {
    TxnID     uint32
    BufPool   buffer.BufferPool
    WAL       wal.WALManager
    Catalog   *catalog.Catalog
    Metrics   *metrics.MetricsCollector
    Tracer    *trace.ExecutionTracer
    QueryID   string
}
```

#### SeqScan Implementation

```go
// internal/executor/seqscan.go
type SeqScan struct {
    ctx       *ExecContext
    tableName string
    schema    *tuple.Schema
    pageID    uint32     // current page being scanned
    slotID    uint16     // current slot in current page
    frame     *buffer.Frame
    metrics   OperatorMetrics
}

func (s *SeqScan) Open(ctx *ExecContext) error {
    s.ctx = ctx
    tbl, err := ctx.Catalog.GetTable(s.tableName)
    if err != nil { return err }
    s.schema = tbl.Schema()
    s.pageID = tbl.RootPage
    s.slotID = 0
    s.metrics.OpenStart = time.Now()
    return s.pinCurrentPage()
}

func (s *SeqScan) Next() (*tuple.Tuple, error) {
    for {
        page := page.NewSlottedPage(s.frame.Data)
        for int(s.slotID) < page.NumSlots() {
            slotID := s.slotID
            s.slotID++
            data, err := page.GetTuple(slotID)
            if err != nil || data == nil { continue }
            slot := page.GetSlot(slotID)
            if slot.Flags&page.SlotFlagDeleted != 0 { continue }  // skip deleted
            t, err := tuple.DeserializeTuple(data, s.schema)
            if err != nil { return nil, err }
            s.metrics.RowsOut++
            s.metrics.PagesRead = uint64(s.pageID - /* startPage */ 0 + 1)
            ctx.Tracer.RecordRow(s.ctx.QueryID, "SeqScan", s.pageID, slotID)
            return t, nil
        }
        // Move to next page
        if !s.advancePage() { return nil, nil }  // exhausted
    }
}
```

#### Operator Execution Trace

```go
// internal/trace/tracer.go
type EventType string

const (
    EventOperatorOpen  EventType = "OperatorOpen"
    EventOperatorClose EventType = "OperatorClose"
    EventRowEmitted    EventType = "RowEmitted"
    EventPageRead      EventType = "PageRead"
    EventCacheHit      EventType = "CacheHit"
    EventCacheMiss     EventType = "CacheMiss"
)

type TraceEvent struct {
    Timestamp    time.Time
    QueryID      string
    OperatorName string
    EventType    EventType
    PageID       uint32
    SlotID       uint16
    RowCount     uint64
    Metadata     map[string]any
}
```

#### Filter Operator With Predicate Evaluation

```go
// internal/executor/filter.go
type Filter struct {
    child     Operator
    predicate ast.Predicate
}

// Predicate evaluation handles all AST predicate types:
func evaluatePredicate(pred ast.Predicate, t *tuple.Tuple) (bool, error) {
    switch p := pred.(type) {
    case *ast.CompPredicate:
        colVal, err := t.GetValue(p.Column)
        if err != nil { return false, err }
        return compareValues(colVal, p.Op, p.Value)

    case *ast.RangePredicate:
        colVal, err := t.GetValue(p.Column)
        if err != nil { return false, err }
        lo, _ := compareValues(colVal, ">=", p.Lo)
        hi, _ := compareValues(colVal, "<=", p.Hi)
        return lo && hi, nil

    case *ast.AndPredicate:
        left, err := evaluatePredicate(p.Left, t)
        if err != nil || !left { return left, err }
        return evaluatePredicate(p.Right, t)

    case *ast.SimilarToPredicate:
        // This predicate is NOT evaluated here.
        // SimilarToPredicate is handled by the HybridScan operator (Phase 7).
        // If it appears in a Filter, it means the planner made an error.
        return false, fmt.Errorf("SimilarToPredicate must be handled by HybridScan, not Filter")

    default:
        return false, fmt.Errorf("unknown predicate type: %T", pred)
    }
}
```

#### Tests to Write

```go
// executor_test.go
func TestSeqScanReturnsAllTuples(t *testing.T)
func TestSeqScanEmptyTable(t *testing.T)
func TestIndexScanPointLookup(t *testing.T)
func TestIndexScanRangeScan(t *testing.T)
func TestFilterEqualityPredicate(t *testing.T)
func TestFilterRangePredicate(t *testing.T)
func TestFilterANDPredicate(t *testing.T)
func TestProjectionSelectsColumns(t *testing.T)
func TestSortAscending(t *testing.T)
func TestSortDescending(t *testing.T)
func TestLimitTruncatesResults(t *testing.T)
func TestEndToEndSelectQuery(t *testing.T)    // full pipeline: parse → plan → exec
```

#### Demo Checkpoint
```sql
-- Execute via CLI:
CREATE TABLE papers (id INT, title VARCHAR, year INT, category VARCHAR);
INSERT INTO papers VALUES (1, 'HybridDB', 2024, 'database');
INSERT INTO papers VALUES (2, 'Vector Search', 2025, 'ml');
SELECT title, year FROM papers WHERE year > 2023 ORDER BY year DESC LIMIT 5;
```
```
→ Result:
   title          | year
   Vector Search  | 2025
   HybridDB       | 2024
```

---

### Phase 6 — Vector Engine (Week 8)

#### Goal
Add vector column support, implement brute-force cosine similarity search, and integrate the embedding service.

#### Embedding Service (Python FastAPI)

```python
# embedding-service/main.py
from fastapi import FastAPI
from pydantic import BaseModel
from model import EmbeddingModel

app = FastAPI()
model = EmbeddingModel()  # loads all-MiniLM-L6-v2 at startup

class EmbedRequest(BaseModel):
    text: str

class EmbedResponse(BaseModel):
    embedding: list[float]
    dimensions: int

@app.post("/embed", response_model=EmbedResponse)
async def embed(req: EmbedRequest) -> EmbedResponse:
    vec = model.embed(req.text)
    return EmbedResponse(embedding=vec.tolist(), dimensions=len(vec))

@app.get("/health")
async def health():
    return {"status": "ok", "model": "all-MiniLM-L6-v2", "dimensions": 384}
```

```python
# embedding-service/model.py
from sentence_transformers import SentenceTransformer
import numpy as np

class EmbeddingModel:
    def __init__(self):
        self.model = SentenceTransformer('all-MiniLM-L6-v2')
        self._cache = {}  # simple in-process cache

    def embed(self, text: str) -> np.ndarray:
        if text in self._cache:
            return self._cache[text]
        vec = self.model.encode(text, normalize_embeddings=True)
        self._cache[text] = vec
        return vec
```

#### Go Embedding Client

```go
// internal/vector/client.go
type HTTPEmbeddingClient struct {
    baseURL    string
    httpClient *http.Client
    cache      map[string][]float32   // in-process cache for identical queries
    dims       int
}

func (c *HTTPEmbeddingClient) Embed(text string) ([]float32, error) {
    if cached, ok := c.cache[text]; ok {
        return cached, nil
    }
    body, _ := json.Marshal(map[string]string{"text": text})
    resp, err := c.httpClient.Post(c.baseURL+"/embed", "application/json", bytes.NewReader(body))
    // ... parse response, cache result
}
```

#### VectorScan Operator

```go
// internal/executor/vectorscan.go
type VectorScan struct {
    ctx        *ExecContext
    tableName  string
    colName    string    // the VECTOR column
    queryText  string    // raw query text; will be embedded at Open()
    queryVec   []float32
    topK       int
    results    []*ScoredTuple  // populated at Open() by brute-force scan
    cursor     int
}

type ScoredTuple struct {
    Tuple *tuple.Tuple
    Score float32   // cosine similarity
}

func (vs *VectorScan) Open(ctx *ExecContext) error {
    vs.ctx = ctx

    // 1. Get query embedding
    queryVec, err := ctx.EmbedClient.Embed(vs.queryText)
    if err != nil { return err }
    vs.queryVec = queryVec

    // 2. Brute-force scan: compute cosine similarity for every tuple
    heap := newMinHeap(vs.topK)  // min-heap keyed by similarity score
    scan := &SeqScan{tableName: vs.tableName}
    scan.Open(ctx)
    for {
        t, err := scan.Next()
        if err != nil { return err }
        if t == nil { break }
        vec, err := t.GetVector(vs.colName)
        if err != nil { continue }  // skip tuples without vector
        score := cosine(vec, queryVec)
        heap.push(&ScoredTuple{Tuple: t, Score: score})
    }
    scan.Close()

    // 3. Sort top-K results descending
    vs.results = heap.sortedDescending()
    vs.cursor = 0
    return nil
}

func (vs *VectorScan) Next() (*tuple.Tuple, error) {
    if vs.cursor >= len(vs.results) { return nil, nil }
    result := vs.results[vs.cursor]
    vs.cursor++
    // Inject similarity_score as a virtual column
    result.Tuple.SetVirtualColumn("similarity_score", result.Score)
    return result.Tuple, nil
}
```

#### Cosine Similarity

```go
// internal/vector/cosine.go
func CosineSimilarity(a, b []float32) float32 {
    if len(a) != len(b) {
        panic(fmt.Sprintf("dimension mismatch: %d vs %d", len(a), len(b)))
    }
    var dot, normA, normB float32
    for i := range a {
        dot  += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    if normA == 0 || normB == 0 { return 0 }
    return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
```

#### Tests to Write

```go
// cosine_test.go
func TestCosineIdenticalVectors(t *testing.T)       // expect 1.0
func TestCosineOrthogonalVectors(t *testing.T)      // expect 0.0
func TestCosineOppositeVectors(t *testing.T)        // expect -1.0
func TestCosineKnownExample(t *testing.T)           // hardcoded vectors with known answer
func TestCosineSymmetric(t *testing.T)              // cosine(a,b) == cosine(b,a)
func TestCosineDimensionMismatchPanics(t *testing.T)

// vectorscan_test.go
func TestVectorScanReturnsTopK(t *testing.T)
func TestVectorScanRankingOrder(t *testing.T)       // verify descending score order
func TestVectorScanWithMockEmbedClient(t *testing.T) // no embedding service needed
func TestVectorScanRecall(t *testing.T)             // compare to exhaustive search
```

#### Demo Checkpoint
```sql
SELECT title, similarity_score
FROM research_papers
WHERE embedding SIMILAR TO 'graph neural networks'
LIMIT 5;
```
```
→ Result (with real embeddings):
   title                              | similarity_score
   Graph Attention Networks           | 0.923
   Message Passing Neural Networks    | 0.891
   GCN for Node Classification        | 0.876
   ...
```

---

### Phase 7 — Hybrid Query Engine (Weeks 9-10)

#### Goal
Implement the hybrid query planner (strategy selection, selectivity estimation) and the HybridScan operator (VF/FF execution with RRF and WLC rank fusion).

#### Hybrid Query Detection in Planner

```go
// internal/sql/planner/rules.go

func (p *Planner) BuildPhysicalPlan(stmt *ast.SelectStmt) (executor.Operator, error) {
    // Rule 1: Hybrid query detection
    structuredPreds, vectorPred := classifyPredicates(stmt.Where)

    if vectorPred != nil && len(structuredPreds) > 0 {
        return p.buildHybridPlan(stmt, structuredPreds, vectorPred)
    }
    if vectorPred != nil {
        return p.buildVectorPlan(stmt, vectorPred, stmt.Limit)
    }
    // ... standard relational planning
}

func (p *Planner) buildHybridPlan(
    stmt *ast.SelectStmt,
    structured []ast.Predicate,
    vector *ast.SimilarToPredicate,
) (executor.Operator, error) {
    selectivity := p.estimateSelectivity(stmt.From, structured)
    strategy := VECTOR_FIRST
    if selectivity < 0.1 {
        strategy = FILTER_FIRST
    }

    return &executor.HybridScan{
        TableName:        stmt.From,
        VectorCol:        vector.Column,
        QueryText:        vector.QueryText,
        StructuredPreds:  structured,
        Strategy:         strategy,
        TopK:             stmt.Limit,
        Kover:            stmt.Limit * 5,  // over-retrieve for VF strategy
        FusionAlgorithm:  executor.FusionRRF,
    }, nil
}
```

#### HybridScan Operator

```go
// internal/executor/hybridscan.go

type Strategy int
const (
    VECTOR_FIRST Strategy = iota
    FILTER_FIRST
)

type HybridScan struct {
    TableName       string
    VectorCol       string
    QueryText       string
    StructuredPreds []ast.Predicate
    Strategy        Strategy
    TopK            int
    Kover           int
    FusionAlgorithm FusionType
    results         []*ScoredTuple
    cursor          int
}

func (hs *HybridScan) Open(ctx *ExecContext) error {
    var candidates []*ScoredTuple
    var err error

    switch hs.Strategy {
    case VECTOR_FIRST:
        candidates, err = hs.executeVectorFirst(ctx)
    case FILTER_FIRST:
        candidates, err = hs.executeFilterFirst(ctx)
    }
    if err != nil { return err }

    hs.results = candidates
    hs.cursor = 0

    // Record hybrid execution metrics
    ctx.Metrics.RecordHybridStats(HybridStats{
        Strategy:            hs.Strategy.String(),
        CandidatesGenerated: len(candidates),
    })
    return nil
}

func (hs *HybridScan) executeFilterFirst(ctx *ExecContext) ([]*ScoredTuple, error) {
    // 1. Execute structured filter
    structuredPlan := buildStructuredPlan(hs.TableName, hs.StructuredPreds, ctx.Catalog)
    structuredPlan.Open(ctx)
    var filtered []*tuple.Tuple
    for {
        t, err := structuredPlan.Next()
        if t == nil || err != nil { break }
        filtered = append(filtered, t)
    }
    structuredPlan.Close()

    ctx.Metrics.RecordStage("filter_complete", len(filtered))

    // 2. Embed query
    queryVec, err := ctx.EmbedClient.Embed(hs.QueryText)
    if err != nil { return nil, err }

    // 3. Score all filtered candidates
    scored := make([]*ScoredTuple, 0, len(filtered))
    for _, t := range filtered {
        vec, err := t.GetVector(hs.VectorCol)
        if err != nil { continue }
        score := vector.CosineSimilarity(vec, queryVec)
        scored = append(scored, &ScoredTuple{Tuple: t, Score: score})
    }

    // 4. Sort by score descending, take top-K
    sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
    if len(scored) > hs.TopK { scored = scored[:hs.TopK] }

    ctx.Metrics.RecordStage("scored_and_ranked", len(scored))
    return scored, nil
}
```

#### Rank Fusion Implementations

```go
// internal/executor/fusion.go

// RRF: Reciprocal Rank Fusion (Cormack et al., 2009)
// Combines ranked lists from multiple retrieval signals.
// k=60 is the standard constant; robust to outliers.
func RRF(lists [][]*ScoredTuple, k float64) []*ScoredTuple {
    scores := map[*tuple.Tuple]float64{}
    for _, list := range lists {
        for rank, st := range list {
            scores[st.Tuple] += 1.0 / (k + float64(rank+1))
        }
    }
    var result []*ScoredTuple
    for t, score := range scores {
        result = append(result, &ScoredTuple{Tuple: t, Score: float32(score)})
    }
    sort.Slice(result, func(i, j int) bool { return result[i].Score > result[j].Score })
    return result
}

// WLC: Weighted Linear Combination
// Requires score normalization first.
func WLC(semantic []*ScoredTuple, alpha, beta float64) []*ScoredTuple {
    normalized := normalizeScores(semantic)
    for _, st := range normalized {
        // filter_score = 1.0 (candidates are already filtered in FF strategy)
        st.Score = float32(alpha*float64(st.Score) + beta*1.0)
    }
    sort.Slice(normalized, func(i, j int) bool { return normalized[i].Score > normalized[j].Score })
    return normalized
}

func normalizeScores(list []*ScoredTuple) []*ScoredTuple {
    if len(list) == 0 { return list }
    min, max := list[len(list)-1].Score, list[0].Score
    if max == min { return list }
    for _, st := range list {
        st.Score = (st.Score - min) / (max - min)
    }
    return list
}
```

#### Tests to Write

```go
// hybridscan_test.go
func TestHybridVectorFirstStrategy(t *testing.T)
func TestHybridFilterFirstStrategy(t *testing.T)
func TestHybridStrategySelection(t *testing.T)    // verify correct strategy chosen by selectivity
func TestHybridRecallVsGroundTruth(t *testing.T)  // compare to exhaustive search
func TestHybridRankFusionRRF(t *testing.T)
func TestHybridRankFusionWLC(t *testing.T)
func TestHybridWithHighlySelectiveFilter(t *testing.T)  // e.g., 2% selectivity → FILTER_FIRST

// fusion_test.go
func TestRRFKnownRankings(t *testing.T)   // hardcoded expected RRF scores
func TestRRFConsistentWithSingleList(t *testing.T)
func TestNormalizeScores(t *testing.T)
```

#### Demo Checkpoint
```sql
EXPLAIN SELECT title, year, similarity_score
FROM research_papers
WHERE year > 2024 AND category = 'database'
AND embedding SIMILAR TO 'vector indexing optimization'
LIMIT 5;
```
```
→ Plan:
  Projection([title, year, similarity_score])
    └── Limit(5)
          └── HybridScan [strategy=FILTER_FIRST, selectivity=0.024]
                ├── VectorScore(queryVec, col=embedding)
                └── Filter(year > 2024 AND category = 'database')
                      └── IndexScan(papers, idx_category, 'database')

→ Estimated filter output: 240 / 10000 rows → FILTER_FIRST selected
```

---

### Phase 8 — Explainability & Observability (Week 11)

#### Goal
Wire up the metrics collection (built in Phase 0, extended in each phase) into a real-time HTTP/WebSocket server, and build the React visualization UI.

#### Go HTTP Server

```go
// internal/server/server.go
func NewServer(executor *executor.Engine, metrics *metrics.Collector) *Server

// Endpoints:
// POST /query           → execute SQL, return results + execution trace
// GET  /explain?q=...   → return plan tree JSON without executing
// GET  /metrics         → current buffer pool and system metrics
// GET  /btree/:name     → B+ tree structure export for visualization
// GET  /wal/tail        → last N WAL records
// WS   /ws/metrics      → WebSocket: push live buffer pool metrics
```

```go
// POST /query response:
type QueryResponse struct {
    Results []map[string]any    `json:"results"`
    Plan    *PlanNode           `json:"plan"`
    Metrics *QueryMetrics       `json:"metrics"`
    Trace   []TraceEvent        `json:"trace"`
}

type PlanNode struct {
    Operator   string      `json:"operator"`
    TimeMs     float64     `json:"timeMs"`
    RowsIn     uint64      `json:"rowsIn"`
    RowsOut    uint64      `json:"rowsOut"`
    PagesRead  uint64      `json:"pagesRead"`
    Metadata   map[string]any `json:"metadata"`  // strategy, predicate text, etc.
    Children   []*PlanNode `json:"children"`
}
```

#### WebSocket Live Metrics

```go
// internal/server/ws.go
// Every 500ms, push current buffer pool state to all connected clients:
type BufferPoolSnapshot struct {
    Timestamp   int64          `json:"timestamp"`
    Frames      []FrameState   `json:"frames"`
    HitRate     float64        `json:"hitRate"`
    Evictions   uint64         `json:"evictions"`
    DirtyCount  int            `json:"dirtyCount"`
    PinnedCount int            `json:"pinnedCount"`
}

type FrameState struct {
    FrameID  int    `json:"frameID"`
    PageID   uint32 `json:"pageID"`
    IsDirty  bool   `json:"isDirty"`
    IsPinned bool   `json:"isPinned"`
    IsEmpty  bool   `json:"isEmpty"`
}
```

#### React UI Components

**ExplainPlanTree.tsx** — uses React Flow:
```typescript
// Convert PlanNode JSON tree to React Flow nodes + edges
function planToFlow(plan: PlanNode, parentId?: string): { nodes: Node[], edges: Edge[] } {
    const id = `${plan.operator}-${Math.random()}`;
    const node: Node = {
        id,
        data: {
            label: `${plan.operator}`,
            timeMs: plan.timeMs,
            rowsIn: plan.rowsIn,
            rowsOut: plan.rowsOut,
        },
        position: { x: 0, y: 0 },  // auto-layout via dagre
        type: 'operatorNode',
    };
    const edges: Edge[] = parentId ? [{ id: `e-${parentId}-${id}`, source: parentId, target: id }] : [];
    const childResults = plan.children?.flatMap(c => planToFlow(c, id).nodes) ?? [];
    return { nodes: [node, ...childResults], edges };
}
```

**BufferPoolHeatmap.tsx** — live via WebSocket:
```typescript
function BufferPoolHeatmap() {
    const snapshot = useMetrics();  // WS hook
    return (
        <div className="grid grid-cols-16 gap-1">
            {snapshot.frames.map(f => (
                <div key={f.frameID}
                     className={cn(
                         "w-6 h-6 rounded",
                         f.isEmpty  && "bg-gray-200",
                         f.isPinned && "bg-red-400",
                         f.isDirty  && "bg-orange-400",
                         !f.isEmpty && !f.isPinned && !f.isDirty && "bg-blue-400"
                     )}
                     title={`Frame ${f.frameID}: Page ${f.pageID}`}
                />
            ))}
        </div>
    );
}
```

#### Tests to Write

```go
// server integration tests
func TestQueryEndpointReturnsResults(t *testing.T)
func TestExplainEndpointReturnsPlan(t *testing.T)
func TestMetricsEndpointReturnsJSON(t *testing.T)
func TestWebSocketPushesUpdates(t *testing.T)
```

#### Demo Checkpoint
```
$ make demo   # loads 10k papers, starts server, opens browser

Browser shows:
├── Query Editor: type SQL, execute, see results
├── Execution Plan Tree: animated React Flow diagram
├── Buffer Pool Heatmap: live frame states
├── WAL Log Viewer: scrolling record stream
└── Metrics Dashboard: hit rate, latency charts
```

---

### Phase 9 — Benchmarking & Hardening (Week 12)

#### Goal
Run the full benchmark suite, identify and fix bugs found under load, stress test the B+ tree and WAL, and produce documented performance numbers.

#### Benchmark Suite

```go
// benchmarks/storage_bench_test.go
func BenchmarkPagerReadPage(b *testing.B)
func BenchmarkPagerWritePage(b *testing.B)
func BenchmarkBufferPoolHitRate(b *testing.B)         // 90% hit rate workload
func BenchmarkSlottedPageInsert(b *testing.B)

// benchmarks/btree_bench_test.go
func BenchmarkBTreeInsert_1k(b *testing.B)
func BenchmarkBTreeInsert_10k(b *testing.B)
func BenchmarkBTreeInsert_100k(b *testing.B)
func BenchmarkBTreeSearch(b *testing.B)
func BenchmarkBTreeRangeScan_1pct(b *testing.B)      // 1% of keys
func BenchmarkBTreeRangeScan_10pct(b *testing.B)

// benchmarks/vector_bench_test.go
func BenchmarkCosineSimilarity_384dim(b *testing.B)
func BenchmarkVectorScan_1k(b *testing.B)
func BenchmarkVectorScan_10k(b *testing.B)
func BenchmarkVectorScan_100k(b *testing.B)

// benchmarks/hybrid_bench_test.go
func BenchmarkHybridQuery_FilterFirst_HighSelectivity(b *testing.B)
func BenchmarkHybridQuery_VectorFirst_LowSelectivity(b *testing.B)
func BenchmarkHybridQuery_CompareStrategies(b *testing.B)
```

#### Stress Tests

```bash
# B+ tree stress test: 10k random insert + delete with invariant checks after each op
$ go test ./test/integration/... -run TestBTreeStress -timeout 300s -v

# WAL durability stress test: random crash points
$ go test ./test/integration/... -run TestRecoveryStress -timeout 120s -v

# Full hybrid query stress: 1000 queries, verify recall
$ go test ./test/integration/... -run TestHybridQueryStress -v
```

#### Expected Benchmark Results (Targets)

| Benchmark | Target | Hard Limit |
|-----------|--------|------------|
| B+ tree point lookup (10k rows) | < 3 ms | 10 ms |
| Full table scan (10k rows) | < 30 ms | 100 ms |
| Vector scan 384-dim (10k rows) | < 50 ms | 200 ms |
| Hybrid query FF (10k rows, 2% selectivity) | < 100 ms | 500 ms |
| Hybrid query VF (10k rows, 80% selectivity) | < 200 ms | 500 ms |
| WAL replay (10k records) | < 1 s | 3 s |
| Buffer pool hit rate (repeat reads) | > 80% | N/A |

---

### Phase 10 — Final Demo & Packaging (Weeks 13-14)

#### Goal
Polish the demo experience, finalize documentation, and prepare for viva.

#### Deliverables Checklist

- [ ] `scripts/demo.sh` — one command: generate data, start server, open browser
- [ ] Dataset: 10k research papers pre-loaded with embeddings (offline, no embedding service cold start in demo)
- [ ] `docs/architecture.md` — final version with all diagrams
- [ ] `README.md` — polished with demo GIF, setup instructions, architecture summary
- [ ] All benchmarks run and results documented in `benchmarks/README.md`
- [ ] Viva preparation notes (Section 15)
- [ ] Demo video (2-3 minutes, screen recording)

---

## 7. Week-by-Week Roadmap

| Week | Phase | Primary Work | Milestone |
|------|-------|--------------|-----------|
| **1** | 0 | Repo setup, interfaces, logging, TypeTag constants | `make build` passes, all stubs compile |
| **1-2** | 1 | Pager, SlottedPage, Tuple serialization | Tuple round-trip across file reopen |
| **3** | 2 | Buffer pool (LRU), WAL, recovery | Crash recovery demo |
| **4** | 3a | B+ tree: node structure, insert, leaf split | 1000 sequential inserts + correct search |
| **5** | 3b | B+ tree: root split, delete, merge, range scan | 10k random ops with invariant checks |
| **6** | 4+5a | SQL Lexer, Parser, AST, Catalog | Parse all supported statements |
| **7** | 5b | SeqScan, IndexScan, Filter, Projection, Sort, Limit | End-to-end SELECT query correct |
| **8** | 6 | Embedding service, VectorScan, cosine similarity | Semantic search demo |
| **9** | 7a | Hybrid planner: strategy selection, selectivity | Strategy selection correct by selectivity |
| **10** | 7b | HybridScan operator, RRF, WLC, full integration | Full hybrid query end-to-end |
| **11** | 8 | HTTP server, WebSocket, React UI | Live dashboard running |
| **12** | 9 | Benchmarks, stress tests, bug fixes | All benchmarks documented |
| **13** | 10a | Demo polish, `demo.sh`, README | One-command demo works |
| **14** | 10b | Viva prep, final testing | Presentation-ready |

### Critical Path

```
Week 1 (setup) → Weeks 1-2 (storage) → Week 3 (buffer+WAL) → Weeks 4-5 (B+tree)
→ Weeks 6-7 (SQL+executor) → Week 8 (vector) → Weeks 9-10 (hybrid)
→ Week 11 (UI) → Week 12 (bench) → Weeks 13-14 (demo+viva)

NO PHASE MAY START UNTIL PREVIOUS PHASE'S TESTS PASS.
```

### Refactor Checkpoints

| After Phase | Refactor Focus |
|-------------|---------------|
| Phase 2 | Review buffer pool and WAL interface contracts; add missing metrics hooks |
| Phase 5 | Review executor operator implementations for consistency; ensure ExecContext is properly threaded |
| Phase 7 | Review HybridScan for code duplication; extract shared rank fusion utilities |
| Phase 9 | Final cleanup: remove dead code, add missing docstrings |

---

## 8. Testing Strategy

### 8.1 Test Philosophy

Three test categories, strict ordering:
1. **Unit tests**: one module, no external dependencies, fast (< 100ms)
2. **Integration tests**: two or more modules, may use temp files, moderate speed (< 10s)
3. **System tests**: full database, real data, slow (< 60s)

Unit tests are written alongside the code. Integration tests are written at the end of each phase. System tests are written in Phase 9.

### 8.2 Property Testing (pgregory.net/rapid)

Property tests express invariants that must hold for ALL inputs, not just examples:

```go
// Tuple serialization: serialize(deserialize(x)) == x
func TestTupleSerializationIsLossless(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        schema := generateRandomSchema(t)
        tuple := generateRandomTuple(t, schema)
        data, _ := tuple.Serialize()
        rt, _ := tuple.DeserializeTuple(data, schema)
        require.Equal(t, tuple, rt, "round-trip failed")
    })
}

// B+ tree: for all insert/search sequences, search(insert(x)) finds x
func TestBTreeSearchFindsInserted(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        keys := rapid.SliceOf(rapid.Int64())(t)
        bt := newTestBTree(t)
        for _, k := range keys {
            bt.Insert(k, fakeRID(k))
        }
        bt.AssertInvariants(t)
        for _, k := range keys {
            rid, found := bt.Search(k)
            require.True(t, found)
            require.Equal(t, fakeRID(k), rid)
        }
    })
}
```

### 8.3 Fuzz Testing

```go
// Fuzz the SQL parser: any byte sequence must not panic the parser
func FuzzParser(f *testing.F) {
    f.Add([]byte("SELECT * FROM t WHERE x > 5"))
    f.Fuzz(func(t *testing.T, data []byte) {
        // Must not panic. May return parse error. Must not hang.
        defer func() { recover() }()  // catch panics
        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()
        parseWithContext(ctx, string(data))
    })
}

// Fuzz the WAL record parser: corrupted records must not crash recovery
func FuzzWALRecovery(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        _, err := wal.ParseRecord(data)
        // Must return error on malformed input, never panic
        if err != nil { return }
    })
}
```

### 8.4 Deterministic Crash Testing

```go
// test/integration/recovery_test.go
type CrashPoint int
const (
    AfterWALBeforePageWrite CrashPoint = iota
    DuringBTreeSplit
    AfterCommitBeforeCheckpoint
    DuringRecoveryRedo
)

func TestCrashAtAllPoints(t *testing.T) {
    for _, cp := range []CrashPoint{AfterWALBeforePageWrite, DuringBTreeSplit, AfterCommitBeforeCheckpoint} {
        t.Run(cp.String(), func(t *testing.T) {
            db := openTestDB(t)
            injectCrashPoint(db, cp)
            insertRows(db, 100, true)  // triggers crash at cp
            db2 := openTestDB(t)       // triggers recovery
            verifyConsistency(t, db2)
        })
    }
}
```

### 8.5 Testing Matrix

| Component | Unit | Property | Fuzz | Integration | Crash |
|-----------|------|----------|------|-------------|-------|
| Pager | ✓ | - | - | ✓ | - |
| SlottedPage | ✓ | ✓ | - | ✓ | - |
| Tuple serialization | ✓ | ✓ | - | - | - |
| Buffer Pool | ✓ | - | - | ✓ | - |
| WAL | ✓ | - | ✓ | ✓ | ✓ |
| B+ Tree | ✓ | ✓ | - | ✓ | ✓ |
| Parser | ✓ | - | ✓ | - | - |
| Catalog | ✓ | - | - | ✓ | - |
| Executor operators | ✓ | - | - | ✓ | - |
| VectorScan | ✓ | - | - | ✓ | - |
| HybridScan | ✓ | - | - | ✓ | - |
| Recovery | - | - | - | ✓ | ✓ |

---

## 9. Debugging Strategy

### 9.1 Page Corruption Debugging

```bash
# 1. Identify the corrupt page by running the page checker:
$ hybriddb-cli check-integrity --db mydb.hdb
→ "Page 47: slot directory checksum mismatch"
→ "Page 47: freeStart (4050) > freeEnd (3900) — impossible"

# 2. Hex dump the page:
$ xxd mydb.hdb | sed -n '$(( 47 * 16 )),$((48*16))p'

# 3. Dump WAL records that modified page 47:
$ hybriddb-cli wal-dump --db mydb.hdb --page 47
→ Shows all WAL records for page 47 in chronological order

# 4. Replay just those records into a test DB:
$ hybriddb-cli wal-replay --db mydb.hdb --page 47 --until-lsn 10042
→ Allows bisecting which operation caused the corruption
```

### 9.2 B+ Tree Split Debugging

```go
// Add split tracing in development mode:
// Set env var: HYBRIDDB_BTREE_TRACE=1

func (bt *BPlusTree) splitLeaf(leaf *LeafNode, ...) {
    if os.Getenv("HYBRIDDB_BTREE_TRACE") != "" {
        log.Logger.Debug("leaf split",
            "leafPageID", leaf.PageID,
            "leafKeys", leaf.Keys,
            "newKey", newKey,
            "splitPoint", len(leaf.Keys)/2,
        )
    }
    // ... split logic
}

// After split, call invariant checker:
if os.Getenv("HYBRIDDB_BTREE_VERIFY") != "" {
    bt.AssertInvariants(nil)  // nil = use log output instead of testing.TB
}
```

**Visual debugging:** Export the B+ tree as JSON and render it in the UI after every insert. The animated tree in Phase 8 is not just for the demo — it is a debugging tool during Phase 3.

### 9.3 WAL Replay Debugging

```go
// Enable step-by-step WAL replay logging:
// Set env var: HYBRIDDB_WAL_DEBUG=1

func Recover(...) error {
    for i, rec := range records {
        if os.Getenv("HYBRIDDB_WAL_DEBUG") != "" {
            log.Logger.Debug("replaying WAL record",
                "recordNum", i,
                "lsn", rec.LSN,
                "type", rec.Type.String(),
                "txnID", rec.TxnID,
                "pageID", rec.PageID,
            )
        }
        // ... apply record
    }
}
```

### 9.4 Hybrid Ranking Debugging

```go
// Dump intermediate ranking stages:
type HybridDebugLog struct {
    Strategy            string
    QueryText           string
    VectorCandidates    []RankedEntry  // (title, score) sorted by semantic similarity
    FilteredCandidates  []RankedEntry  // after filter applied
    FinalResults        []RankedEntry  // after rank fusion
}

// Access via: GET /debug/last-hybrid-query
// This is invaluable for understanding WHY a particular result ranked #1 vs #5.
```

### 9.5 Structured Logging Conventions

```go
// ALWAYS include these fields when logging storage operations:
log.Logger.Info("page read", "pageID", pageID, "bufferHit", hit, "duration_us", elapsed.Microseconds())

// ALWAYS include query ID in execution logs:
log.Logger.Info("operator next", "queryID", ctx.QueryID, "operator", "SeqScan", "rowsEmitted", count)

// NEVER log inside hot paths (inside Next() per-tuple loops) in non-debug mode.
// Use conditional:
if log.Logger.Enabled(ctx.Background(), slog.LevelDebug) {
    log.Logger.Debug("tuple emitted", ...)
}
```

---

## 10. Observability Strategy

### 10.1 Why Build Observability Early

Systems projects that add observability as an afterthought produce two outcomes:
1. The observability instruments the wrong things (added without understanding the execution flow)
2. The debugging cost of fixing Phase 9 bugs is 3x higher (no way to see what is happening)

By defining `MetricsCollector` and `ExecutionTracer` in Phase 0, every subsequent module adds metrics hooks at the time it is written, when the developer understands exactly what is worth measuring.

### 10.2 Metrics Architecture

```
                ┌─────────────────────────────┐
                │       ExecContext           │
                │   MetricsCollector ref      │
                └──────────────┬──────────────┘
                               │ passed to every operator
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
    ┌────▼────┐          ┌──────▼──────┐       ┌──────▼──────┐
    │SeqScan  │          │ IndexScan   │       │ HybridScan  │
    │records: │          │ records:    │       │ records:    │
    │ rows_out│          │ pages_read  │       │ candidates  │
    │ pages_rd│          │ index_hits  │       │ strategy    │
    └─────────┘          └─────────────┘       └─────────────┘
         │                     │                     │
         └─────────────────────┼─────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │  MetricsCollector   │
                    │  aggregates all     │
                    │  operator stats     │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  server/handlers.go │
                    │  serializes to JSON │
                    │  for HTTP + WS      │
                    └─────────────────────┘
```

### 10.3 Key Metrics to Expose

| Metric | Source | Why It Matters |
|--------|--------|----------------|
| Buffer pool hit rate | BufferPool | Shows whether working set fits in memory |
| Dirty page count | BufferPool | Shows pending writes; high = WAL pressure |
| Eviction count | BufferPool | Shows memory pressure |
| Operator rows_in / rows_out | Each operator | Shows filter selectivity visually |
| Operator time_ms | Each operator | Shows where query time is spent |
| Vector candidates generated | VectorScan | Shows K in top-K retrieval |
| Filter pass rate | HybridScan | Shows effectiveness of structured filter |
| WAL LSN | WALManager | Shows durability progress |
| Recovery time | Recovery | Shows WAL replay cost |

---

## 11. Benchmarking Strategy

### 11.1 Synthetic Dataset Generation

```python
# datasets/generate.py
import json
import random
from sentence_transformers import SentenceTransformer

model = SentenceTransformer('all-MiniLM-L6-v2')

CATEGORIES = ['database', 'ml', 'systems', 'networking', 'theory']
ABSTRACTS = [...]  # pool of 200 real arXiv abstracts

def generate(n: int, output_file: str):
    papers = []
    for i in range(n):
        abstract = random.choice(ABSTRACTS)
        embedding = model.encode(abstract, normalize_embeddings=True).tolist()
        papers.append({
            "id": i+1,
            "title": f"Paper {i+1}: {abstract[:40]}...",
            "abstract": abstract,
            "year": random.randint(2018, 2025),
            "category": random.choice(CATEGORIES),
            "citations": random.randint(0, 500),
            "embedding": embedding
        })
    with open(output_file, 'w') as f:
        json.dump(papers, f)

generate(1000, 'research_papers_1k.json')
generate(10000, 'research_papers_10k.json')
```

**Critical:** Pre-compute embeddings. Never compute embeddings during benchmarks — this adds ~50ms per query and masks actual query engine performance.

### 11.2 Benchmark Design Principles

1. **Warm up the buffer pool** before timing: run 100 queries before starting the timer.
2. **Run each benchmark 3 times** and report median.
3. **Fix the random seed** for all random data access patterns.
4. **Separate cold start from steady state**: first query includes embedding service startup.
5. **Use `b.ResetTimer()`** after setup to exclude initialization from timing.

### 11.3 Hybrid Retrieval Recall Evaluation

```go
// For each test query, compute ground truth by exhaustive search:
// groundTruth = TopK(all_rows sorted by cosine_similarity AND satisfying filter)

// Then run the hybrid engine and compute:
// recall@K = |hybridResults ∩ groundTruth| / K

func EvaluateRecall(db *HybridDB, queries []TestQuery, K int) float64 {
    totalRecall := 0.0
    for _, q := range queries {
        groundTruth := exhaustiveHybridSearch(db, q, K)
        hybridResults := db.HybridQuery(q, K)
        overlap := intersection(groundTruth, hybridResults)
        totalRecall += float64(len(overlap)) / float64(K)
    }
    return totalRecall / float64(len(queries))
}
```

Target: recall@5 ≥ 0.80 for Filter-First strategy, ≥ 0.95 for Vector-First (when not constrained by filter).

---

## 12. Risk Analysis

### 12.1 Risk Register

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **B+ tree split bug** | Critical | High | Invariant checker + property tests + visual debugger. Two-week allocation. |
| **WAL invariant violation** | Critical | Medium | Enforce at buffer pool level; crash tests for all code paths. |
| **Scope explosion: SQL parser** | High | High | Grammar subset locked in Phase 0. No new features after Phase 4. |
| **Embedding service cold start** | Medium | High | Pre-compute all demo embeddings. Keep Python process alive. |
| **Buffer pool exhaustion** | Medium | Medium | Size pool generously for expected workload. Add explicit error message. |
| **UI time sink** | Medium | High | UI is last (Phase 8). Hard cap: one week. Simplify if behind schedule. |
| **Premature ANN implementation** | High | Medium | HNSW is explicitly Phase 2 work. Adding it in Phase 6 will delay hybrid engine. |
| **Architecture drift** | High | Medium | Review interfaces after Phase 2 and Phase 5. Freeze interfaces before Phase 7. |
| **WAL file growth** | Low | High | Checkpoint after every 1000 operations. WAL segments before checkpoint are safe to truncate. |
| **Recovery logic incorrect** | Critical | Medium | Property tests + crash simulation at all points. Recovery must be idempotent. |

### 12.2 Scope Control Triggers

If any of these situations occur, immediately stop the current phase and resolve before continuing:

- Any phase is running 3+ days behind schedule → Cut one feature (document it as deferred)
- Test suite coverage drops below 70% → Stop feature work, write tests
- A critical bug is found in a previous phase → Fix it before proceeding
- The UI is taking more than 3 days → Simplify to a static JSON display + terminal output

---

## 13. Technical Debt Strategy

### 13.1 Acceptable Shortcuts in v1

| Shortcut | Why Acceptable |
|----------|---------------|
| Single-threaded execution engine | Correctness without race conditions; parallelism is v2 |
| Brute-force cosine similarity | Correct and fast enough for 10k rows; HNSW is v2 |
| Rule-based planner (no statistics) | Simple selectivity estimates work; cost-based optimization is v2 |
| In-memory sort (Sort operator) | Insufficient for very large results, but fine for LIMIT 10-50 queries |
| No connection pooling | Single-user educational system; not a production concern |
| Simple free page list | No fragmentation optimization; acceptable for educational datasets |

### 13.2 Dangerous Shortcuts to Never Take

| Shortcut | Why Dangerous |
|----------|--------------|
| Skip WAL for "simple" operations | WAL invariant applies to ALL writes or crash recovery is unreliable |
| Ignore B+ tree invariant checks in tests | Silent corruption may not manifest until demo day |
| Use global variables for buffer pool or WAL | Makes independent testing impossible |
| Skip error handling ("this can't fail") | All I/O operations can fail; all disk operations can return partial writes |
| Use `interface{}` / `any` in core storage types | Type safety is critical for tuple deserialization |

### 13.3 What Must Remain Extensible

| Component | Extension Point | v2 Plan |
|-----------|----------------|---------|
| Buffer pool | Replacement policy interface | Swap LRU for CLOCK or LFU |
| Vector index | `VectorIndex` interface | Swap brute-force for HNSW |
| Rank fusion | `FusionAlgorithm` interface | Add learned reranking |
| Query planner | Rule list (ordered slice) | Add cost-based rules |
| Embedding client | `EmbeddingClient` interface | Add OpenAI API support |

---

## 14. Final Demo Strategy

### 14.1 Demo Flow (15 minutes)

```
Minutes 0-2: SETUP
  - "HybridDB is a unified relational + vector database engine built from scratch."
  - Show architecture diagram briefly.
  - Pre-loaded dataset: 10,000 research papers with embeddings.

Minutes 2-4: STRUCTURED QUERY DEMO
  Query: SELECT title, year FROM research_papers
         WHERE category = 'database' AND year > 2022
         ORDER BY year DESC LIMIT 10;
  Show: execution plan tree (IndexScan on category, Filter on year)
  Point out: buffer pool hit rate climbing, pages read count

Minutes 4-6: SEMANTIC QUERY DEMO
  Query: SELECT title, similarity_score FROM research_papers
         WHERE embedding SIMILAR TO 'graph neural networks' LIMIT 5;
  Show: VectorScan operator in execution tree
  Point out: embedding latency, similarity scores in result
  Explain: "This is brute-force cosine similarity over 10,000 384-dimensional vectors"

Minutes 6-9: HYBRID QUERY DEMO (THE CENTERPIECE)
  Query: SELECT title, year, similarity_score FROM research_papers
         WHERE year > 2024 AND category = 'database'
         AND embedding SIMILAR TO 'vector indexing optimization'
         LIMIT 5;
  Show: planner selects FILTER_FIRST (selectivity 2%)
  Show: execution tree — IndexScan → Filter → VectorScore → Limit
  Point out: "240 candidates after filter, cosine similarity computed on 240 not 10,000"
  Point out: 5 final results ranked by semantic similarity

Minutes 9-11: B+ TREE VISUALIZATION
  Insert 10 records with IDs 1..10 → show tree building
  Insert ID 200 → trigger leaf split → animate split
  "This is exactly what PostgreSQL does internally on every index insert"

Minutes 11-13: CRASH RECOVERY DEMO
  Run: INSERT 50 rows → commit
  Run: INSERT 30 more rows → crash before commit (Ctrl+C mid-insert)
  Restart server → show WAL replay log in UI
  Run: SELECT COUNT(*) → "50 rows. The 30 uncommitted rows were rolled back."

Minutes 13-15: Q&A SETUP
  Show: buffer pool heatmap (live updates during repeated queries)
  Show: WAL log viewer (records streaming in)
  Show: EXPLAIN plan with cost annotations
  "I can take questions on any component."
```

### 14.2 Demo Dataset Preparation

```bash
# Pre-compute embeddings offline (DO THIS BEFORE DEMO DAY):
python datasets/generate.py --n 10000 --output datasets/research_papers_10k.json

# Load into HybridDB:
go run ./datasets/load.go --db demo.hdb --input datasets/research_papers_10k.json

# Verify:
hybriddb-cli exec --db demo.hdb "SELECT COUNT(*) FROM research_papers"
→ 10000
```

### 14.3 Maximizing Instructor Impact

The most impactful moments for an instructor evaluating systems understanding are:
1. Watching the B+ tree split animation and being able to explain every step
2. Watching crash recovery and explaining the redo/undo phases
3. Explaining why FILTER_FIRST was chosen over VECTOR_FIRST for the demo query
4. Pointing to the cosine similarity computation and explaining why brute force is O(N*d)

Prepare to explain each of these without the UI — as if in a whiteboard interview.

---

## 15. Viva / Interview Preparation

### 15.1 Expected Professor Questions

**Storage Layer:**
- "Walk me through exactly what happens when I call INSERT — from the SQL string to bytes on disk."
- "What is the WAL invariant and what happens if you violate it?"
- "How does the slotted page handle variable-length tuples without fragmenting?"
- "What is the purpose of the pageLSN field in the page header?"

**B+ Tree:**
- "Walk me through a leaf split that propagates to the root."
- "Why do we copy up the median key in a leaf split but push up in an internal split?"
- "How does range scan use the leaf chain? What is the complexity?"
- "What happens to a leaf node when the last key is deleted and it underflows?"

**Recovery:**
- "What is the difference between REDO and UNDO phases in recovery?"
- "Why must recovery be idempotent?"
- "What is a checkpoint and why does it exist?"
- "Can you have a committed transaction whose effects are not on disk? Explain."

**Hybrid Execution:**
- "When does VECTOR_FIRST outperform FILTER_FIRST and vice versa?"
- "What is Reciprocal Rank Fusion and why is it preferred over simple score addition?"
- "What is the computational complexity of your hybrid query?"
- "How would you extend the system to support HNSW? What would change?"

### 15.2 Systems Interview Questions

| Question | Key Concepts to Cover |
|----------|----------------------|
| "Why is LRU not optimal for sequential scan workloads?" | Sequential flooding; CLOCK-Pro handles it better |
| "How does write-ahead logging guarantee durability?" | WAL invariant; fsync before acknowledging commit |
| "What is the fanout of your B+ tree and why does it matter?" | High fanout → low height → fewer disk reads per lookup |
| "Why did you choose pull-based (Volcano) over push-based execution?" | Simplicity, memory efficiency; push is better for parallelism |
| "What is the recall-latency tradeoff in vector search?" | Brute force = perfect recall, high latency; ANN = approximate recall, lower latency |
| "How does MVCC differ from your simplified WAL approach?" | Version chains vs before/after images; snapshot isolation vs redo/undo |

### 15.3 Tradeoff Questions

| Tradeoff | Both Sides |
|----------|-----------|
| **Brute-force vs HNSW** | Brute force: O(N*d), exact, simple. HNSW: O(log N * d), approximate, complex to implement |
| **Filter-first vs Vector-first** | FF: better when filter is selective; VF: better when filter is loose |
| **LRU vs CLOCK** | LRU: optimal for skewed access; CLOCK: O(1) per access, simpler |
| **Slotted pages vs heap files** | Slotted: variable-length tuples, fragmentation; heap: fixed-length, simpler |
| **Rule-based vs cost-based planner** | Rule-based: deterministic, debuggable; cost-based: better plans, much more complex |
| **WAL vs shadow paging** | WAL: random writes, sequential log; shadow paging: copy-on-write, no recovery log needed |

### 15.4 Failure Mode Questions

| "What happens if..." | Answer Direction |
|---------------------|-----------------|
| "...the WAL file is corrupted at byte 5000?" | Record checksum fails; recovery stops at that record; transactions after that LSN are lost (acceptable for educational system) |
| "...the buffer pool is full and all frames are pinned?" | Return error immediately; no deadlock possible in single-threaded model |
| "...a B+ tree split crashes halfway through?" | Parent may have a pointer to a new node that is not yet linked; REDO phase replays the split, UNDO if not committed |
| "...the embedding service is down during a hybrid query?" | EmbeddingClient returns error; query fails with descriptive message; no silent degradation |
| "...two inserts happen to the same B+ tree node concurrently?" | v1 is single-threaded; no concurrency issue. In v2: latch coupling (crabbing) is the standard approach |

### 15.5 Concepts to Master (No Notes)

Before the viva, be able to explain from memory and whiteboard:

- [ ] Slotted page binary layout (exact byte offsets)
- [ ] WAL record fields and their purpose
- [ ] B+ tree split algorithm step by step
- [ ] REDO/UNDO phases of recovery with a concrete example
- [ ] LRU eviction: when it works, when it fails
- [ ] Cosine similarity formula and its geometric interpretation
- [ ] VF vs FF strategy selection rule and its rationale
- [ ] RRF formula and why k=60
- [ ] The Volcano iterator model: Open/Next/Close contract

---

*HybridDB Build Execution Plan — Version 1.0*
*A semester-long playbook for building a serious educational hybrid query engine*
*One student. One semester. One database.*