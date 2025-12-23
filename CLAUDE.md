# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

s3cli is a command-line tool for AWS S3 compatible storage services, written in Go (1.21+). It uses the Cobra CLI framework for command handling and AWS SDK Go v1 for S3 operations.

## Common Commands

### Building
```bash
make                # Build for current platform
make pkg            # Build for all platforms (Linux/macOS/Windows, amd64/arm64)
make image          # Build Docker image
make install        # Install to /usr/local/bin/
make clean          # Clean build artifacts
```

### Testing
```bash
make test           # Run unit tests (go test ./...)
make testcli        # Run integration tests (requires Min.io play server)
make vet            # Run static analysis (go vet ./...)
```

### OpenSpec Workflow
```bash
openspec list               # List active changes
openspec list --specs       # List specifications
openspec show <item>        # Display change or spec details
openspec validate <id> --strict   # Validate change
openspec archive <id> --yes # Archive completed change
```

## Architecture

### Entry Point
- `main.go` (~1,100 lines) - Cobra CLI setup with 40+ commands
- All commands use persistent flags (endpoint, credentials, region, output format)

### Core Components
- `s3cli.go` - S3Cli struct with all S3 operation methods
- `v2.go` - AWS Signature Version 2 authentication (alternative to V4)
- `main.go` - Command definitions and client initialization

### Key Patterns

**Configuration Precedence:**
1. Command-line flags (e.g., `--endpoint`, `--ak`, `--sk`)
2. Environment variables (`S3_ENDPOINT`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
3. AWS credentials file (when using `--profile`)

**Authentication:**
- AWS Signature V4 (default)
- AWS Signature V2 (with `--v2sign` flag)
- Custom endpoint support via `--endpoint` flag
- Virtual-hosted style vs path-style addressing (controlled by `virtualHostStyle` var)

**Command Categories:**
- Bucket operations: `create-bucket`, `ls`, `policy`, `acl`, `version`, `delete`
- Object operations: `upload`/`put`, `download`/`get`, `list`/`ls`, `delete`/`rm`
- Multi-part upload: `mpu-init`, `mpu-upload`, `mpu-complete`, `mpu-abort`, `mpu-list`
- Presigned URLs: available on `put`, `download`, `delete` commands with `--presign` flag

## OpenSpec Integration

This project uses OpenSpec for spec-driven development. Always refer to `openspec/AGENTS.md` when:

- Creating change proposals for new features
- Planning breaking changes or architecture shifts
- Working on performance or security improvements

**Three-stage workflow:**
1. **Proposal** - Create change in `openspec/changes/<id>/` with proposal.md, tasks.md, and spec deltas
2. **Apply** - Implement changes after approval, tracking tasks as TODOs
3. **Archive** - Move completed changes to `changes/archive/` and update specs

**Change ID format:** kebab-case, verb-led (`add-`, `update-`, `remove-`, `refactor-`)

**Spec delta format:** Use `## ADDED|MODIFIED|REMOVED Requirements` with at least one `#### Scenario:` per requirement

See `.clinerules/workflows/` for detailed workflow instructions.

## Environment Variables

- `S3_ENDPOINT` - S3 server URL
- `AWS_ACCESS_KEY_ID` / `AWS_ACCESS_KEY` - Access key
- `AWS_SECRET_ACCESS_KEY` / `AWS_SECRET_KEY` - Secret key
- `AWS_SESSION_TOKEN` - Session token
- `AWS_REGION` - AWS region

## Testing Notes

Integration tests (`test.sh`) run against Min.io play server and use bucket "s3cli" for testing. Tests cover:
- Bucket CRUD operations
- Object upload/download with various options
- Presigned URLs (V2 and V4)
- Multi-part uploads
- CORS configuration
- Object lock

## Code Locations

- Client initialization: `main.go:54` (`newS3Client()`)
- S3Cli struct definition: `s3cli.go:39`
- V2 signing implementation: `v2.go`
- Error handling: `s3cli.go` (`errorHandler()`)
- Presigned URL logic: `s3cli.go` (`presignV2Raw()`, `presignV4Raw()`)
