# Contributing to opg-ewbi-api

Thank you for your interest in contributing! To keep the codebase healthy and reviews efficient, **all contributions must follow the guidelines below**. Pull requests that don't comply will be sent back for rework.

These guidelines are intentionally minimal. They are based on widely adopted open-source standards:

- [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
- [Kubernetes Community contributor guide](https://github.com/kubernetes/community/blob/master/contributors/guide/contributing.md)

---

## 1. Commits

We follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). Every commit message **must** use this format:

```
<type>(<scope>): <short summary>

<optional body>

<optional footer(s)>
```

### Allowed types

| Type | When to use |
|---|---|
| `feat` | A new feature or user-facing behaviour |
| `fix` | A bug fix |
| `docs` | Documentation-only changes |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or updating tests |
| `chore` | Tooling, CI, dependencies, build scripts |
| `ci` | CI/CD pipeline changes |

### Scope (optional but encouraged)

Use the area of the codebase affected, e.g. `handler`, `metastore`, `deployment`, `api`, `federation`, `multipart`.

### Rules

1. **Subject line ≤ 72 characters.** Use imperative mood ("add", not "added" or "adds").
2. **One logical change per commit.** Don't mix refactors with features or bundle unrelated fixes. If you need to refactor something before your feature works, that's a separate commit.
3. **Body** (optional): A longer commit body may be provided after the short description, providing additional contextual information about the code changes. The body MUST begin one blank line after the description.
4. **Breaking changes**: add `BREAKING CHANGE:` in the footer or append `!` after the type, e.g. `feat(api)!: remove deprecated field`.

### Examples

```
feat(handler): add pagination support to application list endpoint

The handler now accepts page and pageSize query parameters and returns
paginated results with proper Link headers.

Refs: #12
```

```
fix(metastore): handle nil pointer when federation context is missing
```

```
chore: bump oapi-codegen to v2.4.0
```

---

## 2. Pull Requests

### Size

- **Small, focused PRs.** Each PR should represent **one logical unit of work** (one feature, one bug fix, one refactor). If your PR touches more than ~400 lines of non-generated code, consider splitting it.
- Generated code (oapi-codegen output) doesn't count toward this guideline, but should be in its own commit (e.g. `chore: regenerate API code`).

### Description

Every PR **must** fill in the PR template. At minimum it must contain:

1. **What** — a one-line summary of the change.
2. **Why** — the motivation, context, or issue being fixed.
3. **How** — a brief description of the approach taken (especially for non-obvious changes).
4. **Testing** — how you verified the change works (unit tests, manual test, etc.).
5. **Breaking changes** — list any, or state "None".

### Branch naming

Use the pattern: `<type>/<short-description>`, e.g.:

- `feat/pagination-support`
- `fix/nil-pointer-federation`
- `docs/contributing-guidelines`

### Before opening a PR

- [ ] Code compiles (`go build ./...`)
- [ ] Tests pass (`go test ./...`)
- [ ] Linter is clean
- [ ] API code regenerated if OpenAPI spec changed (`docker-compose build apigen`)
- [ ] New/changed behaviour has tests
- [ ] Commits follow Conventional Commits (squash/rebase if needed)
- [ ] PR description is filled in completely

---

## 3. Code Review Expectations

- Reviewers **will request changes** if commits or the PR description don't follow these guidelines. This is not personal — it keeps the project maintainable.
- Address all review comments. If you disagree, explain why in the thread rather than ignoring the comment.
- After addressing feedback, **don't force-push over the review** — push new fixup commits so reviewers can see incremental changes, then squash before merge if needed.
