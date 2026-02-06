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
- [ ] Error map structure: `{message, code, tags}` (HTTP errors also include `headers`, `body`)
- [ ] All error types (OFFICIAL 17 tags): AuthError, ConnectionError, ConnectionFailedError, HttpError, IndexError, KeyError, OperationError, ParallelNestingError, RecursionError, ResourceLimitError, ResponseTypeError, SystemError, TimeoutError, TypeError, UnhandledBranchError, ValueError, ZeroDivisionError
- [ ] **ConnectionFailedError vs ConnectionError distinction**: ConnectionFailedError = connection never established (service down, DNS fail, connection refused); ConnectionError = connection broke mid-transfer. Critical for localhost dev.
- [ ] Multi-tag error support (e.g., HttpError + status-specific tags)
- [ ] Retry with exponential backoff (initial_delay, max_delay, multiplier)
- [ ] Custom retry predicates (subworkflow-based)
- [ ] Retry re-executes ENTIRE try block, not just the failed step

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

#### gRPC API (Workflows Service — from googleapis proto)
- [ ] `ListWorkflows` — List workflows in project/location
- [ ] `GetWorkflow` — Get workflow details
- [ ] `CreateWorkflow` — Create workflow (returns Operation)
- [ ] `DeleteWorkflow` — Delete workflow (returns Operation)
- [ ] `UpdateWorkflow` — Update workflow (returns Operation)
- [ ] `ListWorkflowRevisions` — List workflow revision history

#### gRPC API (Executions Service — from googleapis proto)
- [ ] `ListExecutions` — List executions for a workflow
- [ ] `CreateExecution` — Start a new execution
- [ ] `GetExecution` — Get execution details
- [ ] `CancelExecution` — Cancel a running execution

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

#### Developer Experience (Drop-in Replacement)
- [ ] **Watched workflows directory**: `gcw-emulator --workflows-dir=./workflows --port=8080` — emulator watches a directory for `.yaml`/`.json` workflow files. Each file is parsed and "deployed" as a workflow. The directory can be the actual source directory or a dedicated watched folder.
- [ ] **Hot-reload on file change**: When workflow files in the watched directory are added, modified, or deleted, the emulator automatically re-deploys the affected workflow. File name (without extension) becomes the workflow ID. Workflow ID rules: lowercase letters, digits, hyphens, underscores; must start with a letter; max 128 chars. Invalid filenames are skipped with a warning.
- [ ] **API-only mode**: If `--workflows-dir` is not provided, emulator starts with zero workflows and only accepts deployments via the Workflows CRUD API. Useful for programmatic test setups via the Go test helper.
- [ ] **In-flight execution isolation**: When a workflow file changes, currently running executions continue using the workflow definition they started with. Only NEW executions use the updated definition. This matches real GCW revision semantics.
- [ ] **Workflow steps invoke localhost endpoints**: When the workflow engine encounters `call: http.*` steps, it makes real HTTP calls to localhost services. Developers run their services locally (e.g., on different ports) and the workflow orchestrates them, just like real GCW orchestrates Cloud Run / Cloud Functions.
- [ ] **Trigger execution via API**: `POST /v1/{parent}/executions` starts the loaded workflow — same as real GCW Executions API
- [ ] `WORKFLOWS_EMULATOR_HOST` env var support (follows Google convention: `PUBSUB_EMULATOR_HOST`, `FIRESTORE_EMULATOR_HOST`, etc.)
- [ ] REST API paths match real GCP: `http://localhost:{port}/v1/projects/{project}/locations/{location}/workflows/...`
- [ ] Go test helper package (`pkg/testutil/`): `emulator.New()`, `em.Start()`, `em.Stop()`, `em.DeployWorkflow()`, `em.ExecuteWorkflow()`
- [ ] CLI flags: `--port`, `--project`, `--location`, `--workflows-dir` for emulator binary
- [ ] Single binary / `go install` distribution
- [ ] Docker image
- [ ] Clean separation between transport (HTTP handlers) and business logic (service layer) to enable future gRPC transport
- [ ] No authentication required — emulator accepts all requests without credentials

#### Web UI (Observability & CRUD)
- [ ] **Dashboard**: Overview of deployed workflows and recent executions
- [ ] **Workflow list**: View all deployed workflows with status, source file, last modified
- [ ] **Workflow detail**: View workflow source (YAML), subworkflows, step structure
- [ ] **Execution list**: View executions with state (ACTIVE, SUCCEEDED, FAILED, CANCELLED), start time, duration
- [ ] **Execution detail**: View execution result/error, input arguments, current step (if active)
- [ ] **CRUD operations**: Create/trigger executions, cancel running executions from the UI
- [ ] **Built with Go templates** — server-rendered HTML, no separate frontend build step. Reference: `github.com/Maor.Bril/clauder` for Go + templates pattern.
- [ ] **Served on same port** as the API (e.g., `http://localhost:8080/ui/`) or configurable `--ui-port`

### Out of Scope

- **Google Cloud Connectors (`googleapis.*`)** — These require actual GCP service backends. The emulator will handle HTTP calls but not connector-specific semantics. Users can mock connectors via HTTP interceptors.
- **IAM / Authentication enforcement** — The emulator accepts all requests without auth checks.
- **Billing / Quotas enforcement** — No billing simulation.
- **Multi-region / Regional deployment semantics** — Single local instance.
- **Full Cloud Console UI clone** — We build a lightweight observability/CRUD web UI, not a full GCP Console replica.
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
- **samber/ro** (https://ro.samber.dev) library provides reactive programming primitives in Go (observables, operators, subjects, subscriptions) — REQUIRED for building workflow execution primitives: parallel branch execution (observable per branch, Merge to combine), callback await (BehaviorSubject for callback events), execution state management (BehaviorSubject for state transitions), step pipeline (operators for step processing)
- **Fiber** (gofiber.io) for the HTTP server exposing the REST API
- **github.com/Maor.Bril/clauder** — reference for Go + HTML templates web UI pattern (user-provided example)

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
| ~~samber/ro~~ dropped | User decided plain Go goroutines + sync is fine. Common OSS libs OK. | Dropped |
| Custom expression parser | GCW expression language is unique; no existing Go parser available | — Pending |
| Emulate API surface, not internals | Match behavior, not implementation — simplifies architecture | — Pending |
| REST + gRPC dual transport | Go client libraries use gRPC. Must support both REST (curl, non-Go) and gRPC (native Go/Python/Java clients). Proto definitions are small: 6 RPCs for Workflows, 4 for Executions. Use googleapis proto files directly. | — Pending |
| Follow `*_EMULATOR_HOST` convention | All GCP emulators use `{SERVICE}_EMULATOR_HOST` env var. We use `WORKFLOWS_EMULATOR_HOST` for consistency. | — Pending |
| Go test helper as primary DX | Most Go developers will use `emulator.New()` in tests, not raw HTTP. The test helper calls the service layer directly, no HTTP overhead. | — Pending |
| Watched workflows directory (replaces single-file) | Instead of `--workflow=file.yaml`, the emulator watches an entire directory (`--workflows-dir=./workflows`). Each YAML/JSON file = one workflow. Files are hot-reloaded on change. This mirrors how Terraform manages workflow source as files on disk. "Deploying" = saving a file. | Decided |
| In-flight execution isolation on redeploy | When a workflow file changes, running executions continue with their original definition. Only new executions pick up the change. Matches real GCW revision semantics where each execution is pinned to a revision. | Decided |
| HTTP call steps hit localhost | Workflow `http.*` call steps make real HTTP requests to localhost endpoints. Developers run their services locally and the emulator orchestrates them, matching how real GCW calls Cloud Run/Functions. This is the core integration testing value prop. | Decided |
| Execution triggered via API only | Workflow execution starts via `POST /v1/{parent}/executions` — same as real GCW. No auto-execute on startup. The emulator is a server waiting for API calls. | Decided |
| Web UI for observability & CRUD | Lightweight server-rendered web UI (Go templates, no separate frontend). Shows deployed workflows, executions, results/errors. Allows triggering and cancelling executions. Served alongside the API. Reference: `github.com/Maor.Bril/clauder` for Go + templates pattern. | Decided |
| Go templates for Web UI (no JS framework) | Keep everything in Go. Server-rendered HTML with Go `html/template`. No React/Vue/npm build step. Single binary includes all templates. Matches the "keep as much Go code as possible" constraint. | Decided |

---
*Last updated: 2026-02-05 — added: watched directory (replaces single-file), hot-reload with in-flight isolation, web UI with Go templates*
