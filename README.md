# Go HTTP Frameworks Deep Dive

A practical, code-driven guide to learning HTTP frameworks in Go.

This repository explores **net/http**, **Chi**, **Gin**, **Echo**, **Fiber**, and **Mizu** - by solving the same problems in the same order, using real, runnable programs.

Each topic starts with a minimal working server and then explains what actually happens when a request enters the process and a response leaves it. The focus stays on behavior and execution flow rather than surface level APIs.

The intent is understanding. You should be able to explain why something works, not just how to write it.

## Overview

### What this repository provides

This repository is a structured walk through how Go HTTP frameworks behave.

It helps you:

- see how different frameworks structure the same application
- follow a request through routing, middleware, handlers, and response writing
- understand ownership of servers, routers, contexts, and lifecycles
- compare tradeoffs based on execution, not marketing

This repository does not aim to benchmark performance or list features.

The emphasis stays on clarity, mechanics, and long term understanding.

Each section stands on its own. You can read from start to finish or jump directly to a topic you care about.

### Frameworks covered

| Framework | Description                       | Version  | Release date | GitHub                                                               |
| --------- | --------------------------------- | -------- | ------------ | -------------------------------------------------------------------- |
| net/http  | Go standard library HTTP server   | go1.25.5 | 2025-12-02   | [https://github.com/golang/go](https://github.com/golang/go)         |
| Chi       | Router built on net/http          | v5.2.3   | 2025-08-26   | [https://github.com/go-chi/chi](https://github.com/go-chi/chi)       |
| Gin       | HTTP framework with helpers       | v1.11.0  | 2025-09-20   | [https://github.com/gin-gonic/gin](https://github.com/gin-gonic/gin) |
| Echo      | HTTP framework with error returns | v4.14.0  | 2025-12-11   | [https://github.com/labstack/echo](https://github.com/labstack/echo) |
| Fiber     | fasthttp based framework          | v2.52.10 | 2025-11-19   | [https://github.com/gofiber/fiber](https://github.com/gofiber/fiber) |
| Mizu      | net/http based framework          | v0.2.2   | 2025-12-15   | [https://github.com/go-mizu/mizu](https://github.com/go-mizu/mizu)   |

Versions are included for reference. The behaviors discussed here remain stable across minor releases.

### How to use this repository

Each topic lives in its own folder.

Inside each folder you will find:

- a README.md that explains the topic step by step
- a main.go file for each framework
- small examples that you can run directly

You do not need to search across folders to follow the explanation. Every README includes the full code it discusses.

A recent Go version is enough to run the examples unless noted otherwise.

### Topics and reading order

You can follow the topics in sequence or jump to any section.

| Folder               | Topic                        | File                                                             |
| -------------------- | ---------------------------- | ---------------------------------------------------------------- |
| 01-hello-world       | First HTTP server            | [01-hello-world/README.md](01-hello-world/README.md)             |
| 02-application       | Application setup            | [02-application/README.md](02-application/README.md)             |
| 03-handler-signature | Handlers and request flow    | [03-handler-signature/README.md](03-handler-signature/README.md) |
| 04-routing           | Paths, methods, and matching | [04-routing/README.md](04-routing/README.md)                     |
| 05-route-groups      | Grouping routes              | [05-route-groups/README.md](05-route-groups/README.md)           |
| 06-middleware-chain  | Middleware order             | [06-middleware-chain/README.md](06-middleware-chain/README.md)   |
| 07-short-circuit     | Early exits                  | [07-short-circuit/README.md](07-short-circuit/README.md)         |
| 08-error-handling    | Errors and panics            | [08-error-handling/README.md](08-error-handling/README.md)       |
| 09-request-input     | Reading headers and bodies   | [09-request-input/README.md](09-request-input/README.md)         |
| 10-response-output   | Writing responses            | [10-response-output/README.md](10-response-output/README.md)     |
| 11-json              | JSON handling                | [11-json/README.md](11-json/README.md)                           |
| 12-path-params       | Path parameters              | [12-path-params/README.md](12-path-params/README.md)             |
| 13-static-files      | Static and embedded files    | [13-static-files/README.md](13-static-files/README.md)           |
| 14-templates         | HTML templates               | [14-templates/README.md](14-templates/README.md)                 |
| 15-forms-upload      | Forms and file uploads       | [15-forms-upload/README.md](15-forms-upload/README.md)           |
| 16-websocket         | WebSockets                   | [16-websocket/README.md](16-websocket/README.md)                 |
| 17-sse               | Server-Sent Events           | [17-sse/README.md](17-sse/README.md)                             |
| 18-context-cancel    | Context and cancellation     | [18-context-cancel/README.md](18-context-cancel/README.md)       |
| 19-shutdown          | Graceful shutdown            | [19-shutdown/README.md](19-shutdown/README.md)                   |
| 20-logging           | Logging basics               | [20-logging/README.md](20-logging/README.md)                     |
| 21-metrics-tracing   | Metrics and tracing          | [21-metrics-tracing/README.md](21-metrics-tracing/README.md)     |
| 22-testing           | Testing handlers             | [22-testing/README.md](22-testing/README.md)                     |
| 23-performance       | Performance considerations   | [23-performance/README.md](23-performance/README.md)             |
| 24-interop           | Working with net/http        | [24-interop/README.md](24-interop/README.md)                     |
| 25-tradeoffs          | Tradeoffs | [25-tradeoffs/README.md](25-tradeoffs/README.md)                   |

### How the examples are written

All examples follow the same discipline:

- every example runs as written
- files stay small and focused
- no hidden helpers or magic layers
- the same structure appears across frameworks

This allows you to compare behavior directly without learning a new mental model for each section.

### What you will learn

By working through this repository, you will understand:

- how routing decisions are made
- how middleware wraps execution
- how request context flows through a server
- how different error models affect control flow
- how shutdown and cleanup are coordinated
- how close each framework stays to net/http

These details become critical as applications grow, change ownership, or need to integrate with other systems.

### Final note

Choosing a framework comes down to responsibility.

Every framework takes some control away from you and gives some structure back. The important part is knowing exactly where that trade is made.

Once you understand those boundaries, switching frameworks becomes a design decision rather than a rewrite.

That understanding is the purpose of this repository.


