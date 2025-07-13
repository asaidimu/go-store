# [2.0.0](https://github.com/asaidimu/go-store/compare/v1.0.0...v2.0.0) (2025-07-13)


* refactor(store)!: remove Document type alias, use map[string]any directly ([4a7e4a3](https://github.com/asaidimu/go-store/commit/4a7e4a304067cdeba8c78602e4c60d246bb94ade))


### BREAKING CHANGES

* Users must update their code to use `map[string]any` instead of `store.Document` when interacting with the store for document insertions, updates, and retrievals. For instance, `store.Document{"key": "value"}` should now be `map[string]any{"key": "value"}`.

# 1.0.0 (2025-07-10)


### Features

* **store:** implement core in-memory document database ([474cb0a](https://github.com/asaidimu/go-store/commit/474cb0a5b78b7a57277d5f7856cb101663421d64))
