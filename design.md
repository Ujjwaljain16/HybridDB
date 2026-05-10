# HybridDB — Product Requirements Document + Technical Requirements Document

> **Version 1.0 · Capstone Systems Engineering Specification**
> *A modern educational database engine exploring hybrid semantic + relational retrieval*

---

## Table of Contents

- [PART I — PRODUCT REQUIREMENTS DOCUMENT](#part-i--product-requirements-document)
  - [1. Executive Summary](#1-executive-summary)
  - [2. Problem Statement](#2-problem-statement)
  - [3. Product Vision](#3-product-vision)
  - [4. Goals](#4-goals)
  - [5. Non-Goals](#5-non-goals)
  - [6. User Personas](#6-user-personas)
  - [7. User Stories](#7-user-stories)
  - [8. Functional Requirements](#8-functional-requirements)
  - [9. Non-Functional Requirements](#9-non-functional-requirements)
  - [10. UI/UX Requirements](#10-uiux-requirements)
  - [11. Demo Requirements](#11-demo-requirements)
  - [12. Success Metrics](#12-success-metrics)
- [PART II — TECHNICAL REQUIREMENTS DOCUMENT](#part-ii--technical-requirements-document)
  - [13. System Architecture](#13-system-architecture)
  - [14. Core Engine Design](#14-core-engine-design)
  - [15. Indexing Subsystems](#15-indexing-subsystems)
  - [16. Query Engine](#16-query-engine)
  - [17. Hybrid Query Engine](#17-hybrid-query-engine)
  - [18. Vector Storage](#18-vector-storage)
  - [19. Observability & Educational Tooling](#19-observability--educational-tooling)
  - [20. Benchmarking Strategy](#20-benchmarking-strategy)
  - [21. Testing Strategy](#21-testing-strategy)
  - [22. Semester Roadmap](#22-semester-roadmap)
  - [23. Risk Analysis](#23-risk-analysis)
  - [24. Future Extensions](#24-future-extensions)

---

# PART I — PRODUCT REQUIREMENTS DOCUMENT

---

## 1. Executive Summary

### 1.1 Vision

HybridDB is a lightweight, self-contained educational database engine that unifies three historically separate concerns — relational storage, vector similarity retrieval, and hybrid semantic-plus-structured query execution — inside a single, architecturally transparent system.

The engine is not designed to compete with production databases. It is designed to be **understood**, **visualized**, and **reasoned about** by a student learning database internals. Every layer — from raw page layout on disk to the scoring function that fuses semantic and relational rankings — is exposed, instrumented, and explainable.

### 1.2 Positioning

| Axis | HybridDB | Traditional Mini-DB | pgvector / Pinecone |
|------|----------|---------------------|---------------------|
| **Scope** | Educational, unified | Educational, relational only | Production, vector-focused |
| **Explainability** | First-class feature | Absent | Absent |
| **Hybrid execution** | Native, unified planner | Not supported | Bolt-on via application layer |
| **Storage internals** | Fully exposed, visualized | Partially exposed | Opaque |
| **Audience** | Students, professors | Students | Engineers |

### 1.3 Uniqueness

What makes HybridDB academically interesting is **not** the ambition to outperform existing systems, but the ambition to **explain** them — and to do so for a class of query that no existing educational project addresses: the hybrid semantic-plus-structured retrieval query.

```sql
SELECT *
FROM research_papers
WHERE year > 2024
  AND category = 'database'
  AND embedding SIMILAR TO 'vector indexing optimization'
LIMIT 5;
```

No CMU database course mini-project handles this. No standard database textbook covers it. HybridDB fills that gap.

### 1.4 Educational Value

HybridDB teaches, in one cohesive project:
- How data is physically stored on disk (slotted pages, pager, buffer pool)
- How durability is achieved without full ACID (simplified WAL + redo recovery)
- How structured data is indexed (B+ tree: splits, merges, leaf chaining)
- How unstructured semantic meaning is quantified (embeddings, cosine similarity)
- How a query planner decides what to do (rule-based operator trees)
- How two fundamentally different retrieval paradigms are fused into one result set (rank fusion, reranking)
- How all of the above can be made visible and debuggable (explainability UI)

### 1.5 Modern Relevance

Every major database system — PostgreSQL (pgvector), Redis (FT.HYBRID), Weaviate, Qdrant — is actively working on hybrid retrieval. This is the frontier of database engineering in the AI era. HybridDB positions a student to understand and contribute to that frontier, not as a user of these systems, but as someone who has built equivalent internals from scratch.

---

## 2. Problem Statement

### 2.1 The Separation Problem

Modern applications routinely need to answer queries like:

> *"Find documents semantically similar to this concept, but only among records published after 2023 in the 'systems' category."*

Today, this query cannot be executed natively by either class of database system:

**Relational databases** (PostgreSQL, SQLite) have no notion of semantic similarity. They can filter by `year > 2023` and `category = 'systems'` with great efficiency, but they cannot understand that "vector indexing" and "ANN search" are semantically related.

**Vector databases** (Pinecone, Weaviate, Qdrant) can retrieve semantically similar documents efficiently, but their filter semantics are limited, their storage internals are opaque, and they are fundamentally designed around one retrieval modality.

The result is a **two-system architecture** — a vector database for semantic retrieval, a relational database for structured storage and filtering — stitched together at the application layer with fragile, inefficient glue code.

### 2.2 Why This Is Hard

The fundamental difficulty of hybrid retrieval is not implementation complexity — it is the **conceptual incommensurability** of two ranking paradigms:

- Structured queries produce **boolean results**: a row either satisfies `year > 2024` or it does not.
- Semantic queries produce **graded similarity scores**: a document has cosine similarity 0.87 to a query vector.

Combining these requires:
1. A unified **candidate generation** strategy that doesn't miss relevant results.
2. A **score normalization** approach that makes similarity scores and filter scores comparable.
3. A **rank fusion** algorithm that produces a final ranked output meaningful to the user.
4. A **query planning** layer that decides, for each query, whether to start with vectors (then filter) or start with filters (then rank by similarity).

None of these decisions are obvious, and each involves tradeoffs between recall, latency, and result quality.

### 2.3 Why This Is Academically Interesting

The problem sits at the intersection of four active research areas:
- **Database query optimization** (how to plan hybrid execution)
- **Information retrieval** (how to fuse rankings from heterogeneous sources)
- **Vector indexing** (how approximate nearest neighbor search works and fails)
- **Storage systems** (how to co-locate structured and vector data efficiently)

A capstone project that genuinely engages with all four of these areas — even in simplified form — is a rare and impressive contribution.

### 2.4 The Educational Gap

Traditional database internals courses teach storage engines and query processors over relational data. Modern AI engineering courses teach vector embeddings and similarity search at the API level. **No course bridges these two worlds at the internals level.** HybridDB is that bridge.

---

## 3. Product Vision

### 3.1 Long-Term Philosophy

HybridDB is built on a single philosophical commitment: **every component that is implemented must be fully observable**. A user should be able to look inside any stage of query execution — from the raw bytes on a disk page to the final ranked result set — and understand exactly what happened and why.

This philosophy drives every architectural decision:
- Metrics are collected at every operator boundary.
- The WAL is human-readable by design.
- The B+ tree can be rendered as an interactive diagram.
- The hybrid executor exposes which candidates were generated, which were filtered, and how scores were combined.

### 3.2 Extensibility Thinking

The v1 implementation prioritizes correctness and clarity over completeness. Every major subsystem is designed with a defined extension point:

- The **buffer pool** can swap out its LRU policy for CLOCK or LFU.
- The **vector engine** can replace brute-force cosine similarity with HNSW.
- The **query planner** can be upgraded from rule-based to cost-based.
- The **rank fusion** module is pluggable: RRF, weighted linear combination, or learned reranking are all drop-in replacements.

### 3.3 Future Roadmap Thinking

A successful v1 HybridDB creates a foundation for:
- **MVCC** and snapshot isolation (v2)
- **Temporal queries** with bitemporal indexing (v2)
- **ANN indexes** (HNSW, IVF) replacing brute-force vector search (v2)
- **Cost-based query optimization** with cardinality estimation (v3)
- **Streaming ingestion** with incremental indexing (v3)

---

## 4. Goals

### 4.1 Educational Goals

| ID | Goal |
|----|------|
| **EG-1** | Teach physical storage layout: slotted pages, tuple serialization, free space management |
| **EG-2** | Teach buffer management: page pinning, dirty tracking, LRU eviction |
| **EG-3** | Teach durability: WAL structure, crash recovery, redo-based replay |
| **EG-4** | Teach tree indexing: B+ tree splits, merges, leaf chaining, range scans |
| **EG-5** | Teach vector retrieval: embedding semantics, cosine similarity, ANN approximation tradeoffs |
| **EG-6** | Teach query planning: operator trees, execution strategies, rule-based selection |
| **EG-7** | Teach hybrid retrieval: candidate generation, score normalization, rank fusion |
| **EG-8** | Teach observability: how to instrument a system for learning and debugging |

### 4.2 Technical Goals

| ID | Goal |
|----|------|
| **TG-1** | Implement a correct, persistent, page-based storage engine |
| **TG-2** | Implement a correct B+ tree index with disk persistence |
| **TG-3** | Implement a simplified WAL with redo-based crash recovery |
| **TG-4** | Implement a SQL subset parser producing a typed AST |
| **TG-5** | Implement a pull-based execution engine with standard relational operators |
| **TG-6** | Implement vector column storage and brute-force cosine similarity search |
| **TG-7** | Implement a hybrid query executor with rank fusion |
| **TG-8** | Implement an explainability UI exposing all of the above |

### 4.3 Architectural Goals

| ID | Goal |
|----|------|
| **AG-1** | Maintain strict layer boundaries: storage → indexing → query → interface |
| **AG-2** | Define clean module interfaces so components are independently testable |
| **AG-3** | Design each module with a documented extension point for v2 improvements |
| **AG-4** | Avoid premature optimization: correctness before performance |

### 4.4 Demonstration Goals

| ID | Goal |
|----|------|
| **DG-1** | Demonstrate a live hybrid query executing correctly end-to-end |
| **DG-2** | Show an animated execution plan tree during query execution |
| **DG-3** | Visualize B+ tree structure before and after insertions |
| **DG-4** | Show WAL entries being written and replayed during crash recovery |
| **DG-5** | Display per-operator timing and page access statistics |

### 4.5 Research Relevance Goals

| ID | Goal |
|----|------|
| **RG-1** | Demonstrate understanding of the hybrid search literature (RRF, two-phase retrieval) |
| **RG-2** | Implement and compare at least two rank fusion strategies |
| **RG-3** | Demonstrate the vector-first vs filter-first planning tradeoff empirically |
| **RG-4** | Position the project within the context of pgvector, Qdrant, and Weaviate |

---

## 5. Non-Goals

The following are explicitly excluded from HybridDB v1, with justification.

| Excluded Feature | Reason |
|-----------------|--------|
| **Distributed consensus / Raft** | Adds weeks of implementation complexity with no educational return in database internals |
| **Multi-node replication** | Network partition handling, leader election, and log shipping are a separate research domain |
| **Full SQL compliance** | Full SQL grammar (window functions, CTEs, subqueries) would consume the entire project timeline. A clear, useful subset is more valuable. |
| **Full ACID semantics** | Serializability via 2PL or MVCC is a significant additional system. Simplified WAL + redo recovery is sufficient to teach durability. |
| **Production security** | Authentication, authorization, and TLS are operational concerns irrelevant to the educational goals. |
| **Parallel query execution** | Thread-safe execution with work-stealing schedulers is a separate domain. Single-threaded correctness first. |
| **HNSW / IVF in v1** | Brute-force cosine similarity is correct, explainable, and sufficient for the dataset sizes used. ANN indexing is documented as v2 work. |
| **Cost-based optimizer** | Statistics collection, cardinality estimation, and plan costing are a complete sub-project. Rule-based planning is sufficient for v1. |
| **Cloud deployment** | Kubernetes, container orchestration, and cloud-native storage are operational concerns outside the scope. |

Scope discipline is not a limitation — it is the reason the project can be completed well. Partial implementations of complex features produce neither working systems nor educational value.

---

## 6. User Personas

### Persona 1 — The Database Systems Student

**Name:** Arjun, 3rd year CS undergraduate

**Context:** Taking a database internals course. Has implemented a basic B+ tree. Has never seen a vector database. Wants to understand how modern AI-powered search actually works under the hood.

**Goals with HybridDB:**
- Run a hybrid query and watch it execute step by step
- Inspect the B+ tree visually to understand how splits work
- See the WAL entries that correspond to a sequence of inserts
- Understand why cosine similarity works for semantic matching

**Frustrations today:** Every resource either explains relational internals (ignoring vectors) or explains vector search at the API level (ignoring internals). Nothing bridges both.

---

### Persona 2 — The Systems Engineering Professor

**Name:** Dr. Meera, professor of database systems

**Context:** Teaches a graduate course on database internals. Wants a demo system that can illustrate hybrid retrieval concepts without requiring students to wade through PostgreSQL source code.

**Goals with HybridDB:**
- Use the explainability UI as a teaching aid in lectures
- Assign modifications to specific modules as homework
- Point to the codebase as a clean reference implementation

**Frustrations today:** Mini-database projects are either too simple (can't illustrate modern concepts) or too complex (students can't modify them).

---

### Persona 3 — The Infrastructure Recruiter / Technical Interviewer

**Name:** Priya, senior engineer at a data infrastructure company

**Context:** Evaluating a candidate's systems depth. Wants to see evidence of genuine understanding of storage systems, query engines, and modern retrieval.

**Goals with HybridDB:**
- Ask the candidate to explain the buffer pool replacement policy
- Ask why the hybrid executor might choose filter-first over vector-first
- Ask what happens during crash recovery
- Understand the rank fusion algorithm and its tradeoffs

**Value seen:** A candidate who built this understands something real about how databases work — not just how to call an API.

---

### Persona 4 — The AI Systems Engineer

**Name:** Rajan, building a RAG pipeline at a startup

**Context:** Frustrated by the two-system architecture (vector DB + relational DB). Wants to understand what a unified hybrid execution engine actually looks like internally.

**Goals with HybridDB:**
- Understand the query planning strategies for hybrid retrieval
- See a concrete implementation of rank fusion
- Evaluate whether the tradeoffs made in HybridDB are applicable to their production system

---

## 7. User Stories

### 7.1 Structured Querying

**US-1 — Basic Selection**
As a user, I want to execute `SELECT * FROM papers WHERE year > 2022` and get correct results, so that I can query structured data.

**US-2 — Indexed Lookup**
As a user, I want `SELECT * FROM papers WHERE category = 'database'` to use the B+ tree index on `category`, so that the query is faster than a full table scan.

**US-3 — Range Scan**
As a user, I want range predicates like `year BETWEEN 2020 AND 2024` to use the B+ tree's leaf-chained range scan, so that I understand how range indexing works.

**US-4 — Projection and Limit**
As a user, I want `SELECT title, year FROM papers LIMIT 10` to execute a projection and limit, so that I can retrieve partial columns efficiently.

### 7.2 Semantic Retrieval

**US-5 — Semantic Search**
As a user, I want `SELECT * FROM papers WHERE embedding SIMILAR TO 'distributed hash tables'` to return the 10 most semantically similar papers, so that I can retrieve documents by meaning rather than keywords.

**US-6 — Top-K Control**
As a user, I want to control the number of semantic candidates retrieved via `LIMIT`, so that I can tune the recall-latency tradeoff.

### 7.3 Hybrid Queries

**US-7 — Full Hybrid Query**
As a user, I want to execute:
```sql
SELECT title, year, similarity_score
FROM research_papers
WHERE year > 2024
  AND category = 'database'
  AND embedding SIMILAR TO 'vector indexing optimization'
LIMIT 5;
```
and get correctly ranked results that satisfy both the filter conditions and the semantic similarity criterion.

**US-8 — Plan Switching**
As a user, I want the query planner to automatically choose between vector-first and filter-first strategies based on the selectivity of the filter predicates, so that hybrid queries execute efficiently.

### 7.4 Explainability

**US-9 — Explain Plan**
As a user, I want to prefix any query with `EXPLAIN` and see the full operator tree with estimated costs, so that I can understand the execution plan before running it.

**US-10 — Live Execution Trace**
As a user, I want the UI to animate the execution plan during query execution, highlighting each operator as it runs, so that I can watch the query execute in real time.

**US-11 — Storage Inspection**
As a user, I want to visualize the current state of the buffer pool — which pages are pinned, which are dirty, which were recently accessed — so that I understand how caching works.

### 7.5 Indexing and Debugging

**US-12 — B+ Tree Visualization**
As a user, I want to visualize the B+ tree after each insertion, so that I can watch splits happen and understand the tree structure.

**US-13 — WAL Inspection**
As a user, I want to see WAL records as they are written, so that I understand what gets logged and why.

**US-14 — Recovery Demo**
As a developer, I want to simulate a crash mid-transaction and then watch the database recover by replaying the WAL, so that I can demonstrate and understand crash recovery.

### 7.6 Benchmarking

**US-15 — Performance Comparison**
As a researcher, I want to run a benchmark comparing vector-only, filter-only, and hybrid query execution times over a dataset of 10,000 records, so that I can empirically observe the performance characteristics of each strategy.

---

## 8. Functional Requirements

### 8.1 Storage Engine

**FR-S1 — Fixed-Size Page Storage**

The storage engine shall organize data in fixed-size pages. The default page size is **4096 bytes**, configurable at database creation time. Each page shall be identified by a monotonically increasing `PageID` (uint32).

```
Page File Layout:
+------------------+
| DB Header Page 0 |  (schema, metadata)
+------------------+
| Page 1           |
+------------------+
| Page 2           |
+------------------+
| ...              |
+------------------+
```

**FR-S2 — Slotted Page Architecture**

Each data page shall use a slotted page layout to support variable-length tuples:

```
Slotted Page Layout (4096 bytes):
+--------------------------------------+  offset 0
| Page Header (32 bytes)               |
|  - pageID: uint32                    |
|  - pageType: uint8                   |
|  - numSlots: uint16                  |
|  - freeSpaceOffset: uint16           |
|  - lsn: uint64 (last WAL LSN)        |
+--------------------------------------+  offset 32
| Slot Directory                       |
|  [slotID: uint16, offset: uint16,    |
|   length: uint16, flags: uint8]      |
|  ... (grows downward)                |
+--------------------------------------+
|           Free Space                 |
|         (grows toward each other)    |
+--------------------------------------+
| Tuple Data                           |
|  ... (grows upward from page end)    |
+--------------------------------------+  offset 4096
```

**FR-S3 — Tuple Serialization**

The engine shall serialize tuples with a header followed by columnar data:

```
Tuple Wire Format:
+--------------------+
| Tuple Header       |
|  - numCols: uint8  |
|  - nullBitmap: []  |
+--------------------+
| Col 0: TypeTag(1B) + Data |
| Col 1: TypeTag(1B) + Data |
| ...                       |
+--------------------+

Type Tags:
  0x01 = INT32    (4 bytes)
  0x02 = INT64    (8 bytes)
  0x03 = FLOAT32  (4 bytes)
  0x04 = VARCHAR  (2-byte length prefix + bytes)
  0x05 = VECTOR   (2-byte dim count + dim*4 bytes float32)
  0x06 = NULL     (0 bytes, represented in nullBitmap)
```

**FR-S4 — Free Space Management**

The pager shall maintain a free space list tracking pages with available space for insert operations. Compaction of fragmented pages shall be triggered when free space drops below a configurable threshold (default: 20% of page size).

**FR-S5 — Catalog Pages**

Page 0 shall be reserved as the database catalog page, storing:
- Table definitions (name, column names, types, constraints)
- Index definitions (table, column, index type, root page ID)
- Database-level metadata (page size, version, creation timestamp)

### 8.2 Buffer Pool

**FR-B1 — Page Cache**

The buffer pool shall maintain an in-memory cache of pages. The default pool size is **256 frames**. Each frame holds exactly one page.

**FR-B2 — Pin / Unpin Semantics**

- `PinPage(pageID)` → increments the pin count; pinned pages cannot be evicted.
- `UnpinPage(pageID, isDirty)` → decrements pin count; marks dirty if modified.
- The executor must pin pages before accessing and unpin after use.

**FR-B3 — LRU Replacement**

When the pool is full and a new page must be loaded, the buffer pool shall evict the least-recently-unpinned, non-pinned page. Dirty pages must be written to disk before eviction (write-before-evict protocol).

**FR-B4 — Metrics Exposure**

The buffer pool shall expose: `hit_rate`, `eviction_count`, `dirty_page_count`, `pinned_page_count` as observable metrics.

### 8.3 WAL and Recovery

**FR-W1 — Write-Ahead Logging**

Before any page modification is written to disk, the corresponding WAL record must be flushed to the WAL file. This is the Write-Ahead Logging (WAL) invariant and must not be violated.

**FR-W2 — WAL Record Format**

```
WAL Record Layout:
+--------------------+
| LSN: uint64        |  (Log Sequence Number, monotonically increasing)
| TxnID: uint32      |
| RecordType: uint8  |  (BEGIN, INSERT, UPDATE, DELETE, COMMIT, ABORT)
| TableID: uint32    |
| PageID: uint32     |
| SlotID: uint16     |
| BeforeImage: []byte|  (for UPDATE/DELETE; empty for INSERT)
| AfterImage: []byte |  (for INSERT/UPDATE; empty for DELETE)
| Checksum: uint32   |
+--------------------+
```

**FR-W3 — Crash Recovery**

On startup, the system shall:
1. Scan the WAL from the last checkpoint LSN to the end of the log.
2. **Redo** all operations from committed transactions whose effects may not have reached disk.
3. **Undo** incomplete (non-committed) transactions by applying before-images.
4. Re-establish the buffer pool in a clean state.

**FR-W4 — Checkpointing**

The system shall support a `CHECKPOINT` command that:
1. Flushes all dirty pages to disk.
2. Writes a checkpoint record to the WAL.
3. Records the checkpoint LSN so future recovery can start from there.

### 8.4 Query Parsing

**FR-P1 — Supported SQL Statements**

The parser shall produce a typed AST for the following statements:

| Statement | Example |
|-----------|---------|
| `CREATE TABLE` | `CREATE TABLE papers (id INT, title VARCHAR, year INT, embedding VECTOR(384))` |
| `CREATE INDEX` | `CREATE INDEX ON papers (category)` |
| `INSERT` | `INSERT INTO papers VALUES (1, 'HybridDB', 2024, [0.1, 0.2, ...])` |
| `SELECT` | `SELECT title, year FROM papers WHERE year > 2023 LIMIT 10` |
| `SELECT (hybrid)` | `SELECT * FROM papers WHERE year > 2023 AND embedding SIMILAR TO 'vector search'` |
| `EXPLAIN SELECT` | Produces plan tree without executing |
| `DELETE` | `DELETE FROM papers WHERE id = 5` |
| `UPDATE` | `UPDATE papers SET year = 2025 WHERE id = 3` |

**FR-P2 — AST Node Types**

```
ASTNode Types:
  SelectStmt { columns, from, where, orderBy, limit }
  WhereClause { predicates: [Predicate] }
  Predicate:
    CompPredicate  { col, op, val }        -- year > 2024
    RangePredicate { col, lo, hi }         -- year BETWEEN 2020 AND 2024
    SimilarToPredicate { col, queryText }  -- embedding SIMILAR TO '...'
    AndPredicate   { left, right }
    OrPredicate    { left, right }
```

**FR-P3 — Grammar Simplifications**

To remain implementable, the parser explicitly does not support:
- Subqueries
- CTEs (`WITH` clauses)
- `GROUP BY` / `HAVING`
- Window functions
- Multi-table joins in v1 (planned for v2)

### 8.5 Query Execution

**FR-E1 — Pull-Based Iterator Model**

All operators shall implement the Volcano/iterator interface:

```go
type Operator interface {
    Open(ctx *ExecContext) error
    Next() (*Tuple, error)   // returns nil when exhausted
    Close() error
    Schema() Schema
}
```

**FR-E2 — Standard Relational Operators**

| Operator | Purpose | Input | Output |
|----------|---------|-------|--------|
| `SeqScan` | Full table scan | Table name | All tuples |
| `IndexScan` | B+ tree point/range lookup | Index, predicate | Matching tuples |
| `Filter` | Predicate evaluation | Operator, predicate | Tuples satisfying predicate |
| `Projection` | Column selection | Operator, column list | Projected tuples |
| `Sort` | In-memory sort | Operator, sort keys | Sorted tuples |
| `Limit` | Result truncation | Operator, count | First N tuples |
| `VectorScan` | Brute-force cosine search | Table, query embedding, K | Top-K similar tuples with scores |
| `HybridScan` | Combined retrieval | Structured plan, vector plan | Fused, ranked tuples |

**FR-E3 — Execution Context**

Each query execution receives an `ExecContext` carrying:
- Buffer pool reference
- WAL reference
- Transaction ID
- Metrics collector (for observability)
- Execution trace recorder (for UI)

### 8.6 B+ Tree Index

**FR-I1 — Supported Operations**

The B+ tree shall support: `Insert(key, rid)`, `Search(key) → rid`, `RangeScan(lo, hi) → []rid`, `Delete(key)`.

**FR-I2 — Node Structure**

```
Internal Node (4096 bytes):
+-------------------+
| NodeHeader        |
|  - isLeaf: bool   |
|  - numKeys: uint16|
|  - parentID: uint32|
+-------------------+
| Keys[0..N-1]      |
| Pointers[0..N]    |   (N+1 child page IDs)
+-------------------+

Leaf Node (4096 bytes):
+-------------------+
| NodeHeader        |
|  - isLeaf: bool   |
|  - numKeys: uint16|
|  - nextLeafID: uint32 | (for chaining)
|  - prevLeafID: uint32 |
+-------------------+
| Keys[0..N-1]      |
| RecordIDs[0..N-1] |   (pageID, slotID) pairs
+-------------------+
```

**FR-I3 — Split Logic**

When a node overflows (numKeys == maxKeys):
1. Allocate a new node.
2. Distribute keys: lower half stays, upper half goes to new node.
3. For internal nodes: push median key up to parent. For leaf nodes: copy median key up (leaf keys are not removed).
4. If root splits, allocate new root.

**FR-I4 — Merge Logic**

When a node underflows (numKeys < minKeys = ⌈maxKeys/2⌉ - 1):
1. First attempt to borrow a key from a sibling via rotation.
2. If borrowing is not possible, merge with a sibling and pull down the separator key from the parent.
3. Recursively handle parent underflow.

**FR-I5 — Persistence**

All B+ tree nodes are stored as database pages managed by the buffer pool. Node modifications are WAL-logged before the buffer pool writes them to disk.

### 8.7 Vector Search

**FR-V1 — Vector Column Type**

The SQL layer shall support `VECTOR(d)` as a column type, where `d` is the dimensionality (typically 384 for MiniLM, 768 for BERT-base, 1536 for OpenAI text-embedding-3-small).

**FR-V2 — SIMILAR TO Syntax**

```sql
-- Text-based query (system embeds the query string at runtime)
embedding SIMILAR TO 'vector indexing optimization'

-- Direct vector query (pre-computed embedding)
embedding SIMILAR TO [0.1, 0.2, ..., 0.384]
```

**FR-V3 — Similarity Metrics**

The system shall support:
- **Cosine similarity** (default): `sim(a, b) = (a · b) / (|a| * |b|)`. Range: [-1, 1], higher is more similar.
- **Dot product**: `sim(a, b) = a · b`. Suitable when embeddings are unit-normalized.
- **Euclidean distance** (converted to similarity): `sim(a, b) = 1 / (1 + |a - b|)`.

**FR-V4 — Top-K Retrieval**

The VectorScan operator shall return the top-K results by similarity score. K is determined by the `LIMIT` clause or a configurable default (K=10).

**FR-V5 — Embedding Generation**

The system shall include a lightweight embedding client that can:
- Call a local sentence-transformers model (all-MiniLM-L6-v2, 384 dimensions) via a Python subprocess or HTTP microservice.
- Accept pre-computed embeddings as base64-encoded float32 arrays for offline use.

### 8.8 Hybrid Query Engine

**FR-H1 — Hybrid Query Detection**

The query planner shall detect a hybrid query when the WHERE clause contains both at least one `SimilarToPredicate` and at least one structured predicate (comparison or range).

**FR-H2 — Planning Strategies**

The planner shall support two hybrid execution strategies, selectable by rule:

**Vector-First (VF):** Retrieve top-K vector candidates, then apply structured filters to the candidate set.

**Filter-First (FF):** Apply structured filters to produce a reduced candidate set, then rank candidates by vector similarity.

**FR-H3 — Strategy Selection Rule**

```
if (estimated_filter_selectivity < 0.1):  # filter is highly selective
    strategy = FILTER_FIRST
else:
    strategy = VECTOR_FIRST
```

Selectivity is estimated from simple table statistics (row count, distinct value count per column) maintained by the catalog.

**FR-H4 — Rank Fusion**

The hybrid executor shall support two rank fusion algorithms:

**Reciprocal Rank Fusion (RRF):**
```
score(d) = Σ 1 / (k + rank_i(d))
```
where k=60 (standard constant), and the sum is over each ranking list in which document d appears.

**Weighted Linear Combination:**
```
score(d) = α * semantic_score(d) + β * (1 if filter_matches else 0)
```
where α and β are configurable (default: α=0.7, β=0.3).

**FR-H5 — Score Normalization**

Before fusion, all similarity scores shall be normalized to [0, 1] using min-max normalization over the candidate set:
```
normalized(s) = (s - min_score) / (max_score - min_score)
```

---

## 9. Non-Functional Requirements

### 9.1 Performance

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-P1** | Point lookup via B+ tree index on 10k rows | < 5 ms |
| **NFR-P2** | Full table scan on 10k rows | < 50 ms |
| **NFR-P3** | Brute-force vector similarity over 10k 384-dim vectors | < 200 ms |
| **NFR-P4** | Full hybrid query (vector + filter + fusion) on 10k rows | < 500 ms |
| **NFR-P5** | B+ tree insert (including WAL write) | < 10 ms |
| **NFR-P6** | WAL replay for 10k operations | < 2 seconds |

### 9.2 Reliability

| ID | Requirement |
|----|-------------|
| **NFR-R1** | The database must recover to a consistent state after any abrupt process termination. |
| **NFR-R2** | WAL replay must be idempotent: replaying the same WAL twice must not corrupt the database. |
| **NFR-R3** | B+ tree invariants (key ordering, parent-child consistency, leaf chain integrity) must hold after every operation. |

### 9.3 Observability

| ID | Requirement |
|----|-------------|
| **NFR-O1** | Every operator must emit timing metrics (open time, next() call count, total time, bytes read). |
| **NFR-O2** | The buffer pool must expose cache hit rate, eviction rate, and dirty page count in real time. |
| **NFR-O3** | The hybrid executor must expose candidate counts at each stage (generated, filtered, reranked). |

### 9.4 Maintainability

| ID | Requirement |
|----|-------------|
| **NFR-M1** | Each module must have a clearly defined interface with no circular dependencies. |
| **NFR-M2** | No module may exceed 2000 lines of code without a documented justification. |
| **NFR-M3** | All public interfaces must be documented with purpose, preconditions, and postconditions. |

### 9.5 Portability

| ID | Requirement |
|----|-------------|
| **NFR-PT1** | The core engine (Go) must compile and run on Linux, macOS, and WSL without modification. |
| **NFR-PT2** | The visualization UI (React) must run in any modern browser without installation. |

---

## 10. UI/UX Requirements

### 10.1 Execution Plan Visualizer

The UI shall render a query's execution plan as an interactive tree diagram:

```
                    ┌─────────────────┐
                    │  HybridScan     │
                    │  time: 342ms    │
                    │  rows: 5        │
                    └────────┬────────┘
              ┌──────────────┴───────────────┐
              │                              │
    ┌─────────┴────────┐          ┌──────────┴───────┐
    │  Filter           │          │  VectorScan      │
    │  year > 2024      │          │  top-K: 50       │
    │  time: 12ms       │          │  time: 198ms     │
    │  rows in: 10000   │          │  rows: 50        │
    │  rows out: 847    │          └──────────────────┘
    └─────────┬─────────┘
              │
    ┌─────────┴────────┐
    │  IndexScan       │
    │  idx: category   │
    │  time: 8ms       │
    │  rows: 10000     │
    └──────────────────┘
```

Each node shall be clickable, revealing detailed metrics for that operator.

### 10.2 B+ Tree Visualizer

The UI shall render the B+ tree as an interactive node-link diagram:
- Internal nodes rendered as rectangles with dividers for each key.
- Leaf nodes rendered with a distinct color and showing key-RID pairs.
- Leaf-to-leaf pointers shown as horizontal arrows.
- Insertions shall be animated, highlighting the path from root to leaf, then the split if it occurs.

### 10.3 Buffer Pool Heatmap

The UI shall render the buffer pool as a grid of cells, one per frame:
- **Blue**: clean, recently accessed
- **Orange**: dirty (modified, not yet flushed)
- **Red**: pinned
- **Gray**: empty / unused
- On hover: show pageID, pin count, dirty status, last access time.

### 10.4 WAL Log Viewer

The UI shall display WAL records as they are appended, in a scrollable, color-coded table:
- `INSERT` records: green
- `UPDATE` records: orange
- `DELETE` records: red
- `COMMIT` records: blue
- `CHECKPOINT` records: purple

Each record shall be expandable to show the full before/after images.

### 10.5 Vector Search Visualizer

For educational demonstration purposes, the UI shall include a 2D projection (PCA or t-SNE of the first two principal components) of the vector space, showing:
- All stored vectors as gray dots
- The query vector as a red star
- The top-K retrieved vectors as blue dots
- The filtered-out candidates as lighter blue dots

### 10.6 Hybrid Execution Timeline

The UI shall render a Gantt-style timeline of the hybrid query execution:
- Horizontal axis: wall clock time
- Rows: one per operator
- Shows parallelism (or lack thereof) and where time is spent

---

## 11. Demo Requirements

### 11.1 Final Presentation Demo

The presenter shall demonstrate, live, in sequence:

1. **Table creation** with a VECTOR column and B+ tree index on a categorical column.
2. **Data insertion** of 1,000+ research paper records with pre-computed embeddings.
3. **Structured query** using the indexed column — show the execution plan, confirm index scan is used.
4. **Semantic query** — show the vector scan executing, display similarity scores.
5. **Hybrid query** — execute the full example query, animate the execution plan, show rank fusion.
6. **Crash recovery** — kill the process mid-insert, restart, watch WAL replay restore consistency.
7. **B+ tree insertion** — insert 10 records, animate the tree structure and a split.

### 11.2 Viva / Interview Readiness

The presenter must be able to explain, without the UI:
- How a slotted page stores variable-length tuples
- Why WAL records must be written before page modifications (the WAL invariant)
- How B+ tree splits propagate upward
- Why brute-force cosine similarity is O(N * d) and when this is a problem
- The tradeoff between vector-first and filter-first hybrid execution
- What reciprocal rank fusion does and why it is preferable to simple score addition

### 11.3 GitHub README Requirements

The README must include:
- Architecture diagram (ASCII or image)
- Example queries with expected output
- Step-by-step build and run instructions
- Module dependency graph
- Performance benchmark results table
- Link to demo video (2-3 minutes)

---

## 12. Success Metrics

| Dimension | Metric | Target |
|-----------|--------|--------|
| **Correctness** | All unit tests pass | 100% |
| **Correctness** | Hybrid query returns results matching ground truth | ≥ 95% recall on test set |
| **Correctness** | WAL recovery produces identical state as no-crash execution | 100% |
| **Performance** | Hybrid query on 10k records | < 500 ms |
| **Explainability** | Execution plan accurately reflects actual execution | 100% |
| **Completeness** | All 8 functional modules implemented and tested | 100% |
| **Educational** | B+ tree visualization correctly reflects tree state | 100% |
| **Demo** | Full demo executable without errors in < 15 minutes | Pass |

---

# PART II — TECHNICAL REQUIREMENTS DOCUMENT

---

## 13. System Architecture

### 13.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   HybridDB System                        │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │              SQL Interface Layer                 │   │
│  │   ┌──────────┐  ┌──────────┐  ┌──────────────┐  │   │
│  │   │  Lexer   │→ │  Parser  │→ │  AST Builder │  │   │
│  │   └──────────┘  └──────────┘  └──────────────┘  │   │
│  └────────────────────────┬────────────────────────┘   │
│                           │  AST                        │
│  ┌────────────────────────▼────────────────────────┐   │
│  │              Query Planning Layer                │   │
│  │   ┌──────────────┐  ┌────────────────────────┐  │   │
│  │   │ Rule-Based   │  │  Operator Tree Builder  │  │   │
│  │   │ Planner      │  │  (Logical → Physical)   │  │   │
│  │   └──────────────┘  └────────────────────────┘  │   │
│  └────────────────────────┬────────────────────────┘   │
│                           │  Physical Plan              │
│  ┌────────────────────────▼────────────────────────┐   │
│  │              Execution Layer                     │   │
│  │  ┌─────────────────┐  ┌─────────────────────┐  │   │
│  │  │ Relational Exec │  │   Vector Exec        │  │   │
│  │  │ SeqScan         │  │   VectorScan         │  │   │
│  │  │ IndexScan       │  │   CosineSimilarity   │  │   │
│  │  │ Filter          │  │   EmbeddingClient    │  │   │
│  │  │ Projection      │  └──────────┬──────────┘  │   │
│  │  │ Sort, Limit     │             │              │   │
│  │  └────────┬────────┘             │              │   │
│  │           └──────────┬───────────┘              │   │
│  │                 ┌────▼─────────┐                │   │
│  │                 │ HybridScan  │                 │   │
│  │                 │ RankFusion  │                 │   │
│  │                 └─────────────┘                 │   │
│  └────────────────────────┬────────────────────────┘   │
│                           │                             │
│  ┌────────────────────────▼────────────────────────┐   │
│  │              Storage Layer                       │   │
│  │  ┌────────────┐  ┌───────────┐  ┌────────────┐  │   │
│  │  │ Buffer Pool│  │  B+ Tree  │  │    WAL     │  │   │
│  │  │ (LRU Cache)│  │  Index    │  │  Manager  │  │   │
│  │  └────────────┘  └───────────┘  └────────────┘  │   │
│  │  ┌───────────────────────────────────────────┐   │   │
│  │  │                  Pager                    │   │   │
│  │  │          (Disk I/O Abstraction)           │   │   │
│  │  └───────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │        Observability / Metrics Layer             │   │
│  │   MetricsCollector │ TraceRecorder │ UIServer   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 13.2 Module Dependency Rules

```
sql_parser     → (none, pure parsing)
query_planner  → sql_parser, catalog
executor       → query_planner, storage, vector_engine, metrics
storage        → pager, buffer_pool, wal
pager          → (OS file I/O only)
buffer_pool    → pager
wal            → pager
b_tree         → storage, wal
vector_engine  → storage, embedding_client
hybrid_exec    → executor, vector_engine, rank_fusion
metrics        → (no dependencies, pure collection)
ui_server      → metrics (read-only)
```

**Dependency invariant:** Lower layers must never import from upper layers. Circular dependencies are forbidden.

### 13.3 Request Lifecycle

```
Query lifecycle for:
SELECT * FROM papers WHERE year > 2024 AND embedding SIMILAR TO 'vector indexing'

1. SQL Interface
   └── Lexer tokenizes input → token stream
   └── Parser consumes tokens → AST

2. Query Planner
   └── Analyzes WHERE clause → detects SimilarToPredicate → HybridQuery
   └── Estimates filter selectivity from catalog stats
   └── Selects strategy: VECTOR_FIRST (selectivity > 0.1)
   └── Builds physical plan:
         HybridScan
           ├── VectorScan(papers, embed_query('vector indexing'), K=50)
           └── Filter(year > 2024)
                └── SeqScan(papers) [or IndexScan if index exists on year]

3. Execution
   └── Open HybridScan
         └── Open VectorScan → call EmbeddingClient → get query vector [0.1, 0.2, ...]
         └── Open Filter → Open SeqScan → pin buffer pool pages
         └── VectorScan.Next() → brute-force over all tuples → return top-50 candidates
         └── Filter each candidate by year > 2024 → 12 survive
         └── RankFusion: compute RRF scores → sort → return top-5

4. Metrics Collection
   └── Each operator emits: open_time, next_calls, rows_in, rows_out, pages_read

5. UI Update
   └── Execution trace pushed to UI server
   └── Execution plan tree rendered with actual metrics
```

---

## 14. Core Engine Design

### 14.1 Pager

**Purpose:** The pager is the single point of contact between the rest of the system and the operating system file I/O. All disk reads and writes go through the pager.

**Responsibilities:**
- Maintain the database file (single file per database, e.g., `mydb.hdb`).
- Allocate new pages by extending the file.
- Read a page into a provided buffer via `pread`.
- Write a buffer to a given page position via `pwrite`.
- Maintain a free page list for page reuse after deletions.

**API:**
```go
type Pager interface {
    ReadPage(pageID uint32, buf []byte) error
    WritePage(pageID uint32, buf []byte) error
    AllocatePage() (uint32, error)   // returns new pageID, extends file
    FreePage(pageID uint32) error    // adds to free list
    NumPages() uint32
    Close() error
}
```

**Internal Design:**
```
File Layout:
  Byte 0..4095:     DB Header Page (pageID = 0)
  Byte 4096..8191:  Page 1
  Byte 8192..12287: Page 2
  ...
  PageID N is at file offset: N * PAGE_SIZE
```

**Failure Modes:**
- Partial writes: mitigated by WAL (WAL is always written before page writes).
- File system errors: propagated as errors to callers; no silent corruption.

**Future Extensions:**
- Memory-mapped I/O (`mmap`) for read-heavy workloads.
- File pre-allocation to reduce filesystem fragmentation.
- Multiple file support for databases exceeding 4 GB.

### 14.2 Buffer Pool

**Purpose:** Avoid reading the same page from disk repeatedly by caching frequently accessed pages in memory.

**Data Structures:**
```
type Frame struct {
    pageID   uint32
    data     [PAGE_SIZE]byte
    pinCount int
    isDirty  bool
    lastUsed time.Time   // for LRU tracking
}

type BufferPool struct {
    frames   []*Frame
    pageMap  map[uint32]*Frame    // pageID → frame
    lruList  *list.List           // doubly-linked list for LRU
    lruMap   map[uint32]*list.Element  // pageID → LRU list element
    pager    Pager
    mu       sync.Mutex
}
```

**LRU Eviction Algorithm:**
```
FetchPage(pageID):
  1. if pageID in pageMap → move to front of LRU, increment pin, return
  2. find eviction candidate: LRU element with pinCount == 0
  3. if candidate is dirty → WritePage(candidate.pageID, candidate.data)
  4. remove candidate from pageMap and lruMap
  5. ReadPage(pageID, candidate.data)
  6. candidate.pageID = pageID, pinCount = 1, isDirty = false
  7. add to pageMap, lruMap front
  8. return frame
```

**Tradeoffs:**

| Policy | Pros | Cons |
|--------|------|------|
| **LRU** | Simple, good for temporal locality | Suffers on sequential scans (every page evicted immediately after use) |
| **CLOCK** | O(1) per access, no linked list | Slightly worse hit rate than LRU in practice |
| **LRU-K** | Handles repeated sequential scans | More complex implementation |

v1 uses LRU. The buffer pool interface is defined such that the replacement policy is a pluggable strategy object.

### 14.3 WAL Manager

**Purpose:** Ensure durability. If the process crashes after a commit but before dirty pages reach disk, the WAL allows recovery by replaying committed operations.

**WAL Invariant (never violated):**
> *A page modification must not reach disk until the WAL record describing that modification has been flushed to the WAL file.*

This means: `WAL.Flush(lsn)` must be called before `BufferPool.FlushPage(pageID)` for any page whose current LSN equals `lsn`.

**Recovery Algorithm (simplified ARIES):**

```
Startup Recovery:
  1. Read WAL from last checkpoint LSN to EOF.

  REDO Phase:
  2. For each record in LSN order:
     if record.type IN [INSERT, UPDATE, DELETE]:
       if page.pageLSN < record.lsn:   // effect not yet on disk
         apply record.afterImage to page
         page.pageLSN = record.lsn

  UNDO Phase:
  3. Collect all TxnIDs that have no COMMIT record.
  4. For each uncommitted txn, scan backward through WAL:
     apply record.beforeImage to page   // reverse the operation

  5. Write a new CHECKPOINT record.
  6. Database is now consistent and ready to serve queries.
```

**WAL File Format:**
```
WAL File:
+------------------+
| WAL File Header  |   (magic bytes, version, checkpoint LSN)
+------------------+
| Record 1         |
| Record 2         |
| ...              |
| Record N         |
+------------------+
```

Each record is prefixed with its length so the reader can scan forward and backward.

**Failure Modes:**
- Torn WAL write (partial record at EOF): detected by checksum mismatch; record is discarded and recovery stops there.
- WAL growing unbounded: mitigated by periodic checkpointing; WAL segments before the last checkpoint can be truncated.

---

## 15. Indexing Subsystems

### 15.1 B+ Tree

**Purpose:** Provide O(log N) point lookups and efficient O(log N + K) range scans over a single indexed column.

**Why B+ tree, not B tree or hash index:**
- B+ trees store all data in leaves, enabling efficient range scans via leaf chaining.
- B trees store data in internal nodes too, complicating range scans.
- Hash indexes do not support range queries.
- B+ trees have high fanout (hundreds of children per node at 4 KB pages), keeping tree height at 2-4 for millions of records.

**Fanout Analysis:**
```
For INT32 keys (4 bytes) and PageID pointers (4 bytes):
Internal node capacity:
  PAGE_SIZE - HEADER_SIZE = 4096 - 32 = 4064 bytes
  Keys: 4 bytes, Pointers: 4 bytes
  N keys + (N+1) pointers = 4064 → N ≈ 508 keys per internal node
  Fanout ≈ 509

For 3 levels: 509^3 ≈ 132 million records accessible in 3 disk reads.
```

**Split Logic (detailed):**
```
InsertIntoLeaf(leaf, key, rid):
  if leaf is full:
    newLeaf = AllocatePage()
    median = leaf.keys[N/2]
    newLeaf.keys = leaf.keys[N/2 .. N]
    leaf.keys = leaf.keys[0 .. N/2 - 1]
    newLeaf.next = leaf.next
    leaf.next = newLeaf.pageID
    InsertIntoParent(leaf.parent, median, newLeaf.pageID)
  else:
    leaf.insert(key, rid) in sorted order
    WAL.log(INSERT, leaf.pageID, ...)
    MarkDirty(leaf.pageID)

InsertIntoParent(parent, key, rightChildID):
  if parent is None:  // root split
    newRoot = AllocatePage()
    newRoot.keys = [key]
    newRoot.pointers = [leftChildID, rightChildID]
    catalog.rootPageID = newRoot.pageID
  else:
    parent.insert(key, rightChildID) in sorted order
    if parent is full: split parent recursively
```

**Leaf Chain for Range Scans:**
```
RangeScan(lo, hi):
  1. Traverse from root to leaf containing lo.
  2. Scan forward through leaf nodes via nextLeafID pointer.
  3. Yield all (key, rid) pairs where lo ≤ key ≤ hi.
  4. Stop when key > hi or next leaf is null.
```

### 15.2 Vector Search — Brute Force

**Purpose:** For educational clarity and implementation simplicity, v1 implements exact nearest-neighbor search.

**Algorithm:**
```
VectorScan(table, queryVec, K):
  results = MinHeap(capacity=K, comparator=byScore)
  for each tuple t in SeqScan(table):
    if t.embeddingCol is not null:
      score = cosineSimilarity(t.embeddingCol, queryVec)
      if results.size < K:
        results.push((score, t))
      elif score > results.min().score:
        results.pop()
        results.push((score, t))
  return results.sorted(descending)

cosineSimilarity(a, b):
  dot = 0; normA = 0; normB = 0
  for i in range(len(a)):
    dot  += a[i] * b[i]
    normA += a[i] * a[i]
    normB += b[i] * b[i]
  if normA == 0 or normB == 0: return 0
  return dot / (sqrt(normA) * sqrt(normB))
```

**Complexity:** O(N * d) where N = number of tuples, d = embedding dimensionality.

For N=10,000, d=384: approximately 3.84 million multiply-accumulate operations. At ~1 GFLOPS on a modern CPU, this takes approximately **3.84 ms** — well within the 200 ms NFR target.

**Why brute force is acceptable in v1:**

| Dataset Size | d=384 | d=768 | d=1536 |
|-------------|-------|-------|--------|
| 1,000 rows  | < 1ms | < 2ms | < 4ms |
| 10,000 rows | < 10ms | < 20ms | < 40ms |
| 100,000 rows | < 100ms | < 200ms | < 400ms |

For educational dataset sizes (≤ 100k rows), brute force is fast enough. HNSW introduces significant implementation complexity (graph construction, layer navigation, efSearch tuning) that is not educational value in v1.

**v2 Path to HNSW:**
- HNSW reduces search complexity from O(N * d) to O(log(N) * d * efSearch).
- Implementation requires a multi-layer proximity graph stored as additional pages.
- The VectorScan operator interface remains unchanged; the underlying algorithm is swapped.

---

## 16. Query Engine

### 16.1 Parser Implementation

**Lexer:**
The lexer converts the raw SQL string into a token stream. Each token has a type (KEYWORD, IDENTIFIER, INTEGER_LITERAL, FLOAT_LITERAL, STRING_LITERAL, OPERATOR, PUNCTUATION) and a value.

```
Input: "SELECT title, year FROM papers WHERE year > 2024 LIMIT 5"

Tokens:
  [KEYWORD:SELECT] [IDENT:title] [PUNCT:,] [IDENT:year]
  [KEYWORD:FROM] [IDENT:papers]
  [KEYWORD:WHERE] [IDENT:year] [OP:>] [INT:2024]
  [KEYWORD:LIMIT] [INT:5]
```

**AST Construction (recursive descent):**
```
parseSelect():
  expect(SELECT)
  cols = parseColumnList()
  expect(FROM)
  table = parseIdentifier()
  where = nil
  if peek() == WHERE: where = parseWhereClause()
  limit = nil
  if peek() == LIMIT: limit = parseLimit()
  return SelectStmt{cols, table, where, limit}

parseWhereClause():
  left = parsePredicate()
  if peek() == AND: return AndPredicate{left, parseWhereClause()}
  if peek() == OR:  return OrPredicate{left, parseWhereClause()}
  return left

parsePredicate():
  col = parseIdentifier()
  if peek() == SIMILAR:
    expect(SIMILAR); expect(TO)
    query = parseStringLiteral()
    return SimilarToPredicate{col, query}
  op = parseOperator()   // >, <, =, >=, <=, !=
  val = parseLiteral()
  return CompPredicate{col, op, val}
```

### 16.2 Query Planner

**Rule-Based Planning:**

The planner converts the AST into a physical operator tree using a fixed set of rules applied in priority order:

```
Planning Rules (applied top-to-bottom, first match wins):

Rule 1: If WHERE contains SimilarToPredicate AND structured predicates
        → HybridScan (see Section 17)

Rule 2: If WHERE contains SimilarToPredicate only
        → VectorScan(table, query, K=limit)

Rule 3: If WHERE contains CompPredicate on an indexed column
        → Filter(IndexScan(table, index), remainingPredicates)

Rule 4: If WHERE contains range predicate on an indexed column
        → Filter(IndexRangeScan(table, index, lo, hi), remainingPredicates)

Rule 5: Default
        → Filter(SeqScan(table), allPredicates)
```

**Operator Tree Example:**
```
Query: SELECT title FROM papers WHERE category = 'database' AND year > 2022 LIMIT 10

Physical Plan:
  Limit(10)
    └── Projection([title])
          └── Filter(year > 2022)
                └── IndexScan(papers, idx_category, 'database')
```

### 16.3 Execution Engine — Pull-Based Model

**Why pull-based (Volcano model)?**

The pull-based model (also called iterator model or Volcano model) has each operator expose a `Next()` method that produces one tuple at a time when called. The root operator drives execution by repeatedly calling `Next()`.

**Advantages:**
- Simple to implement and reason about.
- Memory-efficient: only one tuple is "in flight" at a time.
- Easy to add new operators without restructuring others.
- Easy to instrument: metrics collection happens inside each `Next()` call.

**Disadvantages (acknowledged tradeoffs):**
- Virtual function call overhead per tuple (significant for large datasets).
- Poor CPU cache utilization (tuple-at-a-time vs. vectorized column-at-a-time).
- Not amenable to SIMD optimization.

These disadvantages are irrelevant for educational dataset sizes and are well-understood tradeoffs that the implementer should be able to articulate.

**Operator Execution Trace:**
```
Execution of: Limit(10) → Projection → Filter(year > 2022) → IndexScan

Time 0: Limit.Open() → Projection.Open() → Filter.Open() → IndexScan.Open()
          IndexScan pins root page, navigates to leaf for 'database'

Time 1: Limit.Next() → Projection.Next() → Filter.Next() → IndexScan.Next()
          IndexScan returns tuple(1, 'HybridDB', 2024, ...)
          Filter evaluates: 2024 > 2022 → TRUE → passes tuple up
          Projection extracts [title] → ('HybridDB',)
          Limit decrements counter: 9 remaining

...

Time 10: Limit counter reaches 0 → returns nil
Time 11: Limit.Close() → Projection.Close() → Filter.Close() → IndexScan.Close()
           IndexScan unpins all pages
```

---

## 17. Hybrid Query Engine

### 17.1 The Core Problem

A hybrid query has two fundamentally different sub-problems:

1. **Semantic sub-problem:** Find the K documents most semantically similar to the query text. This is a continuous, graded problem.
2. **Structured sub-problem:** Find all documents satisfying a boolean predicate. This is a discrete, binary problem.

Neither result fully subsumes the other. A document might rank #1 semantically but fail the date filter. A document might perfectly satisfy all filters but be semantically irrelevant.

### 17.2 Execution Strategies

**Strategy A: Vector-First (VF)**

```
VectorFirst(table, queryVec, structuredPreds, K, Kover):
  1. candidates = VectorScan(table, queryVec, K=Kover)   // retrieve Kover > K
  2. filtered = [t for t in candidates if structuredPreds(t)]
  3. if len(filtered) >= K:
       return filtered[0:K]   // already ranked by similarity
  4. else:
       // Not enough candidates survived filtering
       // Option A: increase Kover and retry (adaptive)
       // Option B: fall back to FilterFirst for this query
       return filtered   // return what we have
```

**When to use VF:** When the structured filter is not very selective (most records pass the filter). In this case, starting with the best semantic candidates and filtering them is cheap.

**Failure mode of VF:** When the filter is highly selective (e.g., only 50 of 10,000 records satisfy it), many of the top-K semantic candidates may be filtered out. To get K final results, we would need to retrieve a very large Kover — potentially scanning the entire dataset, negating the benefit of vector-first retrieval.

**Strategy B: Filter-First (FF)**

```
FilterFirst(table, queryVec, structuredPreds, K):
  1. candidates = execute structured plan → all tuples satisfying preds
  2. for each candidate t:
       t.score = cosineSimilarity(t.embeddingCol, queryVec)
  3. sort candidates by score descending
  4. return candidates[0:K]
```

**When to use FF:** When the structured filter is highly selective (few records pass). Compute similarity only for the small set of filter-passing records.

**Failure mode of FF:** When the filter passes many records (low selectivity), this degenerates into computing similarity for every record — the same as a full VectorScan.

**Adaptive Strategy Selection:**

```
selectStrategy(table, structuredPreds, K):
  tableSize = catalog.rowCount(table)
  estimatedFilterOutput = estimateSelectivity(structuredPreds) * tableSize

  if estimatedFilterOutput < threshold:   // e.g., threshold = 0.1 * tableSize
    return FILTER_FIRST
  else:
    return VECTOR_FIRST

estimateSelectivity(pred):
  // Simple statistics-based estimation
  if pred is CompPredicate(col, '=', val):
    return 1.0 / catalog.distinctValues(col)
  if pred is RangePredicate(col, lo, hi):
    return (hi - lo) / (catalog.maxVal(col) - catalog.minVal(col))
  if pred is AndPredicate(left, right):
    return estimateSelectivity(left) * estimateSelectivity(right)  // independence assumption
```

### 17.3 Rank Fusion

After both strategies produce a candidate set, scores from different "channels" must be combined.

**Why not just use cosine similarity directly?**

When using FF, we only have semantic scores. When using VF, we have semantic ranking (position in top-K). Neither approach naturally combines:
- "This document ranked #3 in semantic similarity"
- "This document has year=2025 (matching the filter strongly)"

For v1, structured predicates are binary (match or no-match), so rank fusion is primarily about combining semantic score with filter satisfaction.

**Reciprocal Rank Fusion (RRF):**

```
RRF(semanticRanking, filterRanking, k=60):
  scores = {}
  for rank, doc in enumerate(semanticRanking):
    scores[doc] = scores.get(doc, 0) + 1.0 / (k + rank + 1)
  for rank, doc in enumerate(filterRanking):   // filterRanking = docs satisfying pred, by some natural order
    scores[doc] = scores.get(doc, 0) + 1.0 / (k + rank + 1)
  return sorted(scores.keys(), key=lambda d: scores[d], reverse=True)
```

**RRF Properties:**
- Robust to score scale differences (uses ranks, not raw scores).
- A document in the top-10 of both lists will always outrank one that appears in only one list.
- k=60 is the standard constant, empirically validated in the IR literature.

**Weighted Linear Combination (WLC) — Alternative:**

```
WLC(doc, alpha=0.7, beta=0.3):
  semantic_normalized = normalize(cosine_similarity(doc, queryVec))   // to [0,1]
  filter_score = 1.0 if all_preds_satisfied(doc) else 0.0
  return alpha * semantic_normalized + beta * filter_score
```

**Tradeoff Comparison:**

| Criterion | RRF | WLC |
|-----------|-----|-----|
| **Score normalization required** | No | Yes |
| **Tuning parameters** | k (stable, k=60) | α, β (query-dependent) |
| **Interpretability** | Moderate | High |
| **Robustness to outlier scores** | High | Low |
| **Literature support** | Strong (Cormack et al.) | Moderate |

### 17.4 Execution Example — Full Trace

**Query:**
```sql
SELECT title, year, similarity_score
FROM research_papers
WHERE year > 2024
  AND category = 'database'
  AND embedding SIMILAR TO 'vector indexing optimization'
LIMIT 5;
```

**Assumed statistics:**
- Table: 10,000 rows
- `year > 2024`: selectivity ≈ 0.30 (3,000 rows)
- `category = 'database'`: selectivity ≈ 0.08 (800 rows)
- Combined AND selectivity: ≈ 0.024 (240 rows)

**Strategy selection:**
```
estimatedFilterOutput = 0.024 * 10000 = 240
threshold = 0.1 * 10000 = 1000
240 < 1000 → FILTER_FIRST
```

**Execution trace:**
```
Step 1: Filter Phase
  IndexScan(category='database') → 800 candidates
  Filter(year > 2024) over 800 candidates → 240 candidates

Step 2: Embedding Query
  EmbeddingClient('vector indexing optimization') → queryVec (384 dims)
  Latency: ~50ms (first call; cached thereafter)

Step 3: Similarity Scoring
  for each of 240 candidates:
    score = cosineSimilarity(candidate.embedding, queryVec)
  240 * 384 multiply-accumulates ≈ 92,160 operations → < 1ms

Step 4: Ranking
  Sort 240 candidates by score descending

Step 5: Projection + Limit
  Return top-5 with (title, year, similarity_score)
```

**Operator tree (physical):**
```
Projection([title, year, similarity_score])
  └── Limit(5)
        └── HybridScan[FilterFirst]
              └── VectorScore(queryVec)
                    └── Filter(year > 2024)
                          └── IndexScan(papers, idx_category, 'database')
```

**Result:**
```
| title                              | year | similarity_score |
|------------------------------------|------|------------------|
| Efficient Vector Indexing at Scale | 2025 | 0.923            |
| Learned Index Structures for ANN   | 2025 | 0.891            |
| DiskANN: Disk-Based ANNS           | 2025 | 0.876            |
| HNSW Graph Optimization Techniques | 2025 | 0.854            |
| Product Quantization Revisited     | 2025 | 0.831            |
```

---

## 18. Vector Storage

### 18.1 Embedding Serialization

Embeddings are stored as typed column values within the slotted page layout. The serialization format for a 384-dimensional float32 vector:

```
Vector Wire Format:
+--------------------+
| TypeTag: 0x05 (1B) |
| Dimensions: 384 (2B, uint16, big-endian) |
| float32[0] (4B, IEEE 754, little-endian) |
| float32[1] (4B)    |
| ...                |
| float32[383] (4B)  |
+--------------------+

Total size: 1 + 2 + 384 * 4 = 1539 bytes per embedding
```

### 18.2 Storage Overhead Analysis

| Embedding Model | Dimensions | Bytes per Row | 10k rows | 100k rows |
|-----------------|-----------|---------------|----------|-----------|
| MiniLM-L6-v2 | 384 | 1,539 B | ~15 MB | ~150 MB |
| BERT-base | 768 | 3,075 B | ~30 MB | ~300 MB |
| OpenAI text-embedding-3-small | 1,536 | 6,147 B | ~60 MB | ~600 MB |

This storage overhead is the primary reason vector databases exist as separate systems. HybridDB co-locates embeddings with relational data, accepting the storage cost in exchange for architectural simplicity.

### 18.3 In-Memory Layout for VectorScan

During a full VectorScan, all embeddings must be loaded into memory. For 10,000 rows with 384-dim embeddings:
```
Memory requirement: 10,000 * 384 * 4 bytes = 15.36 MB
```

This fits comfortably in the buffer pool (default: 256 frames * 4KB = 1 MB for pages; the buffer pool must be sized appropriately for vector-heavy workloads). In practice, embeddings are read page by page during the scan, not all at once.

### 18.4 Compression Possibilities (v2)

- **Scalar quantization:** Store float32 as int8 (25% size). Cosine similarity computed in int8 arithmetic, converted to float for scoring.
- **Product quantization (PQ):** Split 384-dim vector into M=48 sub-vectors of 8 dims each. Quantize each with 256 centroids (8 bits). Reduces to 48 bytes per vector (96.8% compression). Requires codebook training.
- Both are v2 extensions. v1 stores raw float32 for simplicity and correctness.

---

## 19. Observability & Educational Tooling

### 19.1 Metrics Collection Architecture

Every operator wraps its core logic in a metrics collection harness:

```go
type OperatorMetrics struct {
    OperatorName  string
    OpenTime      time.Duration
    CloseTime     time.Duration
    NextCalls     uint64
    RowsIn        uint64    // for filter operators
    RowsOut       uint64
    PagesRead     uint64
    PagesWritten  uint64
    BytesRead     uint64
    TotalTime     time.Duration
}

// Each operator's Next() method:
func (op *FilterOperator) Next() (*Tuple, error) {
    start := time.Now()
    defer func() { op.metrics.TotalTime += time.Since(start) }()
    op.metrics.NextCalls++
    for {
        tuple, err := op.child.Next()
        op.metrics.RowsIn++
        if tuple == nil || err != nil { return tuple, err }
        if op.predicate.Evaluate(tuple) {
            op.metrics.RowsOut++
            return tuple, nil
        }
    }
}
```

### 19.2 Execution Trace Format

The execution trace is a structured JSON object emitted at query completion:

```json
{
  "queryID": "q-20241201-001",
  "totalTimeMs": 342,
  "plan": {
    "operator": "HybridScan",
    "strategy": "FILTER_FIRST",
    "timeMs": 342,
    "rowsOut": 5,
    "children": [
      {
        "operator": "VectorScore",
        "timeMs": 2,
        "rowsIn": 240,
        "rowsOut": 240
      },
      {
        "operator": "Filter",
        "predicate": "year > 2024",
        "timeMs": 12,
        "rowsIn": 800,
        "rowsOut": 240,
        "children": [
          {
            "operator": "IndexScan",
            "index": "idx_category",
            "key": "database",
            "timeMs": 8,
            "pagesRead": 4,
            "rowsOut": 800
          }
        ]
      }
    ]
  },
  "bufferPool": {
    "hitRate": 0.73,
    "evictions": 2,
    "dirtyPages": 0
  },
  "hybrid": {
    "candidatesGenerated": 800,
    "candidatesFiltered": 240,
    "candidatesScored": 240,
    "finalResults": 5
  }
}
```

### 19.3 Educational Value of Observability

The observability layer transforms HybridDB from a black-box database into a **transparent learning instrument**. The key educational payoffs:

1. **Understanding filter selectivity**: Watching `rowsIn: 10000 → rowsOut: 847` on a Filter operator concretely demonstrates what selectivity means.

2. **Understanding buffer pool behavior**: Watching the hit rate change during sequential scans vs. indexed lookups teaches when caching helps and when it doesn't.

3. **Understanding the cost of vector search**: Watching VectorScan take 198ms vs IndexScan taking 8ms quantifies the tradeoff between semantic richness and computational cost.

4. **Understanding rank fusion**: Seeing `candidatesGenerated: 50, candidatesFiltered: 12, finalResults: 5` makes the pipeline concrete.

---

## 20. Benchmarking Strategy

### 20.1 Dataset

**Primary Dataset: Synthetic Research Papers**

```python
# Schema
CREATE TABLE research_papers (
    id       INT,
    title    VARCHAR,
    abstract VARCHAR,
    year     INT,          -- range: [2015, 2025]
    category VARCHAR,      -- values: database, ml, systems, networking, theory
    citations INT,
    embedding VECTOR(384)  -- all-MiniLM-L6-v2 embeddings of abstract
);

# Size: 10,000 rows for standard benchmarks; 100,000 rows for stress tests
```

**Data Generation:**
- Titles and abstracts: sampled from a curated set of real arXiv abstracts (pre-downloaded).
- Embeddings: pre-computed using all-MiniLM-L6-v2 and stored as binary files for offline loading.
- Structural fields: generated with controlled distributions to enable selectivity testing.

### 20.2 Benchmark Suite

**B1 — Point Lookup Benchmark**
```
Query: SELECT * FROM research_papers WHERE id = ?
Dataset: 10k rows, indexed on id
Metric: Latency (p50, p95, p99), pages read
Expected: < 5ms p99
```

**B2 — Range Scan Benchmark**
```
Query: SELECT * FROM research_papers WHERE year BETWEEN ? AND ?
Dataset: 10k rows, indexed on year
Variable: range width (1%, 10%, 50% of domain)
Metric: Latency vs range width, pages read
```

**B3 — Vector Search Benchmark**
```
Query: SELECT * FROM research_papers WHERE embedding SIMILAR TO ? LIMIT 10
Dataset: varying from 1k to 100k rows
Metric: Latency vs N, recall@10 (ground truth via exhaustive search)
```

**B4 — Hybrid Query Benchmark**
```
Query: Full hybrid query with varying filter selectivities
Conditions:
  High selectivity (category = 'database' → 20% of rows, year > 2024 → 10% = 2% combined)
  Low selectivity (category IN [...5 values...] → 100%, year > 2020 → 80% = 80% combined)
Metric: Latency, strategy selected, recall@5 vs ground truth
```

**B5 — Recovery Benchmark**
```
Workload: 10k sequential inserts, crash after 7k, replay and verify
Metric: Recovery time, correctness (all 7k committed rows present, none of the remaining)
```

### 20.3 Benchmark Results Presentation

Results shall be presented as:
1. **Latency tables** (p50, p95, p99 for each benchmark)
2. **Latency vs N charts** (for vector search scalability)
3. **Strategy comparison chart** (VF vs FF latency for varying selectivities)
4. **Recall@K chart** (for hybrid queries vs pure vector search baseline)

---

## 21. Testing Strategy

### 21.1 Unit Tests

| Module | Test Cases |
|--------|------------|
| **Pager** | Read/write correctness, page allocation monotonicity, file extension |
| **Slotted Pages** | Insert tuple, read tuple, delete tuple, compaction trigger, fragmentation handling |
| **Buffer Pool** | Cache hit, cache miss, LRU eviction order, dirty page write-before-evict, pin-unpin correctness |
| **WAL** | Record serialization/deserialization, checksum verification, sequential LSN assignment |
| **B+ Tree** | Insert 1-1000 keys (verify order), range scan correctness, delete with borrow, delete with merge, split propagation to root |
| **Parser** | Parse all supported statement types, error recovery for malformed SQL, AST structure correctness |
| **Filter Operator** | All comparison operators, NULL handling, AND/OR predicate combinations |
| **VectorScan** | Cosine similarity correctness (known vectors), top-K accuracy vs exhaustive search, edge cases (zero vector, identical vectors) |
| **RankFusion** | RRF correctness (known rankings), WLC correctness, score normalization |

### 21.2 Integration Tests

**IT-1 — End-to-End Structured Query**
```
Create table → Insert 100 rows → Create index → Execute range query
Assert: All rows in range returned, no extra rows, correct sort order
```

**IT-2 — End-to-End Hybrid Query**
```
Create table with VECTOR column → Insert 100 rows with embeddings
Execute hybrid query → Compare result set to exhaustive ground truth
Assert: Recall@5 ≥ 0.80 (some loss expected with filter-first strategy)
```

**IT-3 — WAL Recovery**
```
Begin insert of 100 rows → commit after 50 → simulate crash
Restart database → execute SELECT COUNT(*)
Assert: exactly 50 rows present
```

**IT-4 — B+ Tree Invariants**
```
After each of 1000 random insertions and 500 deletions:
Assert: all keys retrievable by Search()
Assert: all leaf keys in sorted order
Assert: leaf chain traversal yields all keys
Assert: no node exceeds maxKeys or underflows below minKeys
```

### 21.3 Crash Simulation

Crash simulation is implemented by injecting a `panic()` or `os.Exit(1)` at deterministic points:
- After WAL write, before page write (tests redo recovery)
- Mid-insert (tuple partially written to page)
- During B+ tree split (new node allocated but parent not yet updated)

Each crash point has a corresponding recovery test verifying the database is consistent post-restart.

### 21.4 Property-Based Testing (Recommended)

For B+ tree and WAL, property-based testing (fuzzing) generates random sequences of operations and verifies invariants hold after each:

```go
// Hypothesis: for any sequence of insertions and deletions,
// all inserted-but-not-deleted keys are findable
for _, ops := range generateRandomOpSequences(1000) {
    db := NewEmptyDB()
    groundTruth := map[Key]bool{}
    for _, op := range ops:
        applyOp(db, op, groundTruth)
        verifyInvariants(db, groundTruth)
}
```

---

## 22. Semester Roadmap

### 22.1 Milestone Overview

```
Week  1-2:  Phase 1 — Core Storage Foundation
Week  3:    Phase 2 — Buffer Pool + WAL
Week  4-5:  Phase 3 — B+ Tree Index
Week  6-7:  Phase 4 — SQL Layer (Parser + Planner + Basic Executor)
Week  8:    Phase 5 — Vector Engine
Week  9-10: Phase 6 — Hybrid Query Engine
Week  11:   Phase 7 — Observability UI
Week  12:   Phase 8 — Integration, Benchmarking, Polish
Week  13:   Demo Preparation
Week  14:   Final Presentation + Viva
```

### 22.2 Detailed Phase Breakdown

**Phase 1 — Core Storage Foundation (Weeks 1-2)**

Deliverables:
- `Pager`: single-file page I/O
- `SlottedPage`: layout, insert, delete, compaction
- `Tuple`: serialization for INT, VARCHAR, FLOAT32, VECTOR, NULL
- `Catalog`: in-memory schema registry backed by Page 0

Milestone gate: Can create a table, insert 100 rows, and read them back after process restart.

---

**Phase 2 — Buffer Pool + WAL (Week 3)**

Deliverables:
- `BufferPool` with LRU replacement
- `WALManager` with append, flush, and checkpoint
- Integration: all page modifications go through WAL before buffer pool

Milestone gate: Can demonstrate crash recovery — insert 50 rows, kill process, restart, verify 50 rows are present.

---

**Phase 3 — B+ Tree Index (Weeks 4-5)**

Deliverables:
- `BPlusTree`: insert, search, range scan, delete
- Leaf node chaining
- WAL integration for node modifications
- B+ tree visualization data export (JSON for UI)

Milestone gate: Insert 10,000 records, perform 100 random searches and range scans — all correct. Verify tree invariants with automated tests.

---

**Phase 4 — SQL Layer (Weeks 6-7)**

Deliverables:
- `Lexer` + `Parser` → AST
- `Planner`: rule-based, produces operator trees
- Operators: SeqScan, IndexScan, Filter, Projection, Sort, Limit
- EXPLAIN support

Milestone gate: Execute `SELECT title FROM papers WHERE year > 2022 ORDER BY year LIMIT 10` correctly, with EXPLAIN output showing operator tree.

---

**Phase 5 — Vector Engine (Week 8)**

Deliverables:
- VECTOR column type in parser and tuple serializer
- `EmbeddingClient`: wraps a local all-MiniLM-L6-v2 model
- `VectorScan` operator: brute-force cosine similarity, top-K
- SIMILAR TO syntax in parser and planner

Milestone gate: Execute `SELECT * FROM papers WHERE embedding SIMILAR TO 'graph neural networks' LIMIT 5` and get semantically relevant results.

---

**Phase 6 — Hybrid Query Engine (Weeks 9-10)**

Deliverables:
- Hybrid query detection in planner
- Filter-first and vector-first execution strategies
- Selectivity estimation from catalog statistics
- RRF and WLC rank fusion
- HybridScan operator

Milestone gate: Execute the full example hybrid query correctly. Compare results to ground truth (exhaustive search). Demonstrate strategy switching based on filter selectivity.

---

**Phase 7 — Observability UI (Week 11)**

Deliverables:
- Go HTTP server exposing metrics and execution traces as JSON
- React frontend with:
  - Execution plan tree visualizer (interactive, real-time)
  - Buffer pool heatmap
  - B+ tree visualizer
  - WAL log viewer
  - Hybrid execution metrics panel

Milestone gate: Run a hybrid query, watch the execution plan animate in the UI, inspect buffer pool state.

---

**Phase 8 — Integration, Benchmarking, Polish (Week 12)**

Deliverables:
- Full benchmark suite execution (B1-B5)
- Benchmark results documented in README
- All unit and integration tests passing
- Demo script prepared and rehearsed

---

### 22.3 Dependency Graph

```
Phase 1 (Storage)
    │
    ▼
Phase 2 (Buffer Pool + WAL)
    │
    ├──────────────────────────┐
    ▼                          ▼
Phase 3 (B+ Tree)         Phase 4 (SQL Layer)
    │                          │
    └──────────┬───────────────┘
               ▼
          Phase 5 (Vector Engine)
               │
               ▼
          Phase 6 (Hybrid Engine)
               │
               ▼
          Phase 7 (UI)
               │
               ▼
          Phase 8 (Integration)
```

---

## 23. Risk Analysis

### 23.1 Technical Risks

**Risk T1 — B+ Tree Bugs**

*Likelihood:* High. B+ tree split and merge logic is notoriously difficult to implement correctly. Off-by-one errors in key distribution, incorrect parent pointer updates, and leaf chain corruption are common failure modes.

*Impact:* High. A corrupted B+ tree silently returns wrong results or causes crashes on range scans.

*Mitigation:*
- Implement with extensive automated tests before any other index-dependent component.
- Add invariant checking (`assertInvariants()`) called after every mutation during testing.
- Use the B+ tree visualizer during development to catch structural errors visually.
- Implement a `VERIFY INDEX` command that traverses the entire tree and validates all invariants.

---

**Risk T2 — WAL Correctness**

*Likelihood:* Medium. The WAL invariant (log before write) is easy to state but easy to violate inadvertently, especially when adding new code paths.

*Impact:* Critical. A violated WAL invariant means crash recovery may miss operations, producing a silently incorrect database state.

*Mitigation:*
- Enforce the WAL invariant at the buffer pool level: `MarkDirty(pageID, lsn)` internally calls `WAL.EnsureFlushed(lsn)` before returning.
- Test crash recovery for every code path that writes to pages.

---

**Risk T3 — Scope Explosion in Parser**

*Likelihood:* Medium. SQL grammar has many edge cases. Handling `NULL`, operator precedence, quoted identifiers, and escape sequences can consume unexpected time.

*Impact:* Medium. Parser bugs produce confusing query failures that are hard to debug.

*Mitigation:*
- Define the grammar subset explicitly upfront (see FR-P3) and refuse to implement anything not on the list.
- Use a recursive descent parser (simpler to debug than parser generators).
- Return clear, structured parse errors with position information.

---

**Risk T4 — Vector Memory Overhead**

*Likelihood:* Medium. 384-dim float32 embeddings are 1.5 KB per row. At 100k rows, this is 150 MB of embedding data — significant for a single-process educational system.

*Impact:* Low to Medium. The system may run slowly or OOM on machines with less than 4 GB RAM.

*Mitigation:*
- Default dataset size is 10,000 rows (15 MB of embeddings) — well within limits.
- Document the memory formula so users can calculate requirements for larger datasets.
- Implement streaming VectorScan (page-by-page) to avoid loading all embeddings simultaneously.

---

**Risk T5 — Embedding Latency**

*Likelihood:* High. The first call to the embedding model (all-MiniLM-L6-v2 via Python subprocess) may take 500ms+ due to model loading.

*Impact:* Low. Only affects cold-start latency of the first semantic query.

*Mitigation:*
- Keep the embedding model process alive between queries (persistent subprocess with stdin/stdout protocol).
- Cache the query embedding for identical query strings.
- Pre-compute embeddings for all demo queries and store as static files for the demo.

---

**Risk T6 — Hybrid Planner Correctness**

*Likelihood:* Medium. The selectivity estimation is a heuristic. The planner may choose the wrong strategy for some queries.

*Impact:* Low. Wrong strategy choice affects latency, not correctness. Results are always correct regardless of strategy.

*Mitigation:*
- Accept that selectivity estimation is imprecise.
- Add a `FORCE STRATEGY (VECTOR_FIRST | FILTER_FIRST)` hint syntax for the demo.
- Document the estimation algorithm and its assumptions (independence, uniform distribution) clearly.

---

### 23.2 Scope Risks

**Risk S1 — Feature Creep**

The temptation to add joins, aggregations, MVCC, or HNSW within v1 is the single greatest risk to project completion.

*Mitigation:* The Non-Goals section (Section 5) is a written commitment. Any proposed addition must be justified against the project timeline and documented as v2 work if not implemented.

**Risk S2 — UI Complexity**

Building a polished React visualization UI can consume significant time if scoped too broadly.

*Mitigation:* The UI is implemented last (Phase 7) with a fixed one-week budget. If time runs short, the UI is simplified to a static execution plan JSON display rather than an animated interactive visualization.

---

## 24. Future Extensions

### 24.1 V2 Extensions (Post-Semester)

**HNSW Vector Index**

Replace brute-force cosine search with a Hierarchical Navigable Small World (HNSW) graph index. This reduces search complexity from O(N * d) to O(log(N) * d * efSearch) at the cost of:
- Additional index construction time
- A multi-layer graph data structure requiring its own storage pages
- Approximate (not exact) results, requiring recall@K evaluation

Not included in v1 because brute force is sufficient for educational dataset sizes and HNSW implementation complexity exceeds the value it adds educationally.

**MVCC and Snapshot Isolation**

Add a version chain to each tuple, allowing concurrent readers to see a consistent snapshot of the database without locking writers. Requires:
- Transaction timestamp management
- Version chain traversal in all scan operators
- Garbage collection of old versions

Not included in v1 because single-threaded correctness is a prerequisite and the implementation complexity is substantial.

**Temporal Queries**

Add bitemporal columns (`valid_from`, `valid_to`, `txn_from`, `txn_to`) and query syntax:
```sql
SELECT * FROM papers AS OF TIMESTAMP '2024-01-01'
```
Interesting research direction connecting to the temporal database literature.

**Cost-Based Query Optimizer**

Replace rule-based planning with statistics-driven cost estimation:
- Collect column histograms for more accurate selectivity estimates
- Implement a cost model for each operator (I/O cost, CPU cost)
- Use dynamic programming to enumerate plan alternatives and select the minimum-cost plan

This is a complete sub-project (4+ weeks) and is explicitly deferred to v2.

**ANN Indexes: IVF and Product Quantization**

- **IVF (Inverted File Index):** Cluster vectors into Voronoi cells, search only nearby cells.
- **PQ (Product Quantization):** Compress vectors for faster approximate distance computation.
- Both reduce memory footprint and improve search throughput at large scales.

**Graph Relationships**

Add edge tables and graph traversal operators, moving toward a multi-model database that handles relational, vector, and graph data natively.

**Streaming Ingestion**

Add a write-ahead ingestion path that handles continuous document insertion with incremental embedding computation and index updates, without requiring full index rebuilds.

### 24.2 Why These Are v2, Not v1

Each of these extensions is genuinely valuable and architecturally interesting. They are deferred from v1 not because they are unimportant, but because:

1. **They depend on a correct v1 foundation.** HNSW requires a correct pager and buffer pool. MVCC requires a correct WAL. A cost-based optimizer requires a correct rule-based planner to be the baseline.

2. **They are each a full project.** A correct HNSW implementation alone is a publishable research contribution. Adding it to v1 would halve the quality of everything else.

3. **The educational goals are already fully met by v1.** A student who can explain slotted pages, B+ tree splits, WAL recovery, cosine similarity, and rank fusion has demonstrated deep systems understanding. Adding HNSW does not deepen that understanding — it extends it into a different specialization.

---

## Appendix A — Technology Stack

| Component | Technology | Justification |
|-----------|------------|---------------|
| **Core Engine** | Go 1.22+ | Simple systems programming, excellent standard library, good concurrency primitives, fast compilation |
| **Visualization UI** | React + TypeScript | Standard frontend stack, excellent graph libraries (D3.js for tree visualization) |
| **Embedding Model** | Python + sentence-transformers (all-MiniLM-L6-v2) | 384 dimensions, lightweight, runs on CPU, MIT license |
| **UI-Engine Protocol** | HTTP/JSON (localhost) | Simple, debuggable, no additional dependencies |
| **Test Framework** | Go `testing` package + `testify` | Standard, well-integrated, property-based testing via `rapid` |
| **Build System** | Go modules + Makefile | Standard Go tooling |

## Appendix B — Glossary

| Term | Definition |
|------|-----------|
| **LSN** | Log Sequence Number. A monotonically increasing integer identifying a WAL record. |
| **RID** | Record Identifier. A (pageID, slotID) pair uniquely identifying a tuple. |
| **Fanout** | The number of children an internal B+ tree node can have. Higher fanout = shorter tree. |
| **Selectivity** | The fraction of rows satisfying a predicate. Selectivity 0.1 means 10% of rows pass. |
| **RRF** | Reciprocal Rank Fusion. A rank fusion algorithm combining multiple ranked lists. |
| **ANN** | Approximate Nearest Neighbor. A class of algorithms that trade recall for speed in vector search. |
| **HNSW** | Hierarchical Navigable Small World. A graph-based ANN index with O(log N) search complexity. |
| **WAL Invariant** | The rule that WAL records must reach disk before the page modifications they describe. |
| **Brute-force search** | Exact nearest neighbor search by computing similarity against every stored vector. O(N*d). |
| **Pull-based execution** | The Volcano/iterator model where operators produce tuples on demand via `Next()` calls. |
| **Hybrid query** | A query combining semantic vector similarity with structured relational predicates. |
| **Rank fusion** | An algorithm for combining ranked result sets from heterogeneous retrieval systems. |

---

*HybridDB PRD + TRD — Version 1.0*
*A modern educational database engine exploring hybrid semantic + relational retrieval*