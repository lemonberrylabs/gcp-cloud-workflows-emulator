# Localhost Orchestration

This is the core value proposition of the emulator: your workflow's `http.*` steps make **real HTTP requests** to your local services, exactly as Google Cloud Workflows calls Cloud Run or Cloud Functions in production.

## The pattern

In production, your GCW workflow calls Cloud Run services:

```yaml
main:
  steps:
    - validate:
        call: http.post
        args:
          url: https://validate-service-xyz.run.app/validate
          body: ${args}
          auth:
            type: OIDC
        result: validation
    - process:
        call: http.post
        args:
          url: https://process-service-xyz.run.app/process
          body: ${validation.body}
          auth:
            type: OIDC
        result: result
    - done:
        return: ${result.body}
```

For local development, point those URLs at localhost:

```yaml
main:
  steps:
    - validate:
        call: http.post
        args:
          url: http://localhost:9090/validate
          body: ${args}
        result: validation
    - process:
        call: http.post
        args:
          url: http://localhost:9091/process
          body: ${validation.body}
        result: result
    - done:
        return: ${result.body}
```

The `auth` config is accepted by the emulator but not enforced -- no credentials needed locally.

## Making URLs configurable

Use `sys.get_env` to switch between local and production URLs:

```yaml
main:
  params: [args]
  steps:
    - get_config:
        assign:
          - validate_url: ${sys.get_env("VALIDATE_SERVICE_URL")}
          - process_url: ${sys.get_env("PROCESS_SERVICE_URL")}
    - validate:
        call: http.post
        args:
          url: ${validate_url + "/validate"}
          body: ${args}
        result: validation
    - process:
        call: http.post
        args:
          url: ${process_url + "/process"}
          body: ${validation.body}
        result: result
    - done:
        return: ${result.body}
```

## Testing the full flow

1. Start your services locally on different ports
2. Start the emulator: `emulator --workflows-dir=./workflows`
3. Trigger an execution via the API
4. The emulator calls your services in order, passing data between them
5. Check the execution result

This verifies that:
- Your workflow syntax is correct
- The service orchestration logic works
- Data flows correctly between services
- Error handling catches failures properly

## Error behavior

### Service not running

When a local service is not running, the emulator raises a `ConnectionFailedError`:

```yaml
- safe_call:
    try:
      call: http.get
      args:
        url: http://localhost:9090/health
      result: response
    except:
      as: e
      steps:
        - check:
            switch:
              - condition: ${"ConnectionFailedError" in e.tags}
                return: "Service is not running on port 9090"
```

### Service returns an error

When a service returns a non-2xx status code, the emulator raises an `HttpError`:

```yaml
- safe_call:
    try:
      call: http.post
      args:
        url: http://localhost:9090/api/orders
        body: ${order_data}
      result: response
    except:
      as: e
      steps:
        - check:
            switch:
              - condition: ${"HttpError" in e.tags and e.code == 400}
                return:
                  error: "Bad request"
                  details: ${e.body}
              - condition: ${"HttpError" in e.tags and e.code == 404}
                return:
                  error: "Not found"
```

The error includes the response `body` and `headers`, so you can inspect what the service returned.

### Automatic retry

Configure retry to handle transient failures:

```yaml
- fetch_with_retry:
    try:
      call: http.get
      args:
        url: http://localhost:9090/api/data
      result: response
    retry:
      predicate: ${http.default_retry}
      max_retries: 3
      backoff:
        initial_delay: 1
        max_delay: 10
        multiplier: 2
```

This retries on 429, 502, 503, 504, ConnectionError, and TimeoutError.

## Parallel service orchestration

Call multiple services concurrently:

```yaml
main:
  params: [args]
  steps:
    - init:
        assign:
          - results: {}
    - fetch_all:
        parallel:
          shared: [results]
          branches:
            - get_user:
                steps:
                  - fetch:
                      call: http.get
                      args:
                        url: '${"http://localhost:9090/users/" + string(args.user_id)}'
                      result: user
                  - save:
                      assign:
                        - results: ${map.merge(results, {"user": user.body})}
            - get_orders:
                steps:
                  - fetch:
                      call: http.get
                      args:
                        url: '${"http://localhost:9091/orders?user=" + string(args.user_id)}'
                      result: orders
                  - save:
                      assign:
                        - results: ${map.merge(results, {"orders": orders.body})}
    - done:
        return: ${results}
```

## Tips

- Use different ports per service to avoid conflicts
- Start all services before running the workflow, or use try/except to handle services that may not be ready
- Run services in Docker Compose alongside the emulator for reproducible setups (see [Docker](./docker.md))
- JSON responses from your services are auto-parsed by the emulator -- return `application/json` from your services and access the data directly as maps/lists
