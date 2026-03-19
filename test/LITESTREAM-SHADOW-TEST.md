# Litestream Shadow Directory Exclusion Test

## Issue Addressed

**Issue #12**: rclone sync fails on litestream shadow directory (`.*.db-litestream/`)

## Problem

When litestream replicates a SQLite database, it creates a shadow directory with pattern `.{dbname}.db-litestream/`:

```
/app/data/
├── mealie.db                    ← excluded by --exclude "*.db*" ✅
├── mealie.db-shm                ← excluded ✅
├── mealie.db-wal                ← excluded ✅
└── .mealie.db-litestream/       ← DIRECTORY → not excluded by "*.db*" ❌
    ├── generation
    └── generations/
        └── abc123/
            └── wal/
                └── 00000000.wal  ← open by litestream → rclone fails
```

**Why `--exclude "*.db*"` is not enough:**

In rclone, `--exclude` by default applies to **files only**. To exclude directory **contents**, you need `--exclude "dir/**"`.

## Fix Applied

Added explicit exclusion for litestream shadow directories:

```go
// internal/sidecar/rclone.go (syncOnce)
args := []string{
    "sync",
    srcPath,
    remotePath,
    "--s3-env-auth",
    "--exclude", "*.db*",                // Exclude SQLite files (db, db-shm, db-wal)
    "--exclude", ".*.db-litestream/**",  // Exclude litestream shadow directories
    "--checksum",
    // ...
}
```

Also added same exclusion in restore for consistency (less critical since litestream not active during restore):

```go
// cmd/dataangel/restore.go (restoreFilesystem)
args := []string{
    "copy",
    remotePath,
    fsPath,
    "--s3-env-auth",
    "--exclude", "*.db*",
    "--exclude", ".*.db-litestream/**",  // Consistency with sidecar
    // ...
}
```

## Patterns Covered

| Pattern | Covers | Why |
|---------|--------|-----|
| `*.db*` | SQLite files | `app.db`, `app.db-shm`, `app.db-wal`, `app.db-journal` |
| `.*.db-litestream/**` | Litestream shadow dirs | `.app.db-litestream/`, `.mealie.db-litestream/`, etc. |

**Generic patterns work for any database name** (app.db, mealie.db, home-assistant_v2.db, etc.)

## Test Procedure

### Setup

Deploy an app with both SQLite and filesystem paths on the same volume:

```yaml
annotations:
  dataangel.io/sqlite-paths: "/app/data/mealie.db"
  dataangel.io/fs-paths: "/app/data"  # Same volume as SQLite
```

**Expected:** Litestream creates `.mealie.db-litestream/` in `/app/data/`

### Test 1: Verify shadow directory exists

```bash
kubectl exec -it <pod> -c dataangel -- ls -la /app/data/

# Expected output:
# drwxr-xr-x    5 911      911            160 Mar 19 12:00 .
# drwxr-xr-x    3 root     root          4096 Mar 19 11:55 ..
# -rw-r--r--    1 911      911         524288 Mar 19 12:00 mealie.db
# -rw-r--r--    1 911      911          32768 Mar 19 12:00 mealie.db-shm
# -rw-r--r--    1 911      911              0 Mar 19 12:00 mealie.db-wal
# drwxr-xr-x    3 911      911             96 Mar 19 12:00 .mealie.db-litestream  ← shadow
```

### Test 2: Verify rclone sync succeeds

**Before fix:** rclone sync would fail with `exit status 1` because litestream holds open files in `.mealie.db-litestream/wal/`

**After fix:** rclone sync should succeed and exclude the shadow directory

```bash
kubectl logs -f <pod> -c dataangel | grep -E "(rclone sync|error)"

# Expected (no errors):
# [rclone] Running sync...
# dataguard_rclone_syncs_total increments
# dataguard_rclone_up = 1
```

### Test 3: Verify shadow directory NOT in S3

```bash
# Check S3 bucket contents
aws s3 ls s3://bucket/filesystem/ --recursive | grep litestream

# Expected: NO .mealie.db-litestream/ in S3
# Only filesystem files (excluding *.db*)
```

### Test 4: Verify DB files NOT in S3 filesystem backup

```bash
aws s3 ls s3://bucket/filesystem/ --recursive | grep -E "\\.db"

# Expected: NO .db, .db-shm, .db-wal files in S3 filesystem/
# SQLite is backed up separately by litestream to s3://bucket/mealie.db
```

## Success Criteria

- ✅ Shadow directory `.*.db-litestream/` exists in local filesystem
- ✅ rclone sync completes without errors
- ✅ Shadow directory NOT present in S3 `filesystem/` path
- ✅ SQLite files (*.db*) NOT present in S3 `filesystem/` path
- ✅ SQLite backed up separately via litestream to S3 root (s3://bucket/dbname.db)
- ✅ Filesystem files (non-DB) synced to S3 `filesystem/`
- ✅ Metrics show `dataguard_rclone_syncs_total` incrementing
- ✅ Metrics show `dataguard_rclone_syncs_failed_total` = 0

## Edge Cases

### Multiple databases in same directory

```
/data/
├── app1.db
├── .app1.db-litestream/
├── app2.db
└── .app2.db-litestream/
```

**Pattern `.*.db-litestream/**` covers both shadow directories.**

### Nested directories with databases

```
/data/
├── users/
│   ├── users.db
│   └── .users.db-litestream/
└── posts/
    ├── posts.db
    └── .posts.db-litestream/
```

**Pattern works recursively** - all `.*.db-litestream/**` paths excluded at any depth.

## Rationale

**Why two exclusions?**

1. `--exclude "*.db*"` - Prevents SQLite files from being synced via rclone (they're already handled by litestream's separate replication)
2. `--exclude ".*.db-litestream/**"` - Prevents litestream's internal shadow directory from being synced (it contains open files and internal state)

**Why `/**` suffix?**

In rclone:
- `--exclude "pattern"` - Excludes matching files
- `--exclude "dir/**"` - Excludes directory AND all its contents recursively

Without `/**`, only the directory entry would be excluded, but not its contents.

## Backup Strategy Reminder

DataAngel uses a **dual backup strategy**:

1. **Litestream** (SQLite databases):
   - Continuous replication
   - Point-in-time recovery
   - S3 path: `s3://bucket/{dbname}.db`

2. **Rclone** (Filesystem):
   - Periodic sync
   - Excludes SQLite files (already handled by litestream)
   - S3 path: `s3://bucket/filesystem/`

**No overlap** - each tool handles its own domain without conflicts.
