---
phase: quick
plan: 002
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/ci.yml
autonomous: true

must_haves:
  truths:
    - "Tests run on push to main and PRs"
    - "Docker image builds and pushes to GHCR on main branch after tests pass"
    - "GHCR uses ghcr.io/bketelsen/docko as image name"
  artifacts:
    - path: ".github/workflows/ci.yml"
      provides: "CI workflow with Docker build and GHCR push"
      contains: "ghcr.io/bketelsen/docko"
  key_links:
    - from: "build-and-push job"
      to: "test job"
      via: "needs: [lint, test]"
      pattern: "needs:.*test"
---

<objective>
Add Docker build and push to GHCR to existing GitHub Actions CI workflow.

Purpose: Enable automatic container image builds on main branch after tests pass, pushing to GitHub Container Registry for deployment.
Output: Updated .github/workflows/ci.yml with docker-build-push job.
</objective>

<execution_context>
@/home/bjk/.claude/get-shit-done/workflows/execute-plan.md
@/home/bjk/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
Existing workflow at .github/workflows/ci.yml already has:
- lint job (golangci-lint)
- test job (go test, needs generate step)
- build job (go build, depends on lint+test)
- sqlc-vet job (validates SQL against live DB)

Existing Dockerfile is multi-stage (builder + alpine production).
Go version in go.mod is 1.25.0, but workflow uses 1.25 (keep workflow version as-is for compatibility).
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add Docker build and push job to CI workflow</name>
  <files>.github/workflows/ci.yml</files>
  <action>
Add a new job `docker-build-push` to the existing CI workflow:

1. Only runs on main branch (use `if: github.ref == 'refs/heads/main'`)
2. Depends on lint and test jobs (`needs: [lint, test]`)
3. Uses standard Docker build/push actions:
   - actions/checkout@v5
   - docker/setup-buildx-action@v3
   - docker/login-action@v3 with GHCR (registry: ghcr.io, username: ${{ github.actor }}, password: ${{ secrets.GITHUB_TOKEN }})
   - docker/metadata-action@v5 to generate tags (type=sha, type=ref,event=branch)
   - docker/build-push-action@v6 with push: true, tags from metadata, image: ghcr.io/bketelsen/docko

4. Add permissions block at job level: packages: write, contents: read

The existing Dockerfile works as-is (multi-stage, produces /app/docko binary).
</action>
<verify>
Read the updated .github/workflows/ci.yml and verify:

- docker-build-push job exists
- Has condition `if: github.ref == 'refs/heads/main'`
- Has `needs: [lint, test]`
- Has correct GHCR login and push configuration
- Image name is ghcr.io/bketelsen/docko
  </verify>
  <done>
  CI workflow updated with docker-build-push job that:
- Only runs on main branch pushes (not PRs)
- Runs after lint and test pass
- Builds Docker image using existing Dockerfile
- Pushes to ghcr.io/bketelsen/docko with sha and branch tags
  </done>
  </task>

</tasks>

<verification>
- [ ] .github/workflows/ci.yml contains docker-build-push job
- [ ] Job has `if: github.ref == 'refs/heads/main'` condition
- [ ] Job depends on lint and test (`needs: [lint, test]`)
- [ ] Uses docker/login-action with ghcr.io registry
- [ ] Uses docker/build-push-action with push: true
- [ ] Image tagged as ghcr.io/bketelsen/docko
- [ ] Job has packages: write permission
</verification>

<success_criteria>
GitHub Actions workflow updated such that:

1. On PRs: lint, test, build, sqlc-vet run (existing behavior preserved)
2. On main push: lint, test run, then docker-build-push builds and pushes image to GHCR
3. No manual secrets needed (uses built-in GITHUB_TOKEN)
   </success_criteria>

<output>
After completion, create `.planning/quick/002-add-github-actions-workflow-for-testing/002-SUMMARY.md`
</output>
