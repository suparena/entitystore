# EntityStore Release Process

This document describes the release process for EntityStore.

## Prerequisites

- Go 1.21 or later installed
- Git configured with signing capability
- GitHub CLI (`gh`) installed (optional but recommended)
- Docker installed (for integration tests)

## Release Types

- **Patch Release** (0.0.X): Bug fixes and minor improvements
- **Minor Release** (0.X.0): New features, backward compatible
- **Major Release** (X.0.0): Breaking changes

## Release Process

### 1. Prepare Release

#### Update Version
```bash
# For patch release (e.g., 0.1.0 → 0.1.1)
make version-bump-patch

# For minor release (e.g., 0.1.0 → 0.2.0)
make version-bump-minor

# For major release (e.g., 0.1.0 → 1.0.0)
make version-bump-major
```

#### Update CHANGELOG
1. Edit `CHANGELOG.md`
2. Move items from "Unreleased" to the new version section
3. Add release highlights
4. Review and organize changes by category

### 2. Test Release

#### Run Full Test Suite
```bash
# Run all tests including integration
make clean
make lint
make test-race
make test-coverage
make test-integration  # Requires DynamoDB Local
```

#### Test Build
```bash
# Test building for all platforms
make build-all

# Test the version output
./bin/indexmap-pps --version
```

#### Dry Run
```bash
# Preview what would happen
make release-dry-run
```

### 3. Commit Changes

```bash
# Stage all changes
git add -A

# Commit with version bump message
git commit -m "Bump version to $(cat VERSION)"

# Push to main branch
git push origin main
```

### 4. Create Release

#### Option A: Using Make (Recommended)
```bash
# This will:
# - Verify prerequisites
# - Run tests
# - Build all platforms
# - Create git tag
make release

# Push the tag
git push origin v$(cat VERSION)
```

#### Option B: Using GitHub CLI
```bash
# Create release with GitHub CLI
gh release create v$(cat VERSION) \
  --title "EntityStore v$(cat VERSION)" \
  --notes-file CHANGELOG.md \
  --target main
```

#### Option C: Manual Process
```bash
# Create and push tag
git tag -a v$(cat VERSION) -m "Release version $(cat VERSION)"
git push origin v$(cat VERSION)

# Then create release on GitHub UI
```

### 5. Post-Release

1. **Verify Release**
   - Check GitHub Releases page
   - Verify all binaries are uploaded
   - Test downloading and running binaries

2. **Update Documentation**
   - Update installation instructions if needed
   - Update version references in docs

3. **Announce Release**
   - Create release announcement if major version
   - Update project README if needed

## Local Release

For testing or internal distribution:

```bash
# Create local release artifacts
make release-local

# This creates:
# - dist/entitystore-X.Y.Z.tar.gz
# - dist/entitystore-X.Y.Z.zip
```

## Automation

The GitHub Actions workflow automatically:
1. Builds binaries for all platforms
2. Runs tests
3. Creates GitHub release
4. Uploads artifacts
5. Generates checksums

## Version Information

Version information is embedded in binaries:

```go
// Set by build flags
Version   = "0.1.0"
GitCommit = "abc123"
BuildDate = "2025-01-18T..."
GoVersion = "go1.22"
```

Access via:
```bash
indexmap-pps --version
```

## Troubleshooting

### Tag Already Exists
```bash
# Delete local tag
git tag -d v0.1.0

# Delete remote tag (be careful!)
git push origin :refs/tags/v0.1.0
```

### Build Failures
```bash
# Clean and retry
make clean
make deps
make build-all
```

### Version Mismatch
Ensure VERSION file, version.go, and git tag all match.

## Security Considerations

1. **Sign Commits**: Use GPG signing for release commits
2. **Verify Dependencies**: Run `make deps-verify` before release
3. **Security Scan**: Run `make security` before release
4. **Checksums**: Always generate and verify checksums