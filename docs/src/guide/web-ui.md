# Web UI

The emulator includes a built-in web UI served on the same port as the API.

## Accessing the UI

Browse to `http://localhost:{port}/ui/` (the root path `/` also redirects to the UI).

## Pages

### Dashboard (`/ui`)

Overview showing:
- All deployed workflows
- Recent executions across all workflows
- Counts by state (active, succeeded, failed, cancelled)

### Workflow List (`/ui/workflows`)

All deployed workflows with:
- Workflow ID and state
- Total execution count
- Active execution count
- Last update time

### Workflow Detail (`/ui/workflows/{id}`)

Single workflow showing:
- Full YAML source code
- Execution history for this workflow

### Execution List (`/ui/executions`)

All executions across all workflows, sorted by start time.

Per-workflow execution list at `/ui/workflows/{id}/executions`.

### Execution Detail (`/ui/executions/{workflowId}/{execId}`)

Single execution showing:
- State (ACTIVE, SUCCEEDED, FAILED, CANCELLED)
- Start and end times, duration
- Input arguments
- Result (on success) or error details (on failure)
