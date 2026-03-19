# Project Context for AI Agents

## Technology Stack & Versions
- **Language**: Go 1.22+
- **Build Tool**: Go modules (separate go.mod per component)
- **Testing**: Go stdlib `testing`, AAA pattern, TDD RED/GREEN/REFACTOR
- **Kubernetes**: Kustomize components for conditional injection
- **AWS**: AWS SDK v2 (S3 interactions)
- **Database**: SQLite (via Litestream sidecar)

## Critical Implementation Rules

### Language-Specific Rules
- **Naming**: CamelCase for functions, camelCase for variables, kebab-case for K8s resources
- **Modules**: Separate `go.mod` per component (cmd/, internal/, pkg/)
- **Imports**: No cross-module imports without `replace` directives in go.mod
- **Error Handling**: Wrap errors with `fmt.Errorf("context: %w", err)`
- **Logging**: Use structured logging with log levels (INFO, WARNING, ERROR, CRITICAL)

### Testing Rules
- **Pattern**: AAA (Arrange, Act, Assert)
- **TDD Cycle**: RED → GREEN → REFACTOR
- **Coverage**: All public functions must have tests
- **Mocking**: Use interfaces and mock implementations for external dependencies
- **Test Data**: Compute expected values programmatically, don't hardcode

### Anti-patterns & Edge Cases
- **No Type Suppression**: Never use `as any`, `@ts-ignore`, `@ts-expect-error`
- **No Empty Catch Blocks**: Always handle errors properly
- **No Test Deletion**: Fix code, not tests
- **No Partial Implementation**: Complete stories 100% before moving on
- **Atomic Commits**: ~3 commits per story (RED, GREEN, REFACTOR)

## Project Structure

### Key Directories
- `cmd/`: Command-line tools and entry points
  - `cmd/init/`: Init container for Kubernetes
  - `cmd/cli/`: CLI library package
  - `cmd/dataangel-cli/`: CLI entry point
  - `cmd/sidecar-litestream/`: Litestream sidecar
  - `cmd/sidecar-rclone/`: Rclone sidecar
- `internal/`: Internal packages
  - `internal/restore/`: Restore logic and state checking
  - `internal/validation/`: Data validation logic
  - `internal/k8s/`: Kubernetes integration
  - `internal/lock/`: Distributed locking (S3)
- `pkg/`: Public packages
  - `pkg/s3/`: S3 types and interfaces

### Module Dependencies
```
cmd/dataangel-cli
  ├── cmd/cli
  │   └── pkg/s3
  └── pkg/s3

cmd/init
  └── internal/restore
      └── pkg/s3 (via replace)

internal/restore
  └── pkg/s3 (via replace)
```

## Epic 1 Implementation Details

### Story 1.1: Annotations Configuration
- Parser for Kubernetes annotations in `internal/k8s/annotations.go`
- Integration with sidecar-litestream
- Kustomize component for conditional injection

### Story 1.2: Init Container State Detection
- `GetLocalState()`: Reads local file, computes SHA256 checksum
- `CompareStates()`: Compares local vs remote state
- `CheckDataHealth()`: Validates data integrity
- Init container exit codes: 0=skip, 1=restore needed, 2=error

### Story 1.3: Conditional Restore
- `ShouldSkip()`: Determines if restore should be skipped
- `RestoreFromS3()`: Downloads and verifies data integrity
- `VerifyRestoredData()`: Validates checksums
- Mock S3 downloader for testing

### Story 1.4: CLI Verification
- `VerifyBackupState()`: Checks backup status in S3
- `FormatBackupList()`: Formats backup information
- CLI commands: `verify`, `force-release-lock`

## Common Patterns

### S3 Interaction Pattern
```go
type S3Client interface {
    Download(ctx context.Context, bucket, key, destPath string) error
    ListBackups(ctx context.Context, bucket, path string) ([]BackupInfo, error)
}
```

### State Comparison Pattern
```go
type DataState struct {
    Exists    bool
    Checksum  string
    Timestamp time.Time
    Size      int64
    Path      string
}

type RestoreDecision int
const (
    DecisionSkip RestoreDecision = iota
    DecisionRestore
    DecisionCorrupted
)
```

### Init Container Pattern
1. Load configuration from environment variables
2. Get local state
3. Get remote state (via S3)
4. Compare states
5. Execute skip or restore
6. Exit with appropriate code

## Testing Strategy

### Unit Tests
- Each function has dedicated test file
- Mock implementations for external dependencies
- TDD cycle for all new functionality

### Integration Tests
- Test workflows across multiple components
- Use temporary directories for file operations
- Mock S3 operations

### Manual Testing
- Build and run init container with environment variables
- Test CLI commands with mock S3 client
- Verify exit codes and output

## Deployment Considerations

### Kubernetes Init Containers
- Exit code 0: Success (skip or restore completed)
- Exit code 1: Failure (restore needed but failed)
- Exit code 2: Configuration error

### Environment Variables
- `DATA_GUARD_BUCKET`: S3 bucket name
- `DATA_GUARD_PATH`: Remote backup path
- `DATA_GUARD_LOCAL_PATH`: Local data path
- `DATA_GUARD_CHECKSUM`: Expected checksum for verification

## Future Enhancements
1. Real S3 integration (replace mocks)
2. Performance benchmarks
3. End-to-end integration tests
4. Multi-region S3 support
5. Backup encryption
6. Restore rollback capabilities

---
*Generated: 2026-03-17*
*Project: dataGuard (Data-Guard K8s data protection)*