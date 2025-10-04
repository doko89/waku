# GitHub Actions Setup Guide

This guide explains how to use GitHub Actions for WAKU project to create releases.

## Prerequisites

- Repository admin or write access
- GitHub account

## No Setup Required! 🎉

The workflows are ready to use out of the box. No Personal Access Token (PAT) or additional configuration needed.

The tag workflow now uses direct `git push` which automatically triggers the build workflow.

## How to Use

1. Go to "Actions" tab in your repository

2. You should see two workflows:
   - ✅ Create Tag
   - ✅ Build and Release

### 4. Test the Workflow

1. Go to "Actions" → "Create Tag"

2. Click "Run workflow"

3. Enter a version number (e.g., `1.0.1`)

4. Click "Run workflow"

5. Watch the workflow run:
   - ✅ Tag workflow creates the tag
   - ✅ Build workflow automatically starts (triggered by the tag)
   - ✅ Binaries are built for all platforms
   - ✅ Docker images are built and pushed
   - ✅ GitHub Release is created with all artifacts

## Workflow Behavior

### With PAT_TOKEN (Recommended)

```
User triggers "Create Tag" workflow
    ↓
Tag is created and pushed
    ↓
Build workflow is AUTOMATICALLY triggered
    ↓
Binaries + Docker images are built
    ↓
GitHub Release is created
```

### Without PAT_TOKEN (Fallback)

If you don't set up PAT_TOKEN, the workflow will still work but with a limitation:

```
User triggers "Create Tag" workflow
    ↓
Tag is created and pushed
    ↓
Build workflow tries to trigger (may fail)
    ↓
User needs to manually trigger "Build and Release" workflow
```

**Manual trigger steps:**
1. Go to "Actions" → "Build and Release"
2. Click "Run workflow"
3. Enter the tag name (e.g., `v1.0.1`)
4. Click "Run workflow"

## Troubleshooting

### Build workflow doesn't start automatically

**Possible causes:**
1. PAT_TOKEN is not set or expired
2. PAT_TOKEN doesn't have correct permissions

**Solution:**
- Verify PAT_TOKEN secret exists in repository settings
- Check token expiration date
- Regenerate token with correct scopes if needed
- Manually trigger build workflow as fallback

### Tag already exists error

**Solution:**
```bash
# Delete tag locally
git tag -d v1.0.0

# Delete tag on GitHub
git push origin :refs/tags/v1.0.0

# Try again with the same or different version
```

### Build fails

**Check:**
1. Go to "Actions" tab
2. Click on the failed workflow run
3. Check the logs for specific errors
4. Common issues:
   - Go version mismatch
   - Missing dependencies
   - Docker build errors

## Security Notes

### PAT Token Security

- ✅ **DO**: Store PAT in repository secrets (encrypted)
- ✅ **DO**: Use minimal required scopes
- ✅ **DO**: Set expiration date
- ✅ **DO**: Rotate tokens periodically
- ❌ **DON'T**: Share PAT with anyone
- ❌ **DON'T**: Commit PAT to code
- ❌ **DON'T**: Use PAT in public logs

### Token Permissions

The PAT only needs:
- `repo` - To push tags and create releases
- `workflow` - To trigger workflows

It does NOT need:
- Admin permissions
- Delete permissions
- User permissions

## Alternative: Using GitHub App

For better security, you can use a GitHub App instead of PAT:

1. Create a GitHub App with repository permissions
2. Install the app on your repository
3. Use app authentication in workflows

This is more complex but provides better security and audit trails.

## Support

If you encounter issues:
1. Check workflow logs in Actions tab
2. Verify all secrets are set correctly
3. Check token permissions and expiration
4. Open an issue on GitHub

## Summary

✅ **Recommended Setup:**
- Create PAT with `repo` and `workflow` scopes
- Add as `PAT_TOKEN` secret
- Workflows will work automatically

✅ **Minimal Setup:**
- Skip PAT creation
- Manually trigger build workflow after creating tag
- Still fully functional, just requires one extra step

