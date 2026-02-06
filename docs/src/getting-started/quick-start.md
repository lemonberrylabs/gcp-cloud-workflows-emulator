# Quick Start

This guide gets you from zero to running a workflow in under 2 minutes.

## 1. Create a workflow file

Create a directory for your workflows and add a file:

```bash
mkdir workflows
```

Create `workflows/greet.yaml`:

```yaml
main:
  params: [args]
  steps:
    - build_greeting:
        assign:
          - message: '${"Hello, " + args.name + "!"}'
    - done:
        return: ${message}
```

## 2. Start the emulator

```bash
gcw-emulator --workflows-dir=./workflows
```

You should see:

```
GCW Emulator listening on 0.0.0.0:8787
```

## 3. Run the workflow

In another terminal:

```bash
curl -s -X POST \
  http://localhost:8787/v1/projects/my-project/locations/us-central1/workflows/greet/executions \
  -H "Content-Type: application/json" \
  -d '{"argument": "{\"name\": \"Alice\"}"}' | jq .
```

This returns an execution object with a `name` field. Copy the execution name and check the result:

```bash
curl -s http://localhost:8787/v1/projects/my-project/locations/us-central1/workflows/greet/executions/<exec-id> | jq .
```

The response includes:

```json
{
  "state": "SUCCEEDED",
  "result": "\"Hello, Alice!\""
}
```

## 4. Open the Web UI

Browse to [http://localhost:8787/ui/](http://localhost:8787/ui/) to see your workflow and execution in a dashboard.

## 5. Edit and iterate

Edit `workflows/greet.yaml` and save. The emulator detects the change and redeploys the workflow automatically. Run it again to see your changes.

## Next steps

- [CLI & Configuration](../guide/configuration.md) -- all environment variables and flags
- [Integration Testing](../guide/integration-testing.md) -- use the emulator in Go tests
- [Workflow Syntax](../reference/workflow-syntax.md) -- all step types and features
- [Localhost Orchestration](../advanced/localhost-orchestration.md) -- the core integration testing pattern
