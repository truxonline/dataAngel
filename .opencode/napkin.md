# Napkin Runbook

## Curation Rules
- Re-prioritize on every read.
- Keep recurring, high-value notes only.
- Max 10 items per category.
- Each item includes date + "Do instead".

## Execution & Validation (Highest Priority)
1. **[2026-03-17] Multi-module Go project pattern**
   Do instead: Use `cmd/sidecar-*/go.mod` for each binary, no root go.mod, rely on `go work` or `replace` directives if needed for local development.

## Shell & Command Reliability
1. **[2026-03-17] Git commit hooks**
   Do instead: Ensure `pre-commit` is installed and configured for linting/formatting before committing.

## Domain Behavior Guardrails
1. **[2026-03-17] BMAD Cycle Strictness**
   Do instead: Always run `/bmad-help` to identify the next step in the BMAD workflow. Never skip steps (CS -> VS -> AT -> DS -> CR).

## User Directives
1. **[2026-03-17] TDD Preference**
   Do instead: Write failing tests first (RED), then implementation (GREEN), then refactor (REFACTOR).
2. **[2026-03-17] Agent Aggressiveness**
   Do instead: Prefer a single aggressive agent going to the end over 50 agents trying things.
3. **[2026-03-17] Self-Hosted Preference**
   Do instead: Avoid cloud dependencies and API keys; prefer self-hosted tools and scraping public data.
4. **[2026-03-17] French Language**
   Do instead: Respond in French except for code/technical terms.