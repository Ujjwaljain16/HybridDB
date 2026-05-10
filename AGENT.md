# AGENT.md — HybridDB Autonomous Engineering Specification

> Definitive execution specification for AI coding agents working on HybridDB.
> This document defines architecture constraints, implementation philosophy,
> development sequencing, quality gates, testing rules, debugging standards,
> and execution protocols for building HybridDB correctly.

-------------------------------------------------------------------------------
SECTION 1 — PROJECT IDENTITY
-------------------------------------------------------------------------------

Project Name:
HybridDB

Project Type:
Educational Hybrid Relational + Vector Database Engine

Core Philosophy:
Build a deeply educational, architecturally rigorous, modern database engine
that combines:
- relational query execution
- vector similarity retrieval
- hybrid semantic + structured querying
- explainability tooling
- observability tooling

inside a single cohesive execution engine.

This project is NOT:
- a startup MVP
- a distributed hyperscaler database
- a production PostgreSQL replacement
- a cloud-native microservices system
- a vector database competitor

This project IS:
- a systems engineering capstone
- a database internals learning engine
- an educational hybrid query system
- a modern query execution research-inspired project

-------------------------------------------------------------------------------
SECTION 2 — ABSOLUTE PROJECT RULES
-------------------------------------------------------------------------------

The AI agent MUST obey ALL rules in this document.

These are NON-NEGOTIABLE.

RULE 1:
Correctness ALWAYS has priority over optimization.

RULE 2:
Never implement future scalability abstractions prematurely.

RULE 3:
Never introduce distributed systems concepts.

RULE 4:
Never introduce Kubernetes, Kafka, Raft, Paxos, sharding,
or replication.

RULE 5:
Never use external embedded databases:
- SQLite
- RocksDB
- LevelDB
- BoltDB
- LMDB

Storage must be implemented manually.

RULE 6:
Never skip tests.

RULE 7:
Every subsystem must be independently testable.

RULE 8:
Every phase must pass ALL tests before continuing.

RULE 9:
Never implement ANN indexes (HNSW/IVF/PQ) in v1.

RULE 10:
Vector search MUST initially be brute-force cosine similarity.

RULE 11:
Never optimize before instrumentation exists.

RULE 12:
Observability must be added alongside implementation,
not afterward.

RULE 13:
Never use ORM-style abstractions.

RULE 14:
All storage must use explicit binary layouts.

RULE 15:
All serialization formats must be deterministic.

RULE 16:
Never introduce concurrency into core execution engine v1.

RULE 17:
Core query execution is SINGLE-THREADED in v1.

RULE 18:
Never silently recover from corruption.

RULE 19:
Fail loudly and explicitly on invariant violations.

RULE 20:
Every major subsystem MUST define:
- invariants
- failure modes
- tests
- metrics

-------------------------------------------------------------------------------
SECTION 3 — MANDATORY TECH STACK
-------------------------------------------------------------------------------

CORE ENGINE:
- Go >= 1.22

FRONTEND:
- React
- TypeScript
- Tailwind
- React Flow
- Recharts

EMBEDDING SERVICE:
- Python 3.11+
- FastAPI
- sentence-transformers
- all-MiniLM-L6-v2

COMMUNICATION:
- REST APIs
- WebSockets

STORAGE:
- custom binary pages
- custom WAL

TESTING:
- Go standard testing
- property-based testing
- fuzz testing

PROHIBITED:
- Java Spring
- NestJS
- ORM libraries
- GraphQL
- gRPC
- Kubernetes
- Docker Compose orchestration
- distributed infra

-------------------------------------------------------------------------------
SECTION 4 — ARCHITECTURAL PHILOSOPHY
-------------------------------------------------------------------------------

HybridDB is built as a layered systems architecture.

Dependency direction is STRICTLY ENFORCED.

Allowed dependency flow:

Pager
  ↓
Slotted Pages
  ↓
Tuple Serialization
  ↓
Buffer Pool
  ↓
WAL
  ↓
B+ Tree
  ↓
Catalog
  ↓
Parser
  ↓
Planner
  ↓
Execution Engine
  ↓
Vector Engine
  ↓
Hybrid Query Engine
  ↓
Metrics / Trace
  ↓
HTTP Server
  ↓
React UI

NO REVERSE DEPENDENCIES ARE ALLOWED.

Examples of FORBIDDEN architecture:
- parser importing storage internals
- WAL importing query engine
- UI importing Go internals directly
- vector engine modifying storage engine directly
- execution engine bypassing buffer pool

-------------------------------------------------------------------------------
SECTION 5 — ENGINEERING PRINCIPLES
-------------------------------------------------------------------------------

PRINCIPLE 1 — Correctness Before Performance

Naive correct implementation > optimized incorrect implementation.

Examples:
- brute-force cosine similarity before ANN
- linear scans before cost-based optimization
- simple parser before grammar generators

-------------------------------------------------------------------------------

PRINCIPLE 2 — Observability Before Optimization

Before optimizing:
- add metrics
- add tracing
- add timing
- add visualization hooks

No optimization is allowed without measurable evidence.

-------------------------------------------------------------------------------

PRINCIPLE 3 — Educational Clarity Over Cleverness

Code must prioritize:
- readability
- explicitness
- traceability
- debuggability

Avoid:
- overly abstract generic frameworks
- magic metaprogramming
- hidden side effects
- clever compact algorithms

-------------------------------------------------------------------------------

PRINCIPLE 4 — Deterministic Behavior

All behavior must be deterministic.

Forbidden:
- hidden randomness
- time-dependent logic
- nondeterministic iteration order
- race conditions

Tests must be reproducible.

-------------------------------------------------------------------------------

PRINCIPLE 5 — Fail Loudly

Invariant violations must:
- return explicit errors
- panic in debug mode
- emit logs
- emit metrics

Silent corruption is unacceptable.

-------------------------------------------------------------------------------

PRINCIPLE 6 — Infrastructure First

Build:
- storage
- durability
- indexing
BEFORE:
- query optimization
- UI polish
- vector retrieval

-------------------------------------------------------------------------------
SECTION 6 — REPOSITORY STRUCTURE
-------------------------------------------------------------------------------

MANDATORY repository layout:

hybriddb/
│
├── cmd/
├── internal/
├── pkg/
├── web/
├── embedding-service/
├── benchmarks/
├── test/
├── docs/
├── diagrams/
├── datasets/
├── scripts/
└── README.md

-------------------------------------------------------------------------------
SECTION 7 — CORE SUBSYSTEMS
-------------------------------------------------------------------------------

===============================================================================
7.1 — PAGER
===============================================================================

PURPOSE:
Abstract disk page I/O.

RESPONSIBILITIES:
- page allocation
- page reads
- page writes
- free page tracking
- file layout management

REQUIREMENTS:
- fixed-size pages
- deterministic page IDs
- direct offset addressing
- binary-safe operations

FORBIDDEN:
- variable page sizes
- object serialization libraries
- mmap in v1

MANDATORY INVARIANTS:
- page size always constant
- pageID → offset mapping deterministic
- partial writes detected
- invalid page IDs rejected

MANDATORY TESTS:
- persistence across restart
- page allocation monotonicity
- free page reuse
- corruption detection

===============================================================================
7.2 — SLOTTED PAGES
===============================================================================

PURPOSE:
Store variable-length tuples inside fixed-size pages.

MANDATORY FEATURES:
- slot directory
- free-space tracking
- tuple deletion markers
- fragmentation handling
- page compaction

MANDATORY PAGE LAYOUT:

┌────────────────────┐
│ Header             │
├────────────────────┤
│ Slot Directory     │
│ grows downward ↓   │
├────────────────────┤
│ Free Space         │
├────────────────────┤
│ Tuple Data         │
│ grows upward ↑     │
└────────────────────┘

MANDATORY INVARIANTS:
- slot offsets valid
- tuples never overlap
- free space accurate
- deleted tuples not returned

MANDATORY TESTS:
- insert/read/delete
- compaction correctness
- fragmentation scenarios
- page full handling

===============================================================================
7.3 — BUFFER POOL
===============================================================================

PURPOSE:
Cache database pages in memory.

MANDATORY FEATURES:
- fixed-size frame pool
- LRU replacement
- dirty page tracking
- pin/unpin semantics

MANDATORY INVARIANTS:
- pinned pages never evicted
- dirty pages flushed before eviction
- WAL flushed before dirty page flush
- frame/page mappings consistent

MANDATORY METRICS:
- cache hit rate
- evictions
- dirty flushes
- pinned frames

MANDATORY TESTS:
- LRU ordering
- dirty eviction
- pinned frame protection
- cache hit accounting

===============================================================================
7.4 — WAL
===============================================================================

PURPOSE:
Durability and crash recovery.

MANDATORY FEATURES:
- append-only log
- checksums
- redo recovery
- undo recovery
- checkpoints

FORBIDDEN:
- skipping checksums
- silent WAL corruption handling

MANDATORY INVARIANTS:
- LSN monotonicity
- WAL-before-data rule
- checksum correctness

MANDATORY TESTS:
- crash simulation
- replay correctness
- partial WAL corruption
- idempotent recovery

===============================================================================
7.5 — B+ TREE
===============================================================================

PURPOSE:
Ordered indexing and range scans.

MANDATORY FEATURES:
- internal nodes
- leaf nodes
- leaf chaining
- node splits
- merges
- borrowing
- persistence

MANDATORY INVARIANTS:
- sorted keys
- valid separator keys
- balanced depth
- correct leaf chain
- no overflow
- no underflow

MANDATORY TESTS:
- sequential inserts
- reverse inserts
- randomized inserts/deletes
- range scans
- split propagation
- merge correctness

MANDATORY DEBUGGING:
- invariant checker
- ASCII tree dump
- JSON export for visualization

===============================================================================
7.6 — SQL PARSER
===============================================================================

PURPOSE:
Convert SQL strings into AST structures.

MANDATORY APPROACH:
- hand-written recursive descent parser

FORBIDDEN:
- ANTLR
- yacc
- parser generators

SUPPORTED SQL:
- CREATE TABLE
- CREATE INDEX
- INSERT
- SELECT
- WHERE
- ORDER BY
- LIMIT
- EXPLAIN

NOT SUPPORTED:
- JOINs
- GROUP BY
- HAVING
- subqueries
- transactions
- MVCC

MANDATORY TESTS:
- lexer correctness
- AST correctness
- invalid syntax handling
- deterministic parsing

===============================================================================
7.7 — EXECUTION ENGINE
===============================================================================

PURPOSE:
Execute physical query plans.

MANDATORY EXECUTION MODEL:
Volcano iterator model.

MANDATORY OPERATORS:
- SeqScan
- IndexScan
- Filter
- Projection
- Sort
- Limit
- VectorScan
- HybridScan

MANDATORY OPERATOR INTERFACE:

type Operator interface {
    Open(ctx *ExecContext) error
    Next() (*tuple.Tuple, error)
    Close() error
}

MANDATORY METRICS:
- operator latency
- rows processed
- pages read
- pages written

===============================================================================
7.8 — VECTOR ENGINE
===============================================================================

PURPOSE:
Semantic similarity retrieval.

MANDATORY v1 IMPLEMENTATION:
Brute-force cosine similarity.

FORBIDDEN IN v1:
- HNSW
- IVF
- PQ
- graph ANN indexes

MANDATORY FEATURES:
- embedding serialization
- vector storage
- cosine similarity
- top-K retrieval

MANDATORY TESTS:
- cosine correctness
- ranking stability
- deterministic ordering

===============================================================================
7.9 — HYBRID QUERY ENGINE
===============================================================================

PURPOSE:
Combine structured filtering with semantic retrieval.

MANDATORY STRATEGIES:
1. vector-first
2. filter-first

MANDATORY RANK FUSION:
- Reciprocal Rank Fusion (RRF)

MANDATORY EXPLAINABILITY:
- candidate counts
- filtering stages
- ranking stages
- execution timeline

MANDATORY TESTS:
- ranking correctness
- deterministic fusion
- edge-case filtering

-------------------------------------------------------------------------------
SECTION 8 — DEVELOPMENT PHASES
-------------------------------------------------------------------------------

===============================================================================
PHASE 0 — ARCHITECTURE & SETUP
===============================================================================

GOALS:
- repo setup
- interfaces
- tooling
- metrics scaffolding
- logging
- CI

EXIT CRITERIA:
- repository compiles
- lint passes
- test harness functional

FORBIDDEN:
- implementing actual storage logic

===============================================================================
PHASE 1 — STORAGE FOUNDATION
===============================================================================

IMPLEMENT:
- pager
- slotted pages
- tuple serialization

EXIT CRITERIA:
- tuples persist across restart
- storage tests pass
- page dump tooling works

===============================================================================
PHASE 2 — BUFFER POOL & WAL
===============================================================================

IMPLEMENT:
- LRU
- dirty tracking
- WAL
- recovery

EXIT CRITERIA:
- crash recovery works
- WAL replay deterministic
- dirty page flushing correct

===============================================================================
PHASE 3 — B+ TREE
===============================================================================

IMPLEMENT:
- insert
- search
- range scan
- delete
- splits
- merges

EXIT CRITERIA:
- invariant checker passes
- randomized stress tests pass
- persistence verified

===============================================================================
PHASE 4 — SQL + CATALOG
===============================================================================

IMPLEMENT:
- lexer
- parser
- AST
- schema catalog

EXIT CRITERIA:
- SQL parses deterministically
- schemas persist correctly

===============================================================================
PHASE 5 — QUERY EXECUTION
===============================================================================

IMPLEMENT:
- execution operators
- execution planner
- iterator model

EXIT CRITERIA:
- end-to-end relational queries work

===============================================================================
PHASE 6 — VECTOR ENGINE
===============================================================================

IMPLEMENT:
- embeddings
- cosine similarity
- vector scans

EXIT CRITERIA:
- semantic search works
- vector serialization correct

===============================================================================
PHASE 7 — HYBRID EXECUTION
===============================================================================

IMPLEMENT:
- vector-first planning
- filter-first planning
- rank fusion

EXIT CRITERIA:
- hybrid queries deterministic
- explain plans accurate

===============================================================================
PHASE 8 — OBSERVABILITY
===============================================================================

IMPLEMENT:
- metrics
- tracing
- dashboards
- visualizers

EXIT CRITERIA:
- execution plans visible
- B+ tree visualized
- WAL visible
- metrics stream live

===============================================================================
PHASE 9 — HARDENING
===============================================================================

IMPLEMENT:
- fuzzing
- benchmarking
- stress testing
- crash testing

EXIT CRITERIA:
- no invariant failures
- stable benchmark runs
- recovery reliable

-------------------------------------------------------------------------------
SECTION 9 — TESTING PHILOSOPHY
-------------------------------------------------------------------------------

MANDATORY TEST TYPES:
- unit tests
- integration tests
- property tests
- fuzz tests
- crash tests
- replay tests

EVERY STORAGE STRUCTURE MUST HAVE:
- round-trip tests
- corruption tests
- deterministic serialization tests

EVERY INDEX STRUCTURE MUST HAVE:
- invariant validation
- randomized stress testing

EVERY QUERY EXECUTION PATH MUST HAVE:
- deterministic expected outputs

-------------------------------------------------------------------------------
SECTION 10 — DEBUGGING STRATEGY
-------------------------------------------------------------------------------

MANDATORY DEBUGGING TOOLS:
- page dump utility
- WAL dump utility
- B+ tree ASCII renderer
- execution trace viewer
- query explain output

MANDATORY LOGGING:
- page reads/writes
- WAL appends
- node splits
- node merges
- operator execution
- vector scoring

MANDATORY FAILURE ANALYSIS:
Every major failure must emit:
- subsystem
- invariant violated
- relevant IDs
- page IDs
- LSNs
- operator context

-------------------------------------------------------------------------------
SECTION 11 — OBSERVABILITY REQUIREMENTS
-------------------------------------------------------------------------------

MANDATORY VISUALIZATIONS:
- query plan tree
- B+ tree structure
- buffer pool heatmap
- WAL viewer
- execution timeline
- vector candidate pipeline

MANDATORY METRICS:
- query latency
- operator timings
- cache hit rate
- WAL throughput
- vector candidate counts
- rows processed

-------------------------------------------------------------------------------
SECTION 12 — PERFORMANCE PHILOSOPHY
-------------------------------------------------------------------------------

Performance optimization priority:

1. correctness
2. observability
3. algorithmic clarity
4. benchmarkability
5. optimization

Premature optimization is FORBIDDEN.

Initial acceptable performance:
- brute-force vector search
- linear scans
- naive planner
- in-memory sorting

-------------------------------------------------------------------------------
SECTION 13 — CODE STYLE RULES
-------------------------------------------------------------------------------

MANDATORY:
- descriptive variable names
- explicit error handling
- comments explaining invariants
- deterministic serialization
- explicit interfaces

FORBIDDEN:
- hidden magic
- giant god objects
- implicit state mutation
- reflection-heavy logic
- excessive abstraction

-------------------------------------------------------------------------------
SECTION 14 — DOCUMENTATION REQUIREMENTS
-------------------------------------------------------------------------------

EVERY major module MUST contain:
- README.md
- architecture explanation
- invariants
- failure modes
- testing instructions

MANDATORY DOCUMENTS:
- architecture.md
- storage-format.md
- recovery.md
- hybrid-execution.md
- query-lifecycle.md

-------------------------------------------------------------------------------
SECTION 15 — AI AGENT EXECUTION PROTOCOL
-------------------------------------------------------------------------------

Before implementing ANY feature:
1. Identify subsystem dependencies.
2. Verify prerequisite phases complete.
3. Define invariants.
4. Define binary layouts (if storage-related).
5. Define interfaces.
6. Write tests FIRST.
7. Implement minimally.
8. Run tests.
9. Add metrics/tracing.
10. Refactor only after correctness.

-------------------------------------------------------------------------------

When implementing storage code:
ALWAYS:
- define exact byte layout
- specify endianness
- specify offsets
- specify alignment rules
- define corruption handling

-------------------------------------------------------------------------------

When implementing indexes:
ALWAYS:
- define invariants first
- implement invariant checker
- add randomized tests
- verify persistence
- verify recovery

-------------------------------------------------------------------------------

When implementing query operators:
ALWAYS:
- instrument operator timings
- emit execution trace events
- validate schema consistency

-------------------------------------------------------------------------------

When implementing vector retrieval:
ALWAYS:
- verify embedding dimensions
- validate float32 serialization
- ensure deterministic ranking

-------------------------------------------------------------------------------
SECTION 16 — FORBIDDEN FAILURE MODES
-------------------------------------------------------------------------------

The AI agent MUST NEVER:
- silently ignore errors
- swallow WAL corruption
- auto-repair pages silently
- bypass tests
- hardcode unstable assumptions
- mutate storage formats ad hoc
- change TypeTag values after Phase 0
- introduce hidden concurrency

-------------------------------------------------------------------------------
SECTION 17 — FINAL PROJECT VISION
-------------------------------------------------------------------------------

HybridDB should ultimately feel like:

“A serious educational hybrid relational + semantic query engine
with modern observability and explainability tooling.”

The finished project should demonstrate:
- database internals mastery
- systems engineering maturity
- query execution understanding
- storage engine understanding
- indexing knowledge
- hybrid retrieval architecture
- observability engineering
- implementation discipline

The project should impress:
- database professors
- systems engineers
- infrastructure interviewers
- AI retrieval engineers

WITHOUT:
- fake scale claims
- buzzword architecture
- distributed systems theater

-------------------------------------------------------------------------------
END OF AGENT.md
-------------------------------------------------------------------------------