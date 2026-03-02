# Contributing

Contributions are welcome! See the full [CONTRIBUTING.md](https://github.com/lemonberrylabs/gcw-emulator/blob/main/CONTRIBUTING.md) in the repository root for complete guidelines.

## Test Requirements

**Every bug fix and feature must include tests.** PRs without adequate test coverage will not be merged.

- **Bug fixes**: Add a test that reproduces the bug and verifies the fix.
- **New features**: Add both unit tests (`pkg/`) and integration tests (`test/integration/`) that exercise the feature end-to-end.
- **Refactors**: Ensure existing tests still pass. If you're changing behavior, update the tests to match.

## Quick Start

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gcw-emulator.git
   cd gcw-emulator
   ```
3. Create a feature branch:
   ```bash
   git checkout -b feature/my-change
   ```

## Building

```bash
go build ./cmd/gcw-emulator
```

## Running Tests

### Unit tests

```bash
go test ./pkg/...
```

### Integration tests

Start the emulator:

```bash
go run ./cmd/gcw-emulator
```

In another terminal:

```bash
cd test/integration
WORKFLOWS_EMULATOR_HOST=http://localhost:8787 go test -v ./...
```

The integration test suite has 221+ tests covering all step types, standard library functions, error handling, parallel execution, the REST API, the Web UI, and edge cases. See `test/integration/helpers_test.go` for shared test utilities.

## Submitting Changes

1. Ensure `go vet ./...` passes
2. Ensure all unit tests pass: `go test ./pkg/...`
3. Ensure integration tests pass (see above)
4. Write a clear commit message describing the change
5. Submit a pull request against the `main` branch

## Reporting Issues

Use [GitHub Issues](https://github.com/lemonberrylabs/gcw-emulator/issues) to report bugs or request features. Include:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Workflow YAML (if applicable)
- Emulator version or commit hash
