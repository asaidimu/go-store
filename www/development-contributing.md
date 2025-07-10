---
outline: "deep"
lastUpdated: true
editLink: true
prev: true
next: true
title: "Development & Contributing"
description: "Development & Contributing documentation and guidance"
---
# Development & Contributing

## Development & Contributing

We welcome contributions to `go-store`! Here's how to get started.

### Development Setup

1.  **Clone the repository**:
    ```bash
git clone https://github.com/asaidimu/go-store.git
cd go-store
    ```
2.  **Build the project**:
    ```bash
go build -v ./...
    ```
    This command compiles all packages in the current module.

### Scripts

The project includes a simple `Makefile` for common development tasks:

*   `make build`: Compiles the entire project. Equivalent to `go build -v ./...`.
*   `make test`: Runs all unit tests with verbose output. Equivalent to `go test -v ./...`.
*   `make clean`: Removes generated executable files.

### Testing

To run the test suite and ensure everything is working correctly, execute:

```bash
go test -v ./...
```

To run performance benchmarks, which are crucial for an in-memory store:

```bash
go test -bench=. -benchmem
```

The tests cover basic CRUD operations, indexing, concurrency, and error handling.

### Contributing Guidelines

Contributions are what make the open-source community an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1.  **Fork the repository**: Click the "Fork" button at the top right of this page.
2.  **Create your feature branch**:
    ```bash
git checkout -b feature/amazing-feature
    ```
3.  **Commit your changes**: We follow Conventional Commits (see below).
    ```bash
git commit -m 'feat: Add some amazing feature'
    ```
4.  **Push to the branch**:
    ```bash
git push origin feature/amazing-feature
    ```
5.  **Open a Pull Request**: Describe your changes clearly and link to any relevant issues.

#### Commit Message Format

This project follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for automated semantic versioning. Please adhere to this format for your commit messages:

*   `feat`: A new feature (corresponds to `MINOR` in SemVer)
*   `fix`: A bug fix (corresponds to `PATCH` in SemVer)
*   `build`: Changes that affect the build system or external dependencies
*   `ci`: Changes to CI configuration files and scripts
*   `docs`: Documentation only changes
*   `perf`: A code change that improves performance
*   `refactor`: A code change that neither fixes a bug nor adds a feature
*   `style`: Changes that do not affect the meaning of the code (white-space, formatting, missing semicolons, etc.)
*   `test`: Adding missing tests or correcting existing tests

For breaking changes, append `!` after the type/scope, or include `BREAKING CHANGE:` in the footer:

*   `feat!: introduce breaking API change`
*   `fix(auth)!: correct authentication flow`

### Issue Reporting

If you find a bug or have a feature request, please open an issue on the [GitHub Issues page](https://github.com/asaidimu/go-store/issues). When reporting a bug, please include:

*   A clear and concise description of the problem.
*   Steps to reproduce the behavior.
*   Expected behavior.
*   Actual behavior.
*   Your Go version and OS.

