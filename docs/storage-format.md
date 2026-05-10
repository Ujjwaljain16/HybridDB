# HybridDB Storage Format Specification

This document defines the absolute source-of-truth for binary structures in HybridDB. All implementations MUST conform to these exact byte layouts.

## 1. Global Rules

- **Endianness**: Little-Endian (LE) for all multi-byte integers and floating-point numbers.
- **Page Size**: Exactly 4096 bytes (`PageSize = 4096`).
- **Format Version**: Current version is `1`.
- **Unused Space Zeroing**: All unused bytes MUST be explicitly zeroed during serialization to guarantee determinism.
- **RID Formalization**: `RID = (PageID, SlotIndex)`. This uniquely identifies any tuple.

## 2. Page Layout

Every page in HybridDB follows a strict 64-byte header followed by payload.

### 2.1 Page Header (64 Bytes)

```text
+-----------------------+--------+-------------------------------------------------+
| Field                 | Size   | Description                                     |
+-----------------------+--------+-------------------------------------------------+
| Magic Bytes           | 4B     | Ascii "HYDB" (0x48 0x59 0x44 0x42)              |
| Format Version        | 2B     | Storage Format Version (1)                      |
| Page Type             | 2B     | Enumeration defining the page's contents        |
| Page ID               | 8B     | Unique 64-bit ID for the page                   |
| LSN                   | 8B     | Log Sequence Number (reserved for recovery)     |
| Lower Free Space Bound| 2B     | Byte offset to the end of the slot array        |
| Upper Free Space Bound| 2B     | Byte offset to the start of the tuple region    |
| Slot Count            | 4B     | Number of entries in the slot array             |
| Checksum              | 4B     | CRC32C over bytes 0-31 and 36-4095              |
| Reserved              | 28B    | Reserved (MVCC, compressed, dirty, visibility)  |
+-----------------------+--------+-------------------------------------------------+
```
*Note: The Checksum field (bytes 32-35) MUST be zeroed out in memory before the CRC32C computation is performed over the page.*

### 2.2 Page Types

```go
const (
    PageTypeFree          = 0x0000
    PageTypeSlotted       = 0x0001
    PageTypeBTreeInternal = 0x0002
    PageTypeBTreeLeaf     = 0x0003
)
```
*Unknown PageType values MUST fail validation immediately.*

### 2.3 Page Initialization Rules
New pages MUST be initialized following this strict sequence:
1. Zero all 4096 bytes.
2. Initialize header fields (Magic, Version, PageType, PageID).
3. Initialize boundaries (`LowerFreeSpaceBoundary = 64`, `UpperFreeSpaceBoundary = 4096`).
4. Compute and write the checksum last.

### 2.4 Slotted Page Payload

A Slotted Page is split into two regions growing towards each other:
1. **Slot Array**: Grows toward higher addresses from byte 64.
2. **Tuple Data**: Grows toward lower addresses from byte 4095.

```text
0       64       LowerBound                         UpperBound                 4096
+--------+--------+--------+--------------------------+----------+----------+
| Header | Slot 0 | Slot 1 | -----> FREE SPACE <----- | Tuple 1  | Tuple 0  |
+--------+--------+--------+--------------------------+----------+----------+
```
*Maximum tuple size MUST be less than or equal to the usable page space (4096 - 64 = 4032 bytes).*

### 2.5 Slot Entry (8 Bytes)

Slot indices MUST remain stable during compaction to guarantee stable Record IDs (RIDs).

```text
+-----------------------+--------+-------------------------------------------------+
| Field                 | Size   | Description                                     |
+-----------------------+--------+-------------------------------------------------+
| Tuple Offset          | 2B     | Offset from byte 0 to start of Tuple Data       |
| Tuple Length          | 2B     | Size of Tuple Data in bytes                     |
| Flags                 | 2B     | 0x01: Deleted, 0x02: Redirected                 |
| Reserved              | 2B     | Reserved for future use                         |
+-----------------------+--------+-------------------------------------------------+
```
*Note: Current implementation assumes 4KB pages. Tuple offsets are uint16. Changing page size above 64KB requires a format revision.*

## 3. Tuple Layout

Tuples use a rigid format with explicit column tracking to enable safe schema evolution.

### 3.1 Tuple Header (16 Bytes)

```text
+-----------------------+--------+-------------------------------------------------+
| Field                 | Size   | Description                                     |
+-----------------------+--------+-------------------------------------------------+
| Total Length          | 4B     | Total byte length of the serialized tuple       |
| Column Count          | 2B     | Number of columns serialized in this tuple      |
| Null Bitmap Offset    | 2B     | Offset from tuple start to Null Bitmap          |
| Variable Data Offset  | 2B     | Offset from tuple start to Variable Data        |
| Tuple Version         | 2B     | Tuple format/schema version                     |
| Reserved              | 4B     | Reserved for MVCC transaction IDs               |
+-----------------------+--------+-------------------------------------------------+
```

### 3.2 Tuple Payload

Immediately following the header:
1. **Fixed Data Region**: Stores fixed-width column values (INT32, INT64, FLOAT32, BOOL) inline. For variable-width types (VARCHAR, VECTOR), stores a 4-byte offset and 4-byte length pointing into the Variable Data Region.
2. **Null Bitmap**: `ceil(Column Count / 8)` bytes. Bit ordering is **LSB-first**. Bit `1` means the column is NULL.
3. **Variable Data Region**: Concatenated payload for VARCHAR and VECTOR. Unused bytes must be zeroed.

## 4. Vector Encoding

Vectors are strictly composed of 32-bit floats.
Before reading payload, enforce `expectedBytes = dimensions * 4`.

```text
+-----------------------+--------+-------------------------------------------------+
| Field                 | Size   | Description                                     |
+-----------------------+--------+-------------------------------------------------+
| Dimension Count       | 4B     | Number of dimensions (uint32)                   |
| Float Data            | Varies | IEEE 754 float32 array (Length = Dims * 4)      |
+-----------------------+--------+-------------------------------------------------+
```

## 5. Type Tags

Type tags identify the data type of a column.

```go
const (
    TypeINT32   = 0x01
    TypeINT64   = 0x02
    TypeFLOAT32 = 0x03
    TypeVARCHAR = 0x04
    TypeVECTOR  = 0x05
    TypeNULL    = 0x06
    TypeBOOL    = 0x07
)
```
## 6. Stability & Invariants

As of Phase 1 completion, the storage format is **FROZEN**. Any changes to the following structures require a major version bump in the Page Header.

### 6.1 Core Invariants
- **Deterministic Serialization**: Identical page states MUST produce identical byte-for-byte output. No uninitialized memory leaks.
- **Stable RID**: The Slot Index for a tuple MUST never change once allocated, even during page compaction.
- **Checksum Integrity**: No page is valid unless `CRC32C(page with checksum=0) == storedChecksum`.
- **Magic Verification**: Every valid page MUST start with `HYDB`.

### 6.2 Implementation Constraints
- **LSB-First Bitmap**: Null bitmaps always use Least Significant Bit first bit-ordering.
- **Fixed-Size Header**: The 64-byte header is immutable.
- **Boundary Polarity**: `LowerFreeSpaceBound` only moves up (higher addresses); `UpperFreeSpaceBound` only moves down (lower addresses).

### 6.3 Regression Testing
The `internal/storage/page/testdata/pages/` directory contains golden hex snapshots of canonical page states. Any change that alters these bytes without a corresponding spec change is a regression.
