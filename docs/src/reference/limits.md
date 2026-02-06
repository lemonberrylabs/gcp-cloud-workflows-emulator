# Limits

The emulator enforces the same limits as Google Cloud Workflows.

## Execution limits

| Limit | Value | Error on exceed |
|-------|-------|-----------------|
| Assignments per `assign` step | 50 | ResourceLimitError |
| Conditions per `switch` step | 50 | ResourceLimitError |
| Call stack depth (subworkflow nesting) | 20 | RecursionError |
| Steps per execution | 100,000 | ResourceLimitError |
| Expression length | 400 characters | Validation error |

## Parallel execution limits

| Limit | Value | Error on exceed |
|-------|-------|-----------------|
| Branches per `parallel` step | 10 | ResourceLimitError |
| Max concurrent branches/iterations | 20 | ResourceLimitError |
| Parallel nesting depth | 2 | ParallelNestingError |
| Unhandled exceptions per execution (`continueAll`) | 100 | -- |

## Size limits

| Limit | Value |
|-------|-------|
| Workflow source code | 128 KB |
| Variable memory (all variables, arguments, events) | 512 KB |
| Maximum string length | 256 KB |
| HTTP response size | 2 MB |
| Execution argument size | 32 KB |

## HTTP limits

| Limit | Value |
|-------|-------|
| HTTP request timeout | 1800 seconds (30 minutes) |
| Execution duration | 1 year |

## What happens when a limit is exceeded

- **Assignment/switch/branch limits**: Deployment or validation error before execution starts
- **Call stack depth**: `RecursionError` at runtime when depth 20 is exceeded
- **Step count**: `ResourceLimitError` after 100,000 steps in a single execution
- **Parallel nesting**: `ParallelNestingError` when nesting depth exceeds 2
- **Memory/size limits**: `ResourceLimitError` when variable memory or result size exceeds the cap
- **HTTP timeout**: `TimeoutError` when a request exceeds the configured timeout

## Tips for staying within limits

- Assign unused variables to `null` to free memory
- Only store essential portions of large API responses
- Use `list.concat()` judiciously in parallel for loops (each call copies the list)
- Break large workflows into subworkflows for readability, but watch the call stack depth
- Use `concurrency_limit` in parallel steps to control resource usage
