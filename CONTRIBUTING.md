# Contributing to opg-ewbi-operator

Thank you for your interest in contributing! To keep the codebase healthy and reviews efficient, **all contributions must follow the guidelines below**. Pull requests that don't comply will be sent back for rework.

These guidelines are intentionally minimal. They are based on widely adopted open-source standards:

- [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
- [Kubernetes Community contributor guide](https://github.com/kubernetes/community/blob/master/contributors/guide/contributing.md)
- [Operator SDK contributing practices](https://sdk.operatorframework.io/)

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

Use the area of the codebase affected, e.g. `controller`, `api`, `helm`, `crd`, `opg`, `multipart`.

### Rules

1. **Subject line ≤ 72 characters.** Use imperative mood ("add", not "added" or "adds").
2. **One logical change per commit.** Don't mix refactors with features or bundle unrelated fixes. If you need to refactor something before your feature works, that's a separate commit.
3. **Body** (optional): A longer commit body may be provided after the short description, providing additional contextual information about the code changes. The body MUST begin one blank line after the description.
4. **Breaking changes**: add `BREAKING CHANGE:` in the footer or append `!` after the type, e.g. `feat(api)!: remove deprecated field`.

### Examples

```
feat(controller): reconcile ApplicationInstance status on federation change

The controller now watches Federation objects and re-queues any
ApplicationInstance whose spec.federationRef matches, ensuring the
status is updated when the remote cluster state changes.

Refs: #42
```

```
fix(opg): handle nil pointer when artefact has no checksum
```

```
chore: bump controller-gen to v0.16.5
```

---

## 2. Pull Requests

### Size

- **Small, focused PRs.** Each PR should represent **one logical unit of work** (one feature, one bug fix, one refactor). If your PR touches more than ~400 lines of non-generated code, consider splitting it.
- Generated code (deep copy, CRD manifests) doesn't count toward this guideline, but should be in its own commit (e.g. `chore: regenerate CRD manifests`).

### Description

Every PR **must** fill in the PR template. At minimum it must contain:

1. **What** — a one-line summary of the change.
2. **Why** — the motivation, context, or issue being fixed.
3. **How** — a brief description of the approach taken (especially for non-obvious changes).
4. **Testing** — how you verified the change works (unit tests, manual test, etc.).
5. **Breaking changes** — list any, or state "None".

### Branch naming

Use the pattern: `<type>/<short-description>`, e.g.:

- `feat/federation-status-sync`
- `fix/nil-pointer-artefact`
- `docs/contributing-guidelines`

### Before opening a PR

- [ ] Code compiles
- [ ] Tests pass
- [ ] Linter is clean
- [ ] CRD manifests regenerated if API types changed
- [ ] New/changed behaviour has tests
- [ ] Commits follow Conventional Commits (squash/rebase if needed)
- [ ] PR description is filled in completely

---

## 3. Code Review Expectations

- Reviewers **will request changes** if commits or the PR description don't follow these guidelines. This is not personal — it keeps the project maintainable.
- Address all review comments. If you disagree, explain why in the thread rather than ignoring the comment.
- After addressing feedback, **don't force-push over the review** — push new fixup commits so reviewers can see incremental changes, then squash before merge if needed.