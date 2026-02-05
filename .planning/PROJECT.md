# GCP Cloud Workflows Emulator

## What This Is

A full-featured, open-source Google Cloud Workflows emulator written in Go that runs locally. It exposes the exact same REST API as the real Google Cloud Workflows and Workflow Executions services, enabling developers to run integration tests against their workflow definitions without deploying to GCP.

## Core Value

**Exact semantic fidelity to Google Cloud Workflows** — every workflow that runs correctly on GCP must produce the same result in the emulator, and every error the emulator raises must match GCP's behavior.

## Requirements

### Validated

(None yet — ship to validate)

### Active

#### Workflow Engine Core
- [ ] YAML/JSON workflow parser supporting full GCW syntax (steps, subworkflows, params)
- [ ] Expression engine supporting `${}` expressions with all operators (+, -, *, /, %, //, ==, !=, <, >, <=, >=, and, or, not, in)
- [ ] Type system: int (64-bit), double (64-bit), string, bool, null, list, map, bytes
- [ ] Variable scoping: workflow scope, subworkflow scope, for-loop scope, parallel branch scope
- [ ] Step execution engine with sequential execution, `next` jumps, `end`, `break`, `continue`

#### Step Types
- [ ] `assign` — variable assignment (up to 50 assignments per step)
- [ ] `call` — HTTP calls, standard library calls, connector calls, subworkflow calls
- [ ] `switch` — conditional branching (up to 50 conditions)
- [ ] `for` — iteration over lists, maps (via keys()), and ranges (inclusive)
- [ ] `parallel` — concurrent branches (up to 10) and parallel for loops
- [ ] `try/except/retry` — error handling with configurable retry policies and exponential backoff
- [ ] `raise` — custom error throwing (string or map)
- [ ] `return` — workflow/subworkflow termination with return value
- [ ] `next` — flow control jumps (step names, `end`, `break`, `continue`)
- [ ] `steps` — nested step grouping

#### Standard Library
- [ ] Expression helpers: `default()`, `keys()`, `len()`, `type()`, `int()`, `double()`, `string()`, `bool()`
- [ ] `http.*` — HTTP methods (get, post, put, patch, delete, request) with auth, headers, query, timeout
- [ ] `sys.*` — `get_env()`, `log()`, `now()`, `sleep()`, `sleep_until()`
- [ ] `events.*` — `create_callback_endpoint()`, `await_callback()`
- [ ] `text.*` — string operations (find, replace, split, substring, regex, url encode/decode, case)
- [ ] `json.*` — `decode()`, `encode()`, `encode_to_string()`
- [ ] `base64.*` — `decode()`, `encode()`
- [ ] `math.*` — `abs()`, `floor()`, `max()`, `min()`
- [ ] `list.*` — `concat()`, `prepend()`
- [ ] `map.*` — `get()`, `delete()`, `merge()`, `merge_nested()`
- [ ] `time.*` — `format()`, `parse()`
- [ ] `hash.*` — `compute_checksum()`, `compute_hmac()`
- [ ] `uuid.*` — `generate()`
- [ ] `retry.*` — `always`, `never`, `default_backoff`
- [ ] Built-in retry policies: `http.default_retry`, `http.default_retry_non_idempotent`, predicates

#### Parallel Execution
- [ ] Parallel branches with `shared` variable declarations and atomic read/write
- [ ] Parallel for loops
- [ ] `concurrency_limit` enforcement (max 20 concurrent)
- [ ] Exception policies: `unhandled` (abort on first) and `continueAll` (collect up to 100)
- [ ] Parallel nesting depth limit: 2

#### Error Handling
- [ ] Error map structure: `{message, code, tags}`
- [ ] All error types: HttpError, ConnectionError, TimeoutError, TypeError, ValueError, KeyError, IndexError, ZeroDivisionError, RecursionError, ResourceLimitError, MemoryLimitExceededError, ResultSizeLimitExceededError, AuthenticationError
- [ ] Multi-tag error support
- [ ] Retry with exponential backoff (initial_delay, max_delay, multiplier)
- [ ] Custom retry predicates (subworkflow-based)

#### REST API (Workflows)
- [ ] `POST /v1/{parent}/workflows` — Create workflow (returns Operation)
- [ ] `GET /v1/{name}` — Get workflow
- [ ] `GET /v1/{parent}/workflows` — List workflows (pagination, filter, orderBy)
- [ ] `PATCH /v1/{name}` — Update workflow (returns Operation)
- [ ] `DELETE /v1/{name}` — Delete workflow (returns Operation)

#### REST API (Executions)
- [ ] `POST /v1/{parent}/executions` — Create execution (start workflow)
- [ ] `GET /v1/{name}` — Get execution status/result
- [ ] `GET /v1/{parent}/executions` — List executions (pagination, filter, orderBy)
- [ ] `POST /v1/{name}:cancel` — Cancel execution

#### REST API (Callbacks)
- [ ] `GET /v1/{parent}/callbacks` — List callbacks
- [ ] `POST {callback_url}` — Send callback to waiting execution

#### Limits Enforcement
- [ ] 50 assignments per assign step
- [ ] 50 conditions per switch
- [ ] 20 max call stack depth
- [ ] 10 branches per parallel step
- [ ] 2 parallel nesting depth
- [ ] 20 max concurrent branches/iterations
- [ ] 128 KB source code size
- [ ] 512 KB variable memory
- [ ] 256 KB max string length
- [ ] 400 character max expression length
- [ ] 100,000 max steps per execution

#### Developer Experience
- [ ] Single binary / `go install` distribution
- [ ] Docker image
- [ ] Go test helper: `emulator.Start()` / `emulator.Stop()` for integration tests
- [ ] Environment variable configuration for emulator settings

### Out of Scope

- **Google Cloud Connectors (`googleapis.*`)** — These require actual GCP service backends. The emulator will handle HTTP calls but not connector-specific semantics. Users can mock connectors via HTTP interceptors.
- **IAM / Authentication enforcement** — The emulator accepts all requests without auth checks.
- **Billing / Quotas enforcement** — No billing simulation.
- **Multi-region / Regional deployment semantics** — Single local instance.
- **Cloud Console UI** — CLI/API only.
- **Eventarc / Pub/Sub triggers** — Executions are triggered via REST API only.
- **Long-running Operation polling for workflow CRUD** — Create/Update/Delete return immediately in the emulator.
- **CMEK (Customer-Managed Encryption Keys)** — Not relevant for local emulator.
- **Workflow execution history / step entries API** — Focus on execution results, not step-by-step audit trail.

## Context

- **Google Cloud Workflows** is a fully managed serverless orchestration platform for executing services in defined order
- Workflows are defined in YAML or JSON with a custom expression language (`${}`)
- The execution model is sequential by default with explicit `next` jumps and parallel branches
- Two REST APIs: Workflows API (CRUD for workflow definitions) and Executions API (run/monitor executions)
- Existing emulators: None with full feature coverage — this fills a real gap in the GCP ecosystem
- Similar projects: Firebase Emulator Suite (local emulators for Firestore, Auth, etc.) — this follows the same philosophy
- **samber/ro** library provides reactive programming primitives in Go (observables, operators, subscriptions) — may be useful for building the parallel execution and event callback systems
- **Fiber** (gofiber.io) for the HTTP server exposing the REST API

## Constraints

- **Language**: Go (1.22+) — chosen for performance, single-binary distribution, and ecosystem fit
- **API Compatibility**: Must match Google Cloud Workflows REST API request/response formats exactly
- **Open Source**: MIT or Apache 2.0 license — designed for community use
- **No GCP Dependencies**: Must run fully offline with zero GCP credentials required
- **Expression Parser**: Need to build or find a parser for the `${}` expression language (likely a custom parser since it's GCP-specific)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go as implementation language | Performance, single binary, ecosystem fit for cloud tooling | — Pending |
| Fiber for HTTP server | Fast, Express-like API, good for REST endpoint implementation | — Pending |
| samber/ro for reactive primitives | Observable patterns useful for parallel execution and callbacks | — Pending |
| Custom expression parser | GCW expression language is unique; no existing Go parser available | — Pending |
| Emulate API surface, not internals | Match behavior, not implementation — simplifies architecture | — Pending |

---
*Last updated: 2026-02-05 after initialization*
