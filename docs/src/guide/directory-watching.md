# Directory Watching

When started with `--workflows-dir`, the emulator watches a directory for workflow files and hot-reloads them on changes.

## How it works

1. On startup, the emulator reads all `.yaml` and `.json` files in the directory
2. Each file is parsed and deployed as a workflow
3. The filename (without extension) becomes the workflow ID
4. The directory is watched for changes -- add, modify, or delete files and the emulator responds automatically

## Workflow ID rules

The filename (minus extension) must be a valid workflow ID:

- Lowercase letters, digits, hyphens, and underscores only
- Must start with a letter
- Maximum 128 characters

### Special cases

| Filename | Behavior |
|----------|----------|
| `my-workflow.yaml` | Deployed as `my-workflow` |
| `MyWorkflow.yaml` | Lowercased to `myworkflow` (with log warning) |
| `my.workflow.yaml` | Skipped -- dots produce invalid ID `my.workflow` |
| `123-start.yaml` | Skipped -- starts with a digit |
| `README.md` | Ignored -- not `.yaml` or `.json` |

## In-flight execution isolation

When a workflow file changes while an execution is running:

- The running execution continues using the workflow definition it started with
- Only new executions use the updated definition

This matches the real GCW behavior where each execution is pinned to a specific workflow revision.

## Example

```bash
# Start with a workflows directory
emulator --workflows-dir=./workflows

# In another terminal, add a new workflow
cat > workflows/process-order.yaml << 'EOF'
main:
  params: [args]
  steps:
    - validate:
        call: http.post
        args:
          url: http://localhost:9090/validate
          body:
            order_id: ${args.order_id}
        result: validation
    - process:
        call: http.post
        args:
          url: http://localhost:9091/process
          body:
            order_id: ${args.order_id}
            validated: ${validation.body.valid}
        result: result
    - done:
        return: ${result.body}
EOF

# The emulator detects the new file and deploys it immediately
# Now you can execute it via the API
```
