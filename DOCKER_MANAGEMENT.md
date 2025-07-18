# Docker Image Management Strategy

This document outlines how Docker images are managed in this project for both main branch releases and feature branch development.

## Image Tagging Strategy

### Main Branch (`main`)
- **Versioned tags**: `1.0.4`, `1.0.5`, etc. (semantic versioning)
- **Latest tag**: `latest` (always points to the most recent main branch build)
- **Retention**: Main branch images are **never deleted** (permanent retention)

### Feature Branches (`feature/*`, `feat/*`)
- **Branch-based tags**: `feature-auth-fix-a1b2c3d` (branch name + short SHA)
- **Expiration**: Automatically deleted after 7 days
- **Labels**: Include `ghcr.io/expires-after=7d` for GitHub's native cleanup

### Pull Requests
- **PR-based tags**: Same as feature branches
- **Purpose**: Testing and review
- **Lifecycle**: Tied to feature branch lifecycle

## Automatic Cleanup

### Scheduled Cleanup
- **Frequency**: Daily at 2 AM UTC
- **Workflow**: `.github/workflows/cleanup-images.yml`
- **Rules**:
  - Delete feature branch images older than 7 days
  - **Never delete main branch images** (permanent retention for all releases)
  - Delete untagged images immediately

### Manual Cleanup
```bash
# Dry-run cleanup (recommended first)
make cleanup-images

# Actual cleanup (with confirmation)
make cleanup-images-force

# List all images
make list-images

# List all image versions with details
make list-image-versions
```

## GitHub CLI Setup

For manual image management, install the GitHub CLI:

```bash
# Ubuntu/Debian
sudo apt install gh

# macOS
brew install gh

# Authenticate
gh auth login
```

## Image Management Commands

### Makefile Targets

| Command | Description |
|---------|-------------|
| `make list-images` | List all Docker images in GHCR |
| `make list-image-versions` | List all versions with tags and dates |
| `make cleanup-images` | Trigger cleanup workflow (dry-run) |
| `make cleanup-images-force` | Trigger actual cleanup (with confirmation) |

### GitHub Actions Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `docker-publish.yml` | Push to any branch | Build and push images |
| `cleanup-images.yml` | Daily schedule + manual | Clean up old images |

## Image Lifecycle Examples

### Feature Branch Development
1. Create feature branch: `git checkout -b feature/new-auth`
2. Push changes: `git push origin feature/new-auth`
3. GitHub Actions builds: `ghcr.io/owner/custom-otel-collector:feature-new-auth-a1b2c3d`
4. After 7 days: Image automatically deleted

### Main Branch Release
1. Merge to main: `git checkout main && git merge feature/new-auth`
2. Push to main: `git push origin main`
3. GitHub Actions builds:
   - `ghcr.io/owner/custom-otel-collector:1.0.5` (new version)
   - `ghcr.io/owner/custom-otel-collector:latest` (updated)
4. **All versions preserved permanently** (never deleted)

## Best Practices

### For Developers
- Use descriptive feature branch names
- Don't worry about cleanup - it's automatic
- Use `make list-images` to see current images
- Test with feature branch images before merging

### For Maintainers
- Run `make cleanup-images` weekly to preview what will be cleaned
- Monitor the cleanup workflow logs
- Adjust feature branch retention period in `.github/workflows/cleanup-images.yml` if needed
- Use `make list-image-versions` to audit image usage
- **Note**: Main branch images are never automatically deleted

## Configuration

### Retention Periods
Edit `.github/workflows/cleanup-images.yml` to change:
- Feature branch retention: Change `7 * 24 * 60 * 60 * 1000` (7 days)
- **Main branch retention**: Permanent (never deleted by design)

### Cleanup Schedule
Edit the cron expression in `cleanup-images.yml`:
```yaml
schedule:
  - cron: '0 2 * * *'  # Daily at 2 AM UTC
```

### Image Expiration Labels
Feature branch images include GitHub's native expiration label:
```yaml
labels: ghcr.io/expires-after=7d
```

## Manual Main Branch Image Cleanup

Since main branch images are never automatically deleted, you may occasionally want to manually clean up very old releases:

### Using GitHub CLI
```bash
# List all main branch versions to review
make list-image-versions

# Delete a specific version (replace VERSION_ID with actual ID)
gh api --method DELETE /user/packages/container/custom-otel-collector/versions/VERSION_ID

# Or for organization packages:
gh api --method DELETE /orgs/ORG_NAME/packages/container/custom-otel-collector/versions/VERSION_ID
```

### Using GitHub Web Interface
1. Go to your repository
2. Click on "Packages" tab
3. Select "custom-otel-collector"
4. Click on the version you want to delete
5. Click "Delete version"

⚠️ **Warning**: Manual deletion of main branch images is permanent and cannot be undone. Only delete versions you're certain are no longer needed.

## Troubleshooting

### Common Issues

**"Package not found" when listing images**
- Ensure you have the correct permissions
- Check if the package exists in your user/org account
- Verify GitHub CLI authentication: `gh auth status`

**Cleanup workflow fails**
- Check if the GitHub token has `packages: write` permission
- Verify the package name matches in the workflow
- Check for API rate limits in the workflow logs

**Manual cleanup commands don't work**
- Install GitHub CLI: See setup instructions above
- Authenticate: `gh auth login`
- Check repository URL format in git config

### Debugging

```bash
# Check current git remote
git config --get remote.origin.url

# Test GitHub CLI access
gh api user

# Check workflow runs
gh run list --workflow=cleanup-images.yml

# View specific workflow run
gh run view <run-id>
```
