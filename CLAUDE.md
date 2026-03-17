# CLAUDE

Load once per session. Re-read on "reload claude". On conflict with chat instructions: flag and prompt.
All sections are mandatory unless marked optional.

## Core rules

- Handle edge cases explicitly.
- No duplication. Eliminate on detection.
- Readability and maintainability before performance.
- Names: full words; single word unless compound required. No abbreviations (`database` not `db`). No type-redundant suffixes (`connection` not `dbConnection`). No context-redundant suffixes (`load` not `loadImage` from Image class or module).
- Tests required. Prefer granular nested assertions.
- No fragile workarounds, premature abstractions, or security vulnerabilities.
- Align with 12-factor and Agile principles.
- Mark technical debt with TODO comments.

## Code structure

- Splitting: Split modules into granular files. Max 200 LOC excluding comments.
- Block order: Imports → constants/variables near first use → helpers → main flow → single exit point (when practical).
- Imports: stdlib → third-party → internal → local modules.
- Functions: Local variables near first use → precondition checks (fail fast/early return) → extract predicates/callbacks to named variables before invocation → no nested calls in arguments unless trivial or test assertions → return named result, not inline expression (unless trivial).
- Rules: Use small functions with explicit I/O. Keep side-effects (file writes, network, database) near top-level flow, not in deep helpers, unless the caller requires the effect.

## Design principles

Apply by default: DRY, SOLID, Object Calisthenics (small methods, no nesting unless required, no deep chains), KISS, YAGNI. Default to immutability: no argument mutation, no shared-state reassignment; return new values. Mutate only when language or performance mandates; document mutation. Switch: early return, not break. Avoid code smells (feature envy, inappropriate intimacy, middleman, lazy class, dead code, long method, god class, mutable shared state). Measure cognitive complexity.

Extract hardcoded settings to environment variables; group by prefix or domain.

Wrap third-party integrations in adapters. Maintain provider-agnosticism unless project constraints require coupling.

## Schemas and types

- Schemas: Validate all user input with schemas.
- Types: Declare in dedicated files. Reuse in schemas and API contracts. Enforce type-checking in builds. Deserialize JSON/XML to declared types at runtime.
- Enums: Use enums over magic numbers/strings. Format: `MyEnum.MY_VALUE`.

## Docstrings

Docstring every top-level function/method/class: purpose, constraints, edge cases, params, return. Update on behavior change. Use language-standard docstring syntax strictly (GoDoc) — no ad-hoc substitutes. Annotate every parameter and return value with the standard's designated tags. Inline comments suffice for predicates/callbacks.

## Workflow

1. Read code, tests, docs before proposing.
2. Clarify only when ambiguity risks incorrect output.
3. Propose plan and tradeoffs before editing.
4. Await approval unless pre-authorized.
5. Small, reviewable changesets.
6. Include or update tests.
7. Include or update documentation (docstrings, README.md).
8. Maintain setup alignment (tests, checks, containers).
9. Commit without adding co-author details. Use conventional commits. Always sign commits.

## Project setup

### Tests and coverage

Maintain: unit tests, integration tests, smoke tests, end-to-end tests (when not complex), coverage reporting.

- Business logic and edge cases → unit tests.
- Boundaries (database, queue, cache, external service, file system) → integration tests.
- Deterministic tests: no sleeps, no time-dependent logic, stub all network.
- Isolate test cases. Fixtures, mocks, stubs in dedicated folders.
- Shared test setup (fixtures, helpers, utils) in `tests/mocks/`. Parameterize with generics over per-file helpers when logic is identical across files.
- Co-locate unit/integration tests beside source using `_test.go` suffix (Go convention). Root `tests/` for smoke and e2e only.
- One command for all tests; one command for coverage.

### Static checks

Maintain: formatter, linter, type-checker, dependency/code vulnerability scanner. Single command runs all. Auto-fix formatting. Fail build on violations.

## Security

- Input: Treat all external input as untrusted (request bodies, query strings, headers, file uploads, env vars, CLI args, database content, webhooks). Validate and sanitize at boundary. Allowlists over blocklists. Enforce length limits and type checks.
- File system: No arbitrary path access from user input. If required: resolve to allowed base directory, reject traversal (`..`, absolute paths, drive prefixes), prefer server-generated names. No nested relative imports; use aliases. No internal paths in logs or errors.
- Execution: No shell when library call exists. If required: structured argument arrays, no string concatenation, no untrusted shell input.
- Database: Parameterized queries or safe query builders only. No untrusted concatenation. Validate pagination/sort with allowlists.
- Output: Context-encode output (HTML, JSON, SQL, shell, URL). No logged secrets, tokens, credentials, or PII. Environment-based secret injection. Redact sensitive fields in logs.
- Verification: On auth/access/persistence changes: adversarial input tests, vulnerability scan, documentation update.

## Review process

Before starting, ask: complex change or simple change?

- Complex: Architecture → Code quality → Tests → Performance. Up to 4 issues per section. AskUserQuestion after each issue.
- Simple: Same sections, 1 issue per section. AskUserQuestion after each issue.AskUserQuestion after each section.
- Default: simple change.

### What to check

- Architecture: boundaries and ownership; dependencies and coupling; data flow, backpressure, idempotency; scaling risks and SPOF; security boundaries (auth, secrets, data access).
- Code quality: structure, naming, cohesion; duplication; error handling, retries, fallbacks, missing edge cases; debt hotspots; over/under-engineering.
- Tests: gaps across unit/integration/e2e; assertions that fail for correct reasons; edge cases and failure paths (timeouts, retries, invalid input).
- Performance: query and I/O patterns (batching, pagination, N+1); memory growth (streaming vs buffering); caching and invalidation; hot paths and algorithmic complexity; observability (timers, metrics, tracing).

## Issue format

For each issue:

- What: concrete problem
- Where: file path and line numbers
- Why: impact and risk, tied to core rules
- Options: 2-3 paths, always including "do nothing" and "other". Each: Effort (S/M/L), Risk (L/M/H), side effects, maintenance cost
- Recommendation: one opinionated pick, tied to core rules
- Ask: AskUserQuestion before proceeding

Format: numbered issues, lettered options. A = recommended (first). Include "Do nothing" and "Other" (free-text). One AskUserQuestion per issue.

## Documentation drift

Commit documentation updates when changing: test commands/tools/thresholds, static analysis config, container/CI/CD config, deployment config, repo scripts, public behavior, API contracts, architecture, operational procedures, security posture, performance characteristics.

After any change: verify documented commands exist. Container commands first. If documentation and code disagree, fix documentation immediately.

## Final self-check

Before finishing:

- Tests cover new behavior.
- Coverage runs and reports.
- Static checks pass.
- Duplication did not increase without reason.
- Error paths and edge cases are explicit.
- Documentation matches reality (commands and examples).
- If containers exist, primary workflow runs inside containers.
