# Development Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the approved MyRoutine development policy into a reproducible branch, Pull Request, CI, release, and GitHub Ruleset workflow.

**Architecture:** Bootstrap `develop` once from the current `main`, then make every repository change on an issue-linked short-lived branch. Keep policy and automation versioned in Git, use one trustworthy aggregate CI gate for branch protection, and store importable Ruleset payloads so GitHub settings remain auditable.

**Tech Stack:** Git, GitHub, GitHub CLI, GitHub Actions, YAML, Bash, JSON, jq, actionlint 1.7.12, Dependabot, Gitleaks, Go, React/Vite, Expo, Docker

**Spec:** `docs/superpowers/specs/2026-09-26-development-governance-design.md`

## Global Constraints

- `main` is the production branch; `develop` is the staging integration branch.
- Direct pushes to `main` and `develop` are forbidden after bootstrap.
- Work branches are short-lived, issue-linked, lowercase, English, and hyphenated.
- Work Pull Requests use squash merge; release Pull Requests use merge commits.
- Solo development requires a Pull Request, self-review, and green CI, but zero independent approvals.
- The first regular collaborator changes required approvals from zero to one.
- CI and deployment remain separate workflows.
- GitHub Actions use least-privilege token permissions and versioned release refs; branch refs such as `master` are forbidden.
- The stable required status check is `📊 Security Summary`; it must fail when any required upstream job fails or is cancelled.
- Product hosting, AI integration, desktop packaging, and app-store delivery remain outside this plan.
- The existing local Git identity and `github-pessoal` SSH remote remain unchanged.

## Review Focus

1. **A failed or cancelled child CI job:** `scripts/ci/security-gate_test.sh` must prove that the aggregate gate exits non-zero rather than publishing a false success.
2. **A Pull Request to `develop`:** actionlint and trigger assertions must prove the same CI runs for both protected branches.
3. **An invalid or off-main release tag:** release-script tests and the workflow ancestry check must reject it before a GitHub Release is created.
4. **A malformed issue form, Dependabot file, or Ruleset JSON:** YAML parsing, `jq`, and Ruleset API validation must fail before the configuration is applied.
5. **Ruleset bootstrap locking out the owner:** protections must start with zero approvals, no bypass actor, and a verified green aggregate check before activation.

---

### Task 1: Bootstrap the integration branch and isolated work branch

**Files:**
- Read: `docs/superpowers/specs/2026-09-26-development-governance-design.md`
- Read: `docs/superpowers/plans/2026-09-26-development-governance-implementation.md`
- Modify: `.github/workflows/ci.yml` (bootstrap trigger only)

**Interfaces:**
- Consumes: the reviewed spec and plan on `main`
- Produces: remote `develop`, one GitHub governance issue, and the issue-linked governance branch based on `develop`

- [ ] **Step 1: Verify the bootstrap baseline**

Run:

```bash
git status --short
git branch --show-current
git log -2 --oneline
git remote -v
```

Expected: clean worktree, current branch `main`, the spec and plan commits visible, and `origin` using `github-pessoal:LuizMRBsantos/MyRoutine.git`.

- [ ] **Step 2: Publish the reviewed documentation commits**

Run:

```bash
git push origin main
```

Expected: remote `main` advances to the reviewed local commit and the existing CI starts.

- [ ] **Step 3: Verify the bootstrap CI before branching**

Run:

```bash
MYROUTINE_RUN_ID=$(gh run list --branch main --workflow "CI — Build, Test & Security" --limit 1 --json databaseId --jq '.[0].databaseId')
test -n "$MYROUTINE_RUN_ID"
gh run watch "$MYROUTINE_RUN_ID" --exit-status
```

Expected: `CI — Build, Test & Security` completes successfully.

- [ ] **Step 4: Create the governance issue and record its number**

Run:

```bash
gh issue create --title "chore: establish development governance" --body "Implement the approved repository governance design in docs/superpowers/specs/2026-09-26-development-governance-design.md, including collaboration templates, CI gates, releases, environments, and protected-branch Rulesets."
```

Expected: GitHub returns the new issue URL. Use its actual number in every later branch and Pull Request reference; do not invent one.

Resolve reusable task-specific values:

```bash
MYROUTINE_ISSUE_NUMBER=$(gh issue list --state open --search '"chore: establish development governance" in:title' --limit 1 --json number --jq '.[0].number')
MYROUTINE_GOV_BRANCH="chore/${MYROUTINE_ISSUE_NUMBER}-development-governance"
test -n "$MYROUTINE_ISSUE_NUMBER"
```

Expected: both variables resolve from the issue GitHub just created.

- [ ] **Step 5: Create and publish `develop` at the verified `main` commit**

Run:

```bash
git branch develop main
git push --set-upstream origin develop
git ls-remote --heads origin main develop
```

Expected: both remote refs exist and point to the same bootstrap commit.

- [ ] **Step 6: Add the bootstrap Pull Request trigger**

GitHub evaluates Pull Request triggers from the base branch, so the first PR to `develop` cannot validate itself while the base workflow lists only `main`. Switch to `develop` and change only `pull_request.branches` in `.github/workflows/ci.yml` from `[main]` to `[main, develop]`.

- [ ] **Step 7: Validate the bootstrap workflow change**

Run:

```bash
docker run --rm -v "$PWD:/repo" --workdir /repo rhysd/actionlint:1.7.12 -color
git diff --check
```

Expected: both commands pass and the diff contains exactly the one trigger-line change.

- [ ] **Step 8: Commit and push the one-time bootstrap exception**

Run:

```bash
git add .github/workflows/ci.yml
git commit -m "ci(github): validate pull requests to develop"
git push origin develop
```

Expected: one auditable bootstrap commit lands directly on unprotected `develop`; no other governance implementation bypasses a Pull Request.

- [ ] **Step 9: Verify the `develop` push CI and return to `main`**

Run:

```bash
MYROUTINE_DEVELOP_RUN_ID=$(gh run list --branch develop --workflow "CI — Build, Test & Security" --limit 1 --json databaseId --jq '.[0].databaseId')
test -n "$MYROUTINE_DEVELOP_RUN_ID"
gh run watch "$MYROUTINE_DEVELOP_RUN_ID" --exit-status
git switch main
```

Expected: the bootstrap CI passes and the primary checkout returns to `main`.

- [ ] **Step 10: Create an isolated worktree from `develop`**

Use `superpowers:using-git-worktrees` to create `$MYROUTINE_GOV_BRANCH` from `develop`.

Expected: implementation occurs outside the main checkout, with a clean worktree and the correct base.

### Task 2: Publish contributor, security, and operational policy

**Files:**
- Create: `CONTRIBUTING.md`
- Create: `SECURITY.md`
- Create: `docs/DEVELOPMENT_WORKFLOW.md`
- Create: `CHANGELOG.md`
- Modify: `docs/PROJECT_CONTEXT.md`

**Interfaces:**
- Consumes: policy decisions from the approved spec
- Produces: contributor entry point, vulnerability-reporting policy, daily command reference, and release changelog contract

- [ ] **Step 1: Write a documentation acceptance checklist**

Create a temporary review checklist in the task notes asserting that the documents expose these exact concepts: `main`, `develop`, every branch prefix, Definition of Ready, Definition of Done, PR review stages, Conventional Commits, local verification commands, hotfix synchronization, private security reporting, and an `[Unreleased]` changelog section.

- [ ] **Step 2: Write `CONTRIBUTING.md`**

Include prerequisites, issue-first work, branch naming, commit examples, PR lifecycle, merge methods, and exact local checks:

```bash
cd backend && go test -race ./...
cd frontend && npm run type-check && npm run lint && npm run test:ci && npm run build
cd mobile && npx tsc --noEmit && npm test
```

Link to `docs/DEVELOPMENT_WORKFLOW.md`, `SECURITY.md`, and the approved design spec instead of duplicating long policy text.

- [ ] **Step 3: Write `SECURITY.md` and `CHANGELOG.md`**

`SECURITY.md` must forbid public disclosure of unpatched vulnerabilities and direct reporters to GitHub private vulnerability reporting. `CHANGELOG.md` must follow Keep a Changelog headings and begin with `[Unreleased]`; no released product version is invented during governance bootstrap.

- [ ] **Step 4: Write the operational workflow and update project context**

`docs/DEVELOPMENT_WORKFLOW.md` must be the concise day-to-day runbook, including normal, release, and hotfix flows. Update `docs/PROJECT_CONTEXT.md` to link to the spec, plan, contributing guide, and runbook and to distinguish current implementation from future policy.

- [ ] **Step 5: Verify documentation coverage and links**

Run:

```bash
rg -n "main|develop|feature/|fix/|hotfix/|Definition of Ready|Definition of Done|Conventional Commits|Unreleased" CONTRIBUTING.md SECURITY.md CHANGELOG.md docs/DEVELOPMENT_WORKFLOW.md docs/PROJECT_CONTEXT.md
git diff --check
```

Expected: every checklist concept is present, every relative link resolves in the worktree, and `git diff --check` prints nothing.

- [ ] **Step 6: Commit the policy documents**

```bash
git add CONTRIBUTING.md SECURITY.md CHANGELOG.md docs/DEVELOPMENT_WORKFLOW.md docs/PROJECT_CONTEXT.md
git commit -m "docs(project): add contributor workflow"
```

### Task 3: Add structured issue and Pull Request templates

**Files:**
- Create: `.github/pull_request_template.md`
- Create: `.github/ISSUE_TEMPLATE/config.yml`
- Create: `.github/ISSUE_TEMPLATE/feature.yml`
- Create: `.github/ISSUE_TEMPLATE/bug.yml`
- Create: `.github/ISSUE_TEMPLATE/technical.yml`

**Interfaces:**
- Consumes: Definition of Ready, Definition of Done, and PR evidence requirements
- Produces: GitHub forms for product features, bugs, technical work, and one standard PR review contract

- [ ] **Step 1: Write template assertions**

Define the required issue-form IDs before writing YAML:

```text
feature: problem, outcome, scope, acceptance, risks
bug: description, reproduction, expected, actual, environment, severity
technical: objective, rationale, scope, acceptance, risks
```

Define PR headings: Summary, Linked issue, Changes, Test evidence, UI evidence, Risk and rollback, Security, Documentation, Checklist.

- [ ] **Step 2: Create the three issue forms and chooser configuration**

Use GitHub Issue Forms YAML with non-empty `name`, `description`, `title`, `labels`, and `body`. Make objective/reproduction and acceptance fields required. Disable blank issues through `blank_issues_enabled: false` and keep private security reporting out of public issue links.

- [ ] **Step 3: Create the Pull Request template**

The checklist must cover self-review, issue linkage, tests, docs, migrations, secret scanning, backward compatibility, and changelog impact. It must not claim that an approval is required during solo stage.

- [ ] **Step 4: Parse and inspect the templates**

Run:

```bash
ruby -e 'require "yaml"; Dir[".github/ISSUE_TEMPLATE/*.{yml,yaml}"].each { |f| YAML.safe_load_file(f, aliases: false); puts "OK #{f}" }'
rg -n "Summary|Linked issue|Test evidence|Risk and rollback|Security|Documentation|Checklist" .github/pull_request_template.md
git diff --check
```

Expected: every YAML file prints `OK`, every required PR section is found, and the diff check is empty.

- [ ] **Step 5: Commit the collaboration templates**

```bash
git add .github/ISSUE_TEMPLATE .github/pull_request_template.md
git commit -m "chore(github): add issue and pull request templates"
```

### Task 4: Establish ownership and automated dependency maintenance

**Files:**
- Create: `.github/CODEOWNERS`
- Create: `.github/dependabot.yml`

**Interfaces:**
- Consumes: GitHub owner `@LuizMRBsantos` and the repository directories `/backend`, `/frontend`, `/mobile`, `/infra`, `/.github`
- Produces: automatic reviewer routing and grouped dependency-update Pull Requests

- [ ] **Step 1: Write the expected ownership map**

Assert these rules before implementation:

```text
* @LuizMRBsantos
/.github/ @LuizMRBsantos
/backend/ @LuizMRBsantos
/frontend/ @LuizMRBsantos
/mobile/ @LuizMRBsantos
/infra/ @LuizMRBsantos
/SECURITY.md @LuizMRBsantos
```

- [ ] **Step 2: Create `CODEOWNERS`**

Use the exact ownership map above. The catch-all rule comes first; specific sensitive areas follow it.

- [ ] **Step 3: Create `dependabot.yml`**

Configure weekly grouped updates for:

- `gomod` in `/backend`.
- `npm` in `/frontend`.
- `npm` in `/mobile`.
- `docker` in `/backend` and `/frontend`.
- `github-actions` in `/`.

Each entry targets `develop`, limits open Pull Requests to five, and uses `chore(deps)` as the commit prefix. Production and development dependency groups remain separate for npm where Dependabot supports that distinction.

- [ ] **Step 4: Validate ownership and Dependabot YAML**

Run:

```bash
ruby -e 'require "yaml"; YAML.safe_load_file(".github/dependabot.yml", aliases: false); puts "OK dependabot"'
rg -n "@LuizMRBsantos|backend|frontend|mobile|infra|SECURITY" .github/CODEOWNERS
git diff --check
```

Expected: YAML parses, all ownership areas appear, and no whitespace error is reported.

- [ ] **Step 5: Commit ownership and maintenance configuration**

```bash
git add .github/CODEOWNERS .github/dependabot.yml
git commit -m "chore(github): configure ownership and dependency updates"
```

### Task 5: Make CI a trustworthy branch-protection gate

**Files:**
- Modify: `.github/workflows/ci.yml`
- Create: `scripts/ci/security-gate.sh`
- Create: `scripts/ci/security-gate_test.sh`

**Interfaces:**
- Consumes: five job results in this order: `secret-scan`, `backend`, `frontend`, `mobile`, `container-scan`
- Produces: `scripts/ci/security-gate.sh RESULT_1 RESULT_2 RESULT_3 RESULT_4 RESULT_5` returning `0` only when all five values equal `success`, plus required check `📊 Security Summary`

- [ ] **Step 1: Write the failing aggregate-gate tests**

`scripts/ci/security-gate_test.sh` must assert:

- Five `success` arguments exit `0`.
- Any one `failure` exits non-zero.
- Any one `cancelled` exits non-zero.
- Any one `skipped` exits non-zero.
- Missing arguments exit non-zero.

- [ ] **Step 2: Run the tests to verify the gate is absent**

Run:

```bash
bash scripts/ci/security-gate_test.sh
```

Expected: FAIL because `scripts/ci/security-gate.sh` does not exist.

- [ ] **Step 3: Implement the minimal aggregate gate**

Create `scripts/ci/security-gate.sh` with the exact five-argument interface. It writes the Markdown results table to `$GITHUB_STEP_SUMMARY` when that variable is set and exits `1` unless every result is `success`.

- [ ] **Step 4: Run the gate tests**

Run:

```bash
bash scripts/ci/security-gate_test.sh
bash -n scripts/ci/security-gate.sh scripts/ci/security-gate_test.sh
```

Expected: PASS and no shell syntax errors.

- [ ] **Step 5: Harden `.github/workflows/ci.yml`**

Make these exact behavior changes:

- Preserve the bootstrapped `pull_request.branches: [main, develop]` trigger.
- Add `permissions: contents: read` at workflow level.
- Add concurrency keyed by workflow and PR branch, with stale runs cancelled.
- Add finite `timeout-minutes` to every job.
- Give only the SARIF upload job `security-events: write` in addition to `contents: read`.
- Upgrade Node 24-compatible actions to `actions/checkout@v7`, `actions/setup-go@v7`, `actions/setup-node@v7`, `gitleaks/gitleaks-action@v3`, `codecov/codecov-action@v7`, and `github/codeql-action/upload-sarif@v4`.
- Replace `aquasecurity/trivy-action@master` with the immutable release tag `aquasecurity/trivy-action@v0.36.0`.
- Preserve all current backend, frontend, mobile, secret, and container checks.
- Replace the inline summary-only script with `scripts/ci/security-gate.sh` and the results from `needs.secret-scan`, `needs.backend`, `needs.frontend`, `needs.mobile`, and `needs.container-scan`.
- Keep `if: always()` so the aggregate gate reports failures and cancellations.

- [ ] **Step 6: Validate workflow syntax and trigger coverage**

Run:

```bash
docker run --rm -v "$PWD:/repo" --workdir /repo rhysd/actionlint:1.7.12 -color
rg -n "branches: \[main, develop\]|permissions:|concurrency:|timeout-minutes|security-gate.sh" .github/workflows/ci.yml
bash scripts/ci/security-gate_test.sh
```

Expected: actionlint exits `0`, all hardening markers are found, and gate tests pass.

- [ ] **Step 7: Run the product checks affected by CI orchestration**

Run the same backend, frontend, and mobile commands documented in `CONTRIBUTING.md`.

Expected: backend tests, frontend type/lint/test/build, and mobile type/tests all pass.

- [ ] **Step 8: Commit the CI gate**

```bash
git add .github/workflows/ci.yml scripts/ci/security-gate.sh scripts/ci/security-gate_test.sh
git commit -m "ci(github): enforce pull request quality gate"
```

### Task 6: Add release notes and tag validation automation

**Files:**
- Create: `.github/release.yml`
- Create: `.github/workflows/release.yml`
- Create: `scripts/release/validate-release.sh`
- Create: `scripts/release/validate-release_test.sh`

**Interfaces:**
- Consumes: tag argument `vMAJOR.MINOR.PATCH`, changelog path argument, and a commit reachable from `origin/main`
- Produces: validated semantic version and a GitHub Release with generated categorized notes

- [ ] **Step 1: Write failing release-validator tests**

The test script must prove:

- `v0.2.0` with a matching `## [0.2.0]` changelog heading passes.
- `0.2.0`, `v1.2`, and `v1.2.3.4` fail.
- A syntactically valid tag absent from the supplied changelog fails.
- `[Unreleased]` alone never satisfies a versioned release.

- [ ] **Step 2: Run the tests to verify the validator is absent**

Run: `bash scripts/release/validate-release_test.sh`

Expected: FAIL because `scripts/release/validate-release.sh` does not exist.

- [ ] **Step 3: Implement the release validator**

Create `scripts/release/validate-release.sh TAG CHANGELOG_PATH`. Accept only `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$` and require the corresponding `## [MAJOR.MINOR.PATCH]` heading.

- [ ] **Step 4: Run release tests and syntax checks**

Run:

```bash
bash scripts/release/validate-release_test.sh
bash -n scripts/release/validate-release.sh scripts/release/validate-release_test.sh
```

Expected: PASS.

- [ ] **Step 5: Configure generated release-note categories**

Create `.github/release.yml` with categories for breaking changes, features, fixes, security, documentation, dependencies, and maintenance. Exclude labels used only for duplicate or invalid issues.

- [ ] **Step 6: Create the release workflow**

`.github/workflows/release.yml` must:

- Trigger only on pushed `v*.*.*` tags.
- Use `permissions: contents: write` and no broader permission.
- Use the GitHub `production` environment so environment protection can be strengthened later without rewriting the release job.
- Checkout full history.
- Run the release validator against `CHANGELOG.md`.
- Fetch `origin/main` and require `git merge-base --is-ancestor "$GITHUB_SHA" origin/main`.
- Create the GitHub Release with `gh release create "$GITHUB_REF_NAME" --verify-tag --generate-notes`.
- Use a production concurrency group so two releases cannot publish simultaneously.

- [ ] **Step 7: Validate release YAML and workflow**

Run:

```bash
ruby -e 'require "yaml"; YAML.safe_load_file(".github/release.yml", aliases: false); puts "OK release config"'
docker run --rm -v "$PWD:/repo" --workdir /repo rhysd/actionlint:1.7.12 -color
bash scripts/release/validate-release_test.sh
```

Expected: all commands pass.

- [ ] **Step 8: Commit release automation**

```bash
git add .github/release.yml .github/workflows/release.yml scripts/release
git commit -m "ci(release): automate validated GitHub releases"
```

### Task 7: Version and apply GitHub Rulesets

**Files:**
- Create: `.github/rulesets/main.json`
- Create: `.github/rulesets/develop.json`
- Create: `.github/environments/staging.json`
- Create: `.github/environments/production.json`
- Create: `docs/GITHUB_ADMINISTRATION.md`

**Interfaces:**
- Consumes: stable required check context `📊 Security Summary`, exact branch refs `refs/heads/main` and `refs/heads/develop`
- Produces: two active repository branch Rulesets and two deployment-environment foundations with reproducible JSON definitions

- [ ] **Step 1: Capture the current remote governance state**

Run:

```bash
gh api repos/LuizMRBsantos/MyRoutine/rulesets
gh api repos/LuizMRBsantos/MyRoutine/branches/main/protection
gh api repos/LuizMRBsantos/MyRoutine/branches/develop/protection
```

Expected: save the non-secret output in task notes. A `404` for absent protection is an acceptable baseline, not a reason to skip configuration.

- [ ] **Step 2: Create the `develop` Ruleset payload**

Set:

- Name: `Protect develop`.
- Target: `branch`.
- Enforcement: `active`.
- Include: `refs/heads/develop` only.
- No bypass actors.
- Pull Request required with zero approvals during solo stage, conversation resolution required, and stale-review dismissal enabled for future approvals.
- Required status check: `📊 Security Summary` with strict branch freshness.
- Deletion and non-fast-forward updates blocked.

- [ ] **Step 3: Create the `main` Ruleset payload**

Use the same controls with name `Protect main` and include only `refs/heads/main`. Do not require linear history because release Pull Requests intentionally use merge commits.

- [ ] **Step 4: Write the administration runbook**

Document JSON validation, create/import, update, read-back verification, rollback by setting enforcement to `disabled`, and the exact future change from zero to one required approval. State that Rulesets and environments are external state and their read-back output must be reviewed after every update.

- [ ] **Step 5: Define staging and production environment payloads**

Both environment payloads use `wait_timer: 0`, `prevent_self_review: false`, no reviewers during the solo stage, and custom deployment branch policies. `staging` permits only `develop`; `production` permits only `main`. The runbook must state that a future independent production approver is added when the first regular collaborator joins.

- [ ] **Step 6: Validate the payloads locally**

Run:

```bash
jq empty .github/rulesets/main.json .github/rulesets/develop.json .github/environments/staging.json .github/environments/production.json
jq -e '.target == "branch" and .enforcement == "active"' .github/rulesets/main.json .github/rulesets/develop.json
git diff --check
```

Expected: every command exits `0`.

- [ ] **Step 7: Commit the declarative governance configuration**

```bash
git add .github/rulesets .github/environments docs/GITHUB_ADMINISTRATION.md
git commit -m "chore(github): define protected branch rulesets"
```

- [ ] **Step 8: Push the work branch and wait for a green CI run**

Run:

```bash
MYROUTINE_ISSUE_NUMBER=$(gh issue list --state open --search '"chore: establish development governance" in:title' --limit 1 --json number --jq '.[0].number')
MYROUTINE_GOV_BRANCH="chore/${MYROUTINE_ISSUE_NUMBER}-development-governance"
git push --set-upstream origin "$MYROUTINE_GOV_BRANCH"
gh pr create --draft --base develop --head "$MYROUTINE_GOV_BRANCH" --title "chore: establish development governance" --body "Closes #${MYROUTINE_ISSUE_NUMBER}"
gh pr checks --watch --fail-fast
```

Expected: the PR targets `develop`, all jobs run, and `📊 Security Summary` is green.

- [ ] **Step 9: Apply `develop` and `main` Rulesets through the REST API**

After explicit confirmation of repository-admin authority, apply the reviewed payloads using these commands when no same-name Ruleset exists:

```bash
gh api --method POST repos/LuizMRBsantos/MyRoutine/rulesets --input .github/rulesets/develop.json
gh api --method POST repos/LuizMRBsantos/MyRoutine/rulesets --input .github/rulesets/main.json
```

If a same-name Ruleset already exists, obtain its ID from `gh api repos/LuizMRBsantos/MyRoutine/rulesets` and update that ID with `--method PUT` instead of creating a duplicate.

Expected: each response reports `enforcement: active` and the intended branch condition.

- [ ] **Step 10: Create the GitHub deployment environments**

Use `gh api --method PUT repos/LuizMRBsantos/MyRoutine/environments/staging --input .github/environments/staging.json` and the equivalent command for `production`. Then create the custom branch policy `develop` for staging and `main` for production through the deployment-branch-policies endpoints documented in `docs/GITHUB_ADMINISTRATION.md`.

Expected: the API returns environments named `staging` and `production`, each restricted to its intended branch.

- [ ] **Step 11: Read back and test the active protections**

Run:

```bash
gh api repos/LuizMRBsantos/MyRoutine/rulesets
gh api repos/LuizMRBsantos/MyRoutine/rules/branches/main
gh api repos/LuizMRBsantos/MyRoutine/rules/branches/develop
gh api repos/LuizMRBsantos/MyRoutine/environments/staging
gh api repos/LuizMRBsantos/MyRoutine/environments/production
```

Expected: both branches show Pull Request, required status check, deletion, and force-push protections, with zero required approvals.

### Task 8: Complete the bootstrap through the new workflow

**Files:**
- Modify only if review finds a defect in files from Tasks 2–7

**Interfaces:**
- Consumes: the governance Pull Request, active Rulesets, and required aggregate check
- Produces: merged `develop`, a reviewed bootstrap promotion to `main`, and a recorded proof that direct protected-branch work is no longer needed

- [ ] **Step 1: Perform the PR self-review**

Use the new template. Review the complete diff, link the governance issue, attach CI evidence, mark the bootstrap exception, and move the PR out of Draft.

- [ ] **Step 2: Verify the full branch gate**

Run:

```bash
gh pr checks --watch --fail-fast
gh pr view --json mergeStateStatus,reviewDecision,statusCheckRollup
```

Expected: merge state is clean, required checks are successful, and zero external approvals are required during solo stage.

- [ ] **Step 3: Squash merge into `develop`**

Run:

```bash
gh pr merge --squash --delete-branch
```

Expected: one governance delivery commit lands on `develop` and the work branch is deleted remotely.

- [ ] **Step 4: Open the bootstrap promotion Pull Request**

Run:

```bash
gh pr create --base main --head develop --title "chore(release): establish development governance" --body "Bootstrap promotion of the approved governance system. No product deployment or version tag is created because hosting is not configured yet."
```

Expected: the PR documents the one-time no-tag bootstrap exception and runs the complete CI against `main`.

- [ ] **Step 5: Merge the promotion with a merge commit**

After green checks and self-review:

```bash
gh pr checks --watch --fail-fast
gh pr merge --merge
```

Expected: `main` receives the governance system through a merge commit; no release tag or fake product release is created.

- [ ] **Step 6: Synchronize local state and verify the repository**

Run:

```bash
git fetch origin --prune
git log --graph --decorate --oneline --all -20
gh api repos/LuizMRBsantos/MyRoutine/rulesets
gh run list --limit 5
```

Expected: `main` and `develop` show the intended ancestry, both Rulesets remain active, recent CI runs are green, and the repository has no untracked implementation artifacts.

## Final Verification

- [ ] Confirm every file named in Tasks 2–7 exists on `main`.
- [ ] Confirm CI triggers on Pull Requests to both `develop` and `main`.
- [ ] Confirm `📊 Security Summary` fails in the scripted negative tests and passes only when every required job succeeds.
- [ ] Confirm issue forms and the Pull Request template render on GitHub.
- [ ] Confirm Dependabot targets `develop` for all configured ecosystems.
- [ ] Confirm both Rulesets are active with zero approvals and no bypass actors.
- [ ] Confirm direct push, force push, and deletion are blocked on protected branches.
- [ ] Confirm a future change from zero to one approval is documented.
- [ ] Confirm no tag or production release was fabricated during bootstrap.
