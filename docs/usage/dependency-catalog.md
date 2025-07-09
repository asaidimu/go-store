# Dependency Catalog

## External Dependencies

### github.com/google/btree
- **Purpose**: Provides a B-tree implementation used for efficient indexing and sorted data storage within `fieldIndex`.
- **Installation**: `go get github.com/google/btree`
- **Version Compatibility**: `v1.1.2`

### github.com/google/uuid
- **Purpose**: Used for generating universally unique identifiers (UUIDs) for new documents.
- **Installation**: `go get github.com/google/uuid`
- **Version Compatibility**: `v1.6.0`

## Peer Dependencies

### Go runtime
- **Reason**: Required to compile and run the go-store library and applications built with it.
- **Version Requirements**: `>=1.24.4`



---
*Generated using Gemini AI on 7/9/2025, 11:33:48 PM. Review and refine as needed.*