# GCP Cloud Workflows Emulator

A local emulator for [Google Cloud Workflows](https://cloud.google.com/workflows) that lets you develop, test, and debug workflows without deploying to GCP.

## What is Google Cloud Workflows?

Google Cloud Workflows is a fully managed serverless orchestration platform that executes services in a defined order. Workflows are defined in YAML or JSON and can call HTTP APIs, Cloud Run services, Cloud Functions, and other Google Cloud services.

## Why use an emulator?

Developing against the real Google Cloud Workflows service means every change requires a deploy-and-test cycle in the cloud. This emulator eliminates that cycle:

- **Iterate locally** -- edit a YAML file, save, and the workflow is redeployed instantly via hot-reload
- **Test offline** -- no GCP account, no credentials, no network required
- **Orchestrate local services** -- your workflow's `http.get/post/...` steps call `localhost` endpoints, just like real GCW calls Cloud Run or Cloud Functions
- **Run integration tests** -- deploy workflows and trigger executions via the same REST API your production code uses
- **Inspect results** -- use the built-in Web UI to view workflows, executions, results, and errors

## Key capabilities

| Feature | Description |
|---------|-------------|
| **Full REST API** | Same endpoints and request/response formats as the real Workflows and Executions APIs |
| **gRPC API** | Native gRPC support for Go, Java, and Python client libraries |
| **All step types** | assign, call, switch, for, parallel, try/except/retry, raise, return, next, steps |
| **Expression engine** | Complete `${}` expression support with all operators |
| **Standard library** | http, sys, text, json, base64, math, list, map, time, uuid, events, retry |
| **Parallel execution** | Branches, parallel for loops, shared variables, concurrency limits, exception policies |
| **Error handling** | All 17 GCW error tags, exponential backoff, custom retry predicates |
| **Directory watching** | Point at a directory of YAML/JSON files and get hot-reload on save |
| **Web UI** | Built-in dashboard at `/ui/` for inspecting workflows and executions |

## How it works

```
 You edit YAML files       Emulator watches & deploys       Your tests or curl
 +-----------------+       +-----------------------+       +------------------+
 | workflows/      |  ---> | gcw-emulator          |  <--- | POST /executions |
 |   order.yaml    |       |   :8787 (REST)        |       | GET /executions  |
 |   notify.yaml   |       |   :8788 (gRPC)        |       |                  |
 +-----------------+       +-----------+-----------+       +------------------+
                                       |
                                       | http.get/post steps
                                       v
                           +-----------------------+
                           | Your local services   |
                           |   :9090, :9091, ...   |
                           +-----------------------+
```

1. Start the emulator pointing at a directory of workflow YAML files
2. The emulator watches for file changes and hot-reloads workflows
3. Trigger executions via the REST API, gRPC API, or the Web UI
4. Workflow steps that call `http.*` make real HTTP requests to your local services
5. Inspect results in the Web UI or via the GET execution endpoint

## Next steps

- [Installation](./getting-started/installation.md) -- install the emulator
- [Quick Start](./getting-started/quick-start.md) -- deploy and run your first workflow in under 5 minutes
