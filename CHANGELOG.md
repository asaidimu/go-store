# [3.2.0](https://github.com/asaidimu/go-store/compare/v3.1.0...v3.2.0) (2025-07-19)


### Features

* **docs:** introduce new Cursor API documentation and enhance developer experience ([6a873ad](https://github.com/asaidimu/go-store/commit/6a873adc2152ea012041a3b82f9843288b0b57b2))

# [3.1.0](https://github.com/asaidimu/go-store/compare/v3.0.0...v3.1.0) (2025-07-17)


### Features

* **store:** Introduce bidirectional cursor and refactor storage engine ([5d1ae75](https://github.com/asaidimu/go-store/commit/5d1ae75dd340af8fd6c8fada8779bcdb50e9d74a))

# [3.0.0](https://github.com/asaidimu/go-store/compare/v2.0.1...v3.0.0) (2025-07-13)


* build(module)!: update Go module to v3 ([258e66c](https://github.com/asaidimu/go-store/commit/258e66c0e112e84dc1ace4d814328d414f03d5e7))


### BREAKING CHANGES

* The Go module path has been updated from
'github.com/asaidimu/go-store/v2' to 'github.com/asaidimu/go-store/v3'.
Consumers must update their import statements to reflect the new major version.

## [2.0.1](https://github.com/asaidimu/go-store/compare/v2.0.0...v2.0.1) (2025-07-13)


### Bug Fixes

* **mod:** ensure v2 module is indexed by Go proxy ([c3f183d](https://github.com/asaidimu/go-store/commit/c3f183d1d467ace98da2c3089c17000aba0719da))

# [2.0.0](https://github.com/asaidimu/go-store/compare/v1.0.0...v2.0.0) (2025-07-13)


* refactor(store)!: remove Document type alias, use map[string]any directly ([4a7e4a3](https://github.com/asaidimu/go-store/commit/4a7e4a304067cdeba8c78602e4c60d246bb94ade))


### BREAKING CHANGES

* Users must update their code to use `map[string]any` instead of `store.Document` when interacting with the store for document insertions, updates, and retrievals. For instance, `store.Document{"key": "value"}` should now be `map[string]any{"key": "value"}`.

# 1.0.0 (2025-07-10)


### Features

* **store:** implement core in-memory document database ([474cb0a](https://github.com/asaidimu/go-store/v3/commit/474cb0a5b78b7a57277d5f7856cb101663421d64))
