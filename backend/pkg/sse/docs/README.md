# SSE package documentation

This folder is the detailed reference for `pkg/sse`. The short entry point is
[../README.md](../README.md); generic integration and production guidance is in
[../INTEGRATION.md](../INTEGRATION.md); open transport work is in
[../ROADMAP.md](../ROADMAP.md).

| Document | Contents |
| --- | --- |
| [01-overview.md](01-overview.md) | Architecture and stream lifecycle |
| [02-getting-started.md](02-getting-started.md) | Mounting, auth and shutdown |
| [03-protocol.md](03-protocol.md) | SSE frames, ids, heartbeat and close events |
| [04-testing.md](04-testing.md) | Test coverage and commands |
| [05-usage.md](05-usage.md) | Call order, feature recipes, best practices and diagrams |
| [06-production-readiness.md](06-production-readiness.md) | Verdict, P0/P1 gaps and go-live checklist |
| [07-production-readiness-final.md](07-production-readiness-final.md) | Post-fix verdict, score and remaining P1 |
| [08-production-scope.md](08-production-scope.md) | Supported profile, non-goals and milestones |
| [PROTOCOL.md](../PROTOCOL.md) | Canonical SSE wire contract |

The package is generic. Project-specific rollout plans and ADRs belong in the
host project, not in this folder.
