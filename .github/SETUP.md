# GitHub Actions Setup Guide

This guide explains how to set up and use GitHub Actions for WAKU project to create automated releases.

## Prerequisites

- Repository admin or write access
- GitHub account

## Setup Required: SSH Deploy Key

The workflow uses SSH Deploy Key to push tags, which allows it to trigger the build workflow automatically.

### Why SSH Deploy Key?

By default, when a workflow uses `GITHUB_TOKEN` to push tags, GitHub **does not trigger** other workflows (to prevent recursive workflows). To solve this, we use an SSH Deploy Key which is treated as an external push, thus triggering the build workflow.

## Setup Steps

### 1. Generate SSH Key Pair

On your local machine:

```bash
# Generate a new SSH key (no passphrase)
ssh-keygen -t ed25519 -C "github-actions-waku" -f ~/.ssh/waku_deploy_key -N ""

# This creates two files:
# - waku_deploy_key (private key)
# - waku_deploy_key.pub (public key)
```

### 2. Add Deploy Key to Repository

1. **Copy the public key**:
   ```bash
   cat ~/.ssh/waku_deploy_key.pub
   ```

2. **Add to GitHub**:
   - Go to: https://github.com/YOUR_USERNAME/waku/settings/keys
   - Click "Add deploy key"
   - Title: `GitHub Actions Deploy Key`
   - Key: Paste the public key content
   - ✅ Check "Allow write access"
   - Click "Add key"

### 3. Add Private Key to Secrets

1. **Copy the private key**:
   ```bash
   cat ~/.ssh/waku_deploy_key
   ```

2. **Add to GitHub Secrets**:
   - Go to: https://github.com/YOUR_USERNAME/waku/settings/secrets/actions
   - Click "New repository secret"
   - Name: `SSH_DEPLOY_KEY`
   - Secret: Paste the **entire private key** (including `-----BEGIN` and `-----END` lines)
   - Click "Add secret"

### 4. Verify Setup

After adding both keys:

1. Go to: https://github.com/YOUR_USERNAME/waku/actions/workflows/create-tag.yml
2. Click "Run workflow"
3. Enter a test version (e.g., `1.0.3`)
4. Click "Run workflow"
5. Watch both workflows run:
   - ✅ Create Tag workflow
   - ✅ Build and Release workflow (auto-triggered)

## How to Use

### Creating a Release

1. **Go to Actions**:
   - https://github.com/YOUR_USERNAME/waku/actions/workflows/create-tag.yml

2. **Click "Run workflow"**

3. **Enter version** (without 'v' prefix):
   - Example: `1.0.3`

4. **Click "Run workflow"**

5. **Watch the magic happen**:
   - ✅ Tag `v1.0.3` is created
   - ✅ Tag is pushed via SSH
   - ✅ Build workflow automatically starts
   - ✅ Binaries are built for 7 platforms
   - ✅ GitHub Release is created with all artifacts

### Available Workflows

You should see two workflows in Actions tab:

1. **Create Tag and Build** - Main workflow to create releases
2. **Build and Release** - Automatically triggered by tags

## Workflow Behavior

### With SSH_DEPLOY_KEY (Current Setup)

```
User triggers "Create Tag and Build" workflow
    ↓
Checkout code with SSH agent
    ↓
Validate version format
    ↓
Pull latest changes
    ↓
Create tag v1.0.3
    ↓
Push tag via SSH (git@github.com:...)
    ↓
Build workflow is AUTOMATICALLY triggered ✅
    ↓
Build binaries for 7 platforms
    ↓
Generate checksums
    ↓
Create GitHub Release with all artifacts
    ↓
🎉 Done! Release is live!
```

**Total time:** ~5-10 minutes

### Without SSH_DEPLOY_KEY (Not Recommended)

If SSH_DEPLOY_KEY is not set, the workflow will fail or build won't trigger automatically.

**Why?** Because `GITHUB_TOKEN` cannot trigger other workflows (GitHub security feature).

## Troubleshooting

### Build workflow doesn't start automatically

**Possible causes:**
1. `SSH_DEPLOY_KEY` secret is not set
2. Deploy key is not added to repository
3. Deploy key doesn't have write access

**Solution:**
1. Verify `SSH_DEPLOY_KEY` secret exists:
   - Go to: https://github.com/YOUR_USERNAME/waku/settings/secrets/actions
   - Check if `SSH_DEPLOY_KEY` is listed

2. Verify deploy key is added:
   - Go to: https://github.com/YOUR_USERNAME/waku/settings/keys
   - Check if deploy key exists with write access

3. If missing, follow setup steps above

### Tag already exists error

**Solution:**
```bash
# Delete tag locally (if exists)
git tag -d v1.0.3

# Delete tag on GitHub
git push origin :refs/tags/v1.0.3

# Delete release (if exists)
# Go to: https://github.com/YOUR_USERNAME/waku/releases
# Click on the release → Delete

# Try again with the same or different version
```

### SSH authentication failed

**Error message:** `Permission denied (publickey)`

**Solution:**
1. Regenerate SSH key pair
2. Make sure you copied the **entire** private key (including BEGIN/END lines)
3. Make sure public key is added as deploy key with write access
4. Try again

### Build fails

**Check:**
1. Go to "Actions" tab
2. Click on the failed workflow run
3. Check the logs for specific errors
4. Common issues:
   - Go version mismatch (should be 1.24)
   - Missing dependencies (run `go mod download`)
   - Platform-specific build errors

## Security Notes

### SSH Deploy Key Security

- ✅ **DO**: Store private key in repository secrets (encrypted)
- ✅ **DO**: Use key only for this repository
- ✅ **DO**: Enable write access only if needed
- ✅ **DO**: Delete old keys when rotating
- ❌ **DON'T**: Share private key with anyone
- ❌ **DON'T**: Commit private key to code
- ❌ **DON'T**: Use same key for multiple repositories
- ❌ **DON'T**: Add passphrase (GitHub Actions can't handle it)

### Key Rotation

Rotate SSH keys periodically (e.g., every 6-12 months):

1. Generate new SSH key pair
2. Add new public key as deploy key
3. Update `SSH_DEPLOY_KEY` secret with new private key
4. Test workflow
5. Delete old deploy key

## Alternative: Using Personal Access Token (PAT)

If you prefer PAT over SSH:

1. Create PAT with `repo` and `workflow` scopes
2. Add as `PAT_TOKEN` secret
3. Modify workflow to use PAT instead of SSH

**Note:** SSH Deploy Key is more secure and recommended for CI/CD.

## Support

If you encounter issues:
1. Check workflow logs in Actions tab
2. Verify SSH_DEPLOY_KEY secret is set correctly
3. Verify deploy key has write access
4. Check SSH key format (must include BEGIN/END lines)
5. Open an issue on GitHub

## Summary

✅ **Current Setup (SSH Deploy Key):**
- Generate SSH key pair
- Add public key as deploy key (with write access)
- Add private key as `SSH_DEPLOY_KEY` secret
- Workflows work automatically
- More secure than PAT

✅ **Benefits:**
- ✅ Automatic build triggering
- ✅ No token expiration issues
- ✅ Repository-specific access
- ✅ Better security audit trail
- ✅ No manual intervention needed

🎯 **Result:**
One-click release creation with full automation!

