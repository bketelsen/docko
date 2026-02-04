# Quick Task 002: Add GitHub Actions Workflow for Docker Build and Push

**One-liner:** CI workflow extended with GHCR Docker build/push on main branch after tests pass.

## Changes Made

### Task 1: Add Docker build and push job to CI workflow

Added a new `docker-build-push` job to `.github/workflows/ci.yml` that:

1. **Conditional execution:** Only runs on main branch (`if: github.ref == 'refs/heads/main'`)
2. **Dependencies:** Waits for lint and test jobs to pass (`needs: [lint, test]`)
3. **Permissions:** Has `packages: write` for GHCR push access
4. **Actions used:**
   - `actions/checkout@v5` - Clone repository
   - `docker/setup-buildx-action@v3` - Set up Docker Buildx
   - `docker/login-action@v3` - Authenticate to GHCR
   - `docker/metadata-action@v5` - Generate image tags (sha, branch)
   - `docker/build-push-action@v6` - Build and push image

5. **Image configuration:**
   - Registry: `ghcr.io`
   - Image name: `ghcr.io/bketelsen/docko`
   - Tags: SHA-based and branch-based via metadata action
   - Cache: GitHub Actions cache (gha) for faster builds

## Files Modified

| File | Change |
|------|--------|
| `.github/workflows/ci.yml` | Added docker-build-push job (40 lines) |

## Commits

| Hash | Description |
|------|-------------|
| d992de9 | feat(quick-002): add Docker build and push to GHCR |

## Verification Checklist

- [x] .github/workflows/ci.yml contains docker-build-push job
- [x] Job has `if: github.ref == 'refs/heads/main'` condition
- [x] Job depends on lint and test (`needs: [lint, test]`)
- [x] Uses docker/login-action with ghcr.io registry
- [x] Uses docker/build-push-action with push: true
- [x] Image tagged as ghcr.io/bketelsen/docko
- [x] Job has packages: write permission

## Deviations from Plan

None - plan executed exactly as written.

## Duration

~2 minutes
