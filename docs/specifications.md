Data-Guard : Architecture & Specification

1. Executive Summary

1.1 Purpose

Data-Guard is a Kubernetes-native data protection system designed for stateful applications running on iSCSI-backed PVCs. It provides automated backup, corruption detection, and conditional restoration to ensure data survival across node failures, PVC deletion, and application corruption.

1.2 Key Characteristics

Attribute

Value

Deployment Model

Sidecar/Init container pattern

Configuration

Kubernetes annotations

Storage Backend

S3-compatible (MinIO)

Supported Workloads

SQLite databases, filesystem state

Recovery RTO

< 5 minutes (conditional), < 2 minutes (skip)

Recovery RPO

Continuous (Litestream), 60s (Rclone)

Memory Budget

< 128MB combined

1.3 Non-Goals

Point-in-time recovery automation (manual CLI available)

Multi-writer active-active replication

Database engines other than SQLite (PostgreSQL, MySQL use native mechanisms)

Cross-cluster automated disaster recovery

2. Context & Constraints

2.1 Infrastructure Context

Kubernetes cluster with ArgoCD

iSCSI CSI storage backend

MinIO S3 local instance

Infisical for secrets management

Reloader for secret rotation handling

2.2 Constraints

Constraint

Implication

No Helm

Pure Kustomize implementation required

Paranoia on data

Blocking behavior preferred over data fork

Manual recovery acceptable

Complex recovery scenarios require human intervention

Single developer maintenance

Simplicity over feature completeness

2.3 Target Applications

Home Assistant (SQLite + YAML configs)

Paperless-ngx (SQLite + documents)

Immich (PostgreSQL excluded - native backup)

Other SQLite-based stateful applications

3. Requirements

3.1 Functional Requirements

ID

Requirement

Priority

FR-01

Detect healthy local data vs S3 backup

Must

FR-02

Skip restore if local data is valid and newer

Must

FR-03

Restore automatically if local data missing/corrupted

Must

FR-04

Validate SQLite integrity before backup/restore

Must

FR-05

Validate YAML syntax before filesystem backup

Must

FR-06

Block startup if restore required but S3 unavailable

Must

FR-07

Continuous replication for SQLite (Litestream)

Must

FR-08

Periodic sync for filesystem (Rclone, 60s)

Must

FR-09

Prevent multi-replica write conflicts

Must

FR-10

Expose Prometheus metrics

Should

FR-11

CLI tools for manual troubleshooting

Should

FR-12

Graceful shutdown with WAL flush

Must

FR-13

Atomic state file writes

Must

3.2 Non-Functional Requirements

ID

Requirement

Target

NFR-01

Init container execution time

< 30s (skip), < 5min (restore)

NFR-02

Image size

< 200MB

NFR-03

Memory overhead (sidecars)

< 128MB combined

NFR-04

CPU overhead (sidecars)

< 0.1 core average

NFR-05

Availability during S3 outage

0% (blocking by design)

NFR-06

YAML validation CPU time

< 50ms (cached), < 5s (full)

3.3 Scenarios Addressed

Scenario

Mechanism

Pod rescheduling to new node

PVC follows (iSCSI), init detects healthy data, skip restore

kubectl delete pvc

PVC recreated empty, init detects absence, full restore

SQLite corruption

integrity_check fails, restore from S3

YAML truncation

Validation fails, backup skipped, corruption not propagated

Application bug corrupting data

Crash loop detected via metrics, manual intervention required

Rolling update

Graceful shutdown flushes WAL, new pod starts with skip

Node crash

S3 lock expires after TTL, new pod acquires lock and restores

Secret rotation

Reloader restarts pod, data-guard reconnects with new creds



4. Architecture Overview

4.1 High-Level Design (Mermaid)

flowchart TB
   subgraph "Kubernetes Pod"
       direction TB
       
       subgraph "Metadata"
           ANNOT["Annotations
           dataangel.io/enabled: true
           dataangel.io/bucket: app-name
           dataangel.io/sqlite-paths: /data/db
           dataangel.io/fs-paths: /config"]
       end
       
       subgraph "Init Phase"
           INIT["data-guard init
           ─────────────────
           1. Acquire S3 lock
           2. Parse annotations
           3. Check SQLite integrity
           4. Compare generation (WAL salt)
           5. Check FS manifest
           6. Conditional restore
           7. Write state.json (atomic)
           8. Release lock"]
       end
       
       subgraph "Runtime Phase"
           APP["Application
           (Home Assistant/Paperless)"]
           
           subgraph "Data-Guard Sidecar"
               SYNC["SyncOrchestrator
               ─────────────────
               • Litestream replicate
               • Rclone 60s loop
               • YAML validation (cached)
               • Metrics :9090
               • SIGTERM handler"]
           end
       end
       
       subgraph "Shared Storage"
           PVC[("PVC iSCSI
           /config /data")]
           EMPTY{{"emptyDir
           /tmp/data-guard
           (state.json, yaml_cache)"}}
       end
   end
   
   subgraph "External Services"
       S3[("MinIO S3
       Backups & Lock")]
       INF["Infisical
       Secrets"]
       REL["Reloader
       Secret rotation"]
       PROM["Prometheus
       Monitoring"]
   end
   
   ANNOT --> INIT
   INIT -->|"Read/Write"| PVC
   INIT -->|"Acquire/Release"| S3
   INIT -->|"Atomic write"| EMPTY
   
   INIT -->|"Success"| APP
   INIT -->|"Start"| SYNC
   
   APP -->|"Read/Write"| PVC
   
   SYNC -->|"Replicate"| S3
   SYNC -->|"Sync"| S3
   SYNC -->|"Read state"| EMPTY
   SYNC -.->|"Scrape"| PROM
   
   REL -.->|"Restart on change"| APP
   REL -.->|"Restart on change"| SYNC
   
   style INIT fill:#e1f5fe,stroke:#01579b,stroke-width:2px
   style SYNC fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px
   style S3 fill:#fff3e0,stroke:#e65100,stroke-width:2px
   style PVC fill:#fce4ec,stroke:#880e4f,stroke-width:2px


4.2 Data Flow (Mermaid)

flowchart TB
   subgraph "Continuous Backup"
       direction LR
       A1["App Write"] --> P1[("PVC iSCSI")]
       P1 --> L1["Litestream
       replicate"]
       P1 --> R1["Rclone
       sync 60s"]
       L1 --> S1[("S3 Bucket")]
       R1 --> S1
   end
   
   subgraph "Startup Recovery"
       direction TB
       I1["data-guard init"] --> C1{"Data exists?"}
       C1 -->|"No"| R2["Restore all
       from S3"]
       C1 -->|"Yes"| C2{"Integrity OK?"}
       C2 -->|"No"| R3["Restore corrupted
       from S3"]
       C2 -->|"Yes"| C3{"Generation?"}
       C3 -->|"Local < S3"| R4["Restore outdated
       from S3"]
       C3 -->|"Local >= S3"| S2["Skip restore"]
       
       R2 --> D1["App start"]
       R3 --> D1
       R4 --> D1
       S2 --> D1
   end
   
   S1 -.->|"Source"| R2
   S1 -.->|"Source"| R3
   S1 -.->|"Source"| R4
   
   style I1 fill:#e1f5fe,stroke:#01579b,stroke-width:2px
   style C1 fill:#fff9c4,stroke:#f57f17,stroke-width:2px
   style C2 fill:#fff9c4,stroke:#f57f17,stroke-width:2px
   style C3 fill:#fff9c4,stroke:#f57f17,stroke-width:2px
   style R2 fill:#ffccbc,stroke:#bf360c,stroke-width:2px
   style R3 fill:#ffccbc,stroke:#bf360c,stroke-width:2px
   style R4 fill:#ffccbc,stroke:#bf360c,stroke-width:2px
   style S2 fill:#c8e6c9,stroke:#1b5e20,stroke-width:2px
   style D1 fill:#a5d6a7,stroke:#1b5e20,stroke-wi

4.3 State Machine (Init Container)

stateDiagram-v2
   [*] --> ParseConfig: Start
   
   ParseConfig --> AcquireLock: Config valid
   ParseConfig --> [*]: Disabled/Invalid
   
   AcquireLock --> CheckSQLite: Lock acquired
   AcquireLock --> [*]: Lock failed (other instance)
   
   CheckSQLite --> CheckIntegrity: DB exists
   CheckSQLite --> RestoreSQLite: DB missing
   
   CheckIntegrity --> CompareGen: Integrity OK
   CheckIntegrity --> RestoreSQLite: Corrupted
   
   CompareGen --> CheckFS: Local >= S3
   CompareGen --> RestoreSQLite: Local < S3
   
   RestoreSQLite --> CheckFS: Restore OK
   RestoreSQLite --> [*]: Restore failed (S3 down)
   
   CheckFS --> CompareManifest: Paths exist
   CheckFS --> RestoreFS: Paths empty
   
   CompareManifest --> WriteState: Match
   CompareManifest --> RestoreFS: Diverge
   
   RestoreFS --> WriteState: Restore OK
   RestoreFS --> [*]: Restore failed
   
   WriteState --> ReleaseLock
   
   ReleaseLock --> [*]: Success
   
   note right of [*]
       Exit 0: Continue to app
       Exit 1: CrashLoopBackOff
   end note


4.4 Sequence: Graceful Shutdown

sequenceDiagram
   participant K as Kubernetes
   participant S as SyncOrchestrator
   participant L as Litestream
   participant R as Rclone
   participant S3 as MinIO S3
   
   K->>S: SIGTERM (Rolling Update)
   S->>S: Set running=False
   S->>L: SIGTERM
   L->>L: Flush WAL to S3
   L->>S3: Final replication
   L-->>S: Exit 0
   S->>R: SIGTERM (if running)
   R-->>S: Exit (may leave .part)
   S-->>K: Exit 0
   
   note over S: Max 15s wait for Litestream<br/>Kubelet sends SIGKILL after 30s


4.5 Generation Tracking

Resource

Primary Method

Fallback

Comparison

SQLite

WAL salt (header bytes 16-24)

WAL size+mtime

Integer comparison

SQLite (no WAL)

mtime only

-

Timestamp comparison

Filesystem

mtime+size cache

SHA256 (if mtime match, size diff)

Hash comparison



5. Component Specification

5.1 Container Image

Name: ghcr.io/truxonline/data-guard
Base: Alpine Linux 3.19
Size Target: < 200MB
User: data-guard (UID 1000, non-root)

Included Binaries:

Litestream 0.5.9 (static binary)

Rclone 1.73 (static binary)

Python 3.11 + deps (user site-packages in /home/data-guard/.local)

5.2 Annotations Specification

Annotation

Required

Default

Description

dataangel.io/enabled

Yes

-

Must be "true" to activate

dataangel.io/bucket

Yes

-

S3 bucket/prefix for this app

dataangel.io/sqlite-paths

No

-

Comma-separated list of DB paths

dataangel.io/fs-paths

No

-

Comma-separated list of directory paths

dataangel.io/s3-endpoint

No

From env

Override S3 endpoint URL

dataangel.io/full-logs

No

"false"

Verbose logging for troubleshooting

5.3 Init Container: Detailed Logic

#!/usr/bin/env python3
"""
data_guard/init.py - Initialization logic
"""

import os
import sys
import json
import time
import struct
import sqlite3
import hashlib
import tempfile
import subprocess
from pathlib import Path
from dataclasses import dataclass
from typing import List, Optional, Dict, Tuple

import boto3
from botocore.config import Config
from botocore.exceptions import ClientError

@dataclass
class Config:
   enabled: bool
   bucket: str
   sqlite_paths: List[str]
   fs_paths: List[str]
   s3_endpoint: Optional[str]
   full_logs: bool

class S3Lock:
   """Distributed lock using S3 with TTL and steal mechanism."""
   
   def __init__(self, bucket: str, instance_id: str, ttl: int = 300):
       self.bucket = bucket
       self.lock_key = ".data-guard/lock"
       self.instance_id = instance_id
       self.ttl = ttl
       
       # Fail-fast S3 configuration for Init Container
       s3_config = Config(
           connect_timeout=5,
           read_timeout=10,
           retries={'max_attempts': 2}
       )
       
       self.s3 = boto3.client(
           's3',
           endpoint_url=os.environ.get('S3_ENDPOINT'),
           aws_access_key_id=os.environ['S3_ACCESS_KEY_ID'],
           aws_secret_access_key=os.environ['S3_SECRET_ACCESS_KEY'],
           config=s3_config
       )
   
   def acquire(self) -> bool:
       """Try to acquire lock. Steal if expired."""
       try:
           # Try to create lock file
           self.s3.put_object(
               Bucket=self.bucket,
               Key=self.lock_key,
               Body=self.instance_id.encode(),
               Metadata={
                   'timestamp': str(time.time()),
                   'instance': self.instance_id
               }
           )
           return True
           
       except ClientError as e:
           if e.response['Error']['Code'] == 'PreconditionFailed':
               return self._steal_if_expired()
           raise
   
   def _steal_if_expired(self) -> bool:
       """Check if existing lock is expired and steal it."""
       try:
           obj = self.s3.head_object(Bucket=self.bucket, Key=self.lock_key)
           lock_time = float(obj['Metadata'].get('timestamp', 0))
           
           if time.time() - lock_time > self.ttl:
               # Lock expired, delete and retry
               self.s3.delete_object(Bucket=self.bucket, Key=self.lock_key)
               return self.acquire()
               
       except Exception:
           pass
       
       return False
   
   def release(self):
       """Release lock if we own it."""
       try:
           obj = self.s3.head_object(Bucket=self.bucket, Key=self.lock_key)
           if obj['Metadata'].get('instance') == self.instance_id:
               self.s3.delete_object(Bucket=self.bucket, Key=self.lock_key)
       except Exception:
           pass

class SQLiteManager:
   """SQLite operations with WAL-aware generation tracking."""
   
   @staticmethod
   def ensure_wal_mode(db_path: str):
       """Force WAL mode for Litestream compatibility."""
       conn = sqlite3.connect(db_path)
       current = conn.execute("PRAGMA journal_mode;").fetchone()[0]
       if current != "wal":
           conn.execute("PRAGMA journal_mode=WAL;")
       conn.close()
   
   @staticmethod
   def integrity_check(db_path: str, quick: bool = False) -> bool:
       """Check SQLite integrity."""
       pragma = "PRAGMA quick_check;" if quick else "PRAGMA integrity_check;"
       try:
           conn = sqlite3.connect(db_path)
           result = conn.execute(pragma).fetchone()[0]
           conn.close()
           return result == "ok"
       except Exception:
           return False
   
   @staticmethod
   def get_generation(db_path: str) -> int:
       """
       Hierarchical generation detection:
       1. WAL salt (changes every checkpoint)
       2. WAL size + mtime
       3. DB file mtime (fallback)
       """
       wal_path = db_path + "-wal"
       
       # Priority 1: WAL salt from header
       if os.path.exists(wal_path):
           try:
               with open(wal_path, 'rb') as f:
                   header = f.read(32)
                   if len(header) == 32:
                       # Salt at offset 16-24 (8 bytes)
                       salt = struct.unpack('>II', header[16:24])
                       return salt[0]  # First 4 bytes of salt
           except Exception:
               pass
           
           # Priority 2: WAL size + mtime
           stat = os.stat(wal_path)
           return (stat.st_size << 32) | int(stat.st_mtime)
       
       # Priority 3: DB file mtime
       return int(os.path.getmtime(db_path))

class FilesystemManager:
   """Filesystem operations with caching."""
   
   @staticmethod
   def generate_manifest(paths: List[str]) -> Dict[str, Dict]:
       """Generate manifest with mtime+size (fast) and hash (on demand)."""
       manifest = {}
       for path_pattern in paths:
           for filepath in glob.iglob(path_pattern, recursive=True):
               if not os.path.isfile(filepath):
                   continue
               
               stat = os.stat(filepath)
               manifest[filepath] = {
                   "mtime": stat.st_mtime,
                   "size": stat.st_size,
                   "key": f"{stat.st_mtime}:{stat.st_size}"
               }
       return manifest

def write_json_atomic(data: dict, filepath: str):
   """Atomic write using temp file + rename."""
   dir_name = os.path.dirname(filepath)
   
   with tempfile.NamedTemporaryFile(
       mode='w',
       dir=dir_name,
       suffix='.tmp',
       delete=False
   ) as tf:
       json.dump(data, tf)
       temp_path = tf.name
   
   os.replace(temp_path, filepath)

def main():
   """Main init logic."""
   config = parse_annotations()
   if not config.enabled:
       sys.exit(0)
   
   # Instance ID from hostname
   instance_id = f"{os.environ.get('HOSTNAME', 'unknown')}-{int(time.time())}"
   
   # Acquire S3 lock
   lock = S3Lock(config.bucket, instance_id)
   if not lock.acquire():
       print("ERROR: Cannot acquire S3 lock", file=sys.stderr)
       sys.exit(1)
   
   try:
       state = {
           "schema_version": "1",
           "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
           "instance": instance_id,
           "restore_count": 0,
           "restores": []
       }
       
       # Process SQLite databases
       for db_path in config.sqlite_paths:
           action = process_sqlite(db_path, config.bucket, state)
           state["restores"].append(action)
       
       # Process filesystem paths
       if config.fs_paths:
           action = process_filesystem(config.fs_paths, config.bucket, state)
           state["restores"].append(action)
       
       # Circuit breaker
       if state["restore_count"] > 3:
           print("WARNING: Excessive restores detected", file=sys.stderr)
       
       # Write state atomically
       write_json_atomic(state, "/tmp/data-guard/state.json")
       
   finally:
       lock.release()

def process_sqlite(db_path: str, bucket: str, state: dict) -> dict:
   """Process single SQLite database."""
   result = {
       "type": "sqlite",
       "path": db_path,
       "action": "skip",
       "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ")
   }
   
   # Missing: restore
   if not os.path.exists(db_path):
       restore_sqlite(db_path, bucket)
       result["action"] = "restore_missing"
       state["restore_count"] += 1
       return result
   
   # Ensure WAL mode
   SQLiteManager.ensure_wal_mode(db_path)
   
   # Corrupted: restore
   if not SQLiteManager.integrity_check(db_path, quick=True):
       restore_sqlite(db_path, bucket)
       result["action"] = "restore_corrupted"
       state["restore_count"] += 1
       return result
   
   # Compare generation
   local_gen = SQLiteManager.get_generation(db_path)
   s3_gen = get_s3_generation(bucket, db_path)
   
   if local_gen < s3_gen:
       restore_sqlite(db_path, bucket)
       result["action"] = "restore_outdated"
       result["from_generation"] = s3_gen
       result["to_generation"] = local_gen
       state["restore_count"] += 1
   else:
       result["local_generation"] = local_gen
       result["s3_generation"] = s3_gen
   
   return result

def process_filesystem(paths: List[str], bucket: str, state: dict) -> dict:
   """Process filesystem paths."""
   result = {
       "type": "filesystem",
       "paths": paths,
       "action": "skip",
       "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ")
   }
   
   # Check if paths exist and have content
   has_data = any(
       os.path.exists(p) and any(os.scandir(p))
       for p in paths
   )
   
   if not has_data:
       restore_fs(paths, bucket)
       result["action"] = "restore_empty"
       state["restore_count"] += 1
       return result
   
   # Compare manifest with S3
   local_manifest = FilesystemManager.generate_manifest(paths)
   
   if not fs_matches_s3(bucket, local_manifest):
       restore_fs(paths, bucket)
       result["action"] = "restore_diverged"
       state["restore_count"] += 1
   
   return result



5.4 Sidecar: SyncOrchestrator

#!/usr/bin/env python3
"""
data_guard/orchestrator.py - Unified sidecar for SQLite and FS sync
"""

import os
import sys
import json
import time
import signal
import subprocess
import threading
import glob
import yaml
import hashlib
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Dict, List, Optional

import logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MetricsHandler(BaseHTTPRequestHandler):
   """Simple Prometheus metrics endpoint."""
   
   def do_GET(self):
       if self.path == '/metrics':
           self.send_response(200)
           self.send_header('Content-Type', 'text/plain')
           self.end_headers()
           
           metrics = self.server.orchestrator.get_metrics()
           self.wfile.write(metrics.encode())
       else:
           self.send_response(404)
           self.end_headers()
   
   def log_message(self, format, *args):
       # Suppress default logging
       pass

class SyncOrchestrator:
   """
   Unified sidecar managing:
   - Litestream replication (subprocess)
   - Rclone sync loop (subprocess)
   - YAML validation with caching
   - Prometheus metrics
   - Graceful shutdown
   """
   
   def __init__(self, state_path: str = "/tmp/data-guard/state.json"):
       self.state_path = state_path
       self.running = True
       self.litestream_proc: Optional[subprocess.Popen] = None
       self.rclone_proc: Optional[subprocess.Popen] = None
       
       # Caches
       self.yaml_cache_path = "/tmp/data-guard/yaml_cache.json"
       self.yaml_cache: Dict[str, Dict] = {}
       self.last_full_validation = 0
       self.validation_interval = 60  # Seconds between validations
       
       # Metrics
       self.metrics = {
           'litestream_up': 0,
           'rclone_last_sync': 0,
           'rclone_errors': 0,
           'yaml_validation_time': 0,
       }
       
       # Load state
       self.state = self._load_state()
       
       # Setup signal handlers
       signal.signal(signal.SIGTERM, self._handle_shutdown)
       signal.signal(signal.SIGINT, self._handle_shutdown)
   
   def _load_state(self) -> dict:
       """Load state from init container."""
       try:
           with open(self.state_path) as f:
               return json.load(f)
       except Exception as e:
           logger.error(f"Cannot load state: {e}")
           return {}
   
   def _handle_shutdown(self, signum, frame):
       """Graceful shutdown handler."""
       logger.info(f"Signal {signum} received, shutting down gracefully...")
       self.running = False
       
       # Stop Litestream gracefully (flush WAL)
       if self.litestream_proc and self.litestream_proc.poll() is None:
           logger.info("Stopping Litestream...")
           self.litestream_proc.terminate()
           try:
               self.litestream_proc.wait(timeout=15)
               logger.info("Litestream stopped cleanly")
           except subprocess.TimeoutExpired:
               logger.warning("Litestream timeout, killing")
               self.litestream_proc.kill()
       
       # Stop Rclone if running
       if self.rclone_proc and self.rclone_proc.poll() is None:
           self.rclone_proc.terminate()
           try:
               self.rclone_proc.wait(timeout=5)
           except subprocess.TimeoutExpired:
               self.rclone_proc.kill()
       
       sys.exit(0)
   
   def _load_yaml_cache(self) -> Dict[str, Dict]:
       """Load YAML validation cache."""
       if os.path.exists(self.yaml_cache_path):
           try:
               with open(self.yaml_cache_path) as f:
                   return json.load(f)
           except Exception:
               pass
       return {}
   
   def _save_yaml_cache(self, cache: Dict[str, Dict]):
       """Save YAML validation cache atomically."""
       import tempfile
       import os
       
       dir_name = os.path.dirname(self.yaml_cache_path)
       with tempfile.NamedTemporaryFile(
           mode='w', dir=dir_name, suffix='.tmp', delete=False
       ) as tf:
           json.dump(cache, tf)
           temp_path = tf.name
       
       os.replace(temp_path, self.yaml_cache_path)
   
   def validate_yaml_cached(self, paths: List[str]) -> bool:
       """
       Validate YAML files with mtime+size caching.
       Only re-validate changed files.
       """
       start_time = time.time()
       
       # Load previous cache
       cache = self._load_yaml_cache()
       new_cache = {}
       files_to_validate = []
       
       for path_pattern in paths:
           for filepath in glob.iglob(path_pattern, recursive=True):
               if not os.path.isfile(filepath):
                   continue
               
               # Skip binary files
               if not filepath.endswith(('.yaml', '.yml', '.json')):
                   continue
               
               stat = os.stat(filepath)
               current_key = f"{stat.st_mtime}:{stat.st_size}"
               
               # Check if unchanged
               cached = cache.get(filepath)
               if cached and cached.get("key") == current_key:
                   new_cache[filepath] = cached
                   continue
               
               # Need validation
               files_to_validate.append((filepath, current_key))
       
       # Validate changed files
       errors = []
       for filepath, current_key in files_to_validate:
           try:
               with open(filepath, 'r', encoding='utf-8') as f:
                   content = f.read()
                   
                   if filepath.endswith('.json'):
                       import json
                       json.loads(content)
                   else:
                       yaml.safe_load(content)
               
               # Success: add to cache with checksum for corruption detection
               new_cache[filepath] = {
                   "key": current_key,
                   "mtime": os.stat(filepath).st_mtime,
                   "size": os.stat(filepath).st_size,
                   "checksum": hashlib.sha256(content.encode()).hexdigest()[:16]
               }
               
           except Exception as e:
               errors.append(f"{filepath}: {e}")
               logger.error(f"YAML/JSON invalid: {filepath}: {e}")
       
       # Save updated cache
       self._save_yaml_cache(new_cache)
       
       # Update metrics
       validation_time = time.time() - start_time
       self.metrics['yaml_validation_time'] = validation_time
       logger.info(f"YAML validation: {len(files_to_validate)} files in {validation_time:.3f}s")
       
       if errors:
           logger.error(f"Validation failed for {len(errors)} files")
           return False
       
       return True
   
   def generate_rclone_excludes(self) -> List[str]:
       """
       Generate dynamic excludes based on sqlite_paths.
       Prevents silent data loss from unmanaged DB files.
       """
       base_excludes = [
           "*.log",
           "tts/**",
           "backups/**", "tmp_backups/**",
           "lost+found/**",
           ".*-litestream", ".*-litestream/**",
           "*.tmp", "*.temp",
           ".~lock.*", "*.part",
           "**/tmp/**", "**/cache/**",
       ]
       
       # Specific excludes for Litestream-managed DBs
       sqlite_paths = self.state.get('sqlite', {}).get('paths', [])
       for db_path in sqlite_paths:
           basename = os.path.basename(db_path)
           base_excludes.extend([
               basename,
               f"{basename}-*",
               f"{basename}-wal",
               f"{basename}-shm",
           ])
       
       return base_excludes
   
   def run_litestream(self):
       """Run Litestream in subprocess."""
       # Generate config from state
       config_path = "/tmp/litestream.yml"
       self._generate_litestream_config(config_path)
       
       self.litestream_proc = subprocess.Popen(
           ['litestream', 'replicate', '-config', config_path],
           stdout=subprocess.PIPE,
           stderr=subprocess.STDOUT
       )
       
       # Monitor output for metrics
       for line in self.litestream_proc.stdout:
           if not self.running:
               break
           # Parse Litestream log for metrics
           if b"replica" in line and b"position" in line:
               self.metrics['litestream_up'] = 1
   
   def _generate_litestream_config(self, path: str):
       """Generate Litestream config from state."""
       import yaml as pyyaml
       
       sqlite_paths = self.state.get('sqlite', {}).get('paths', [])
       
       config = {
           'dbs': []
       }
       
       for db_path in sqlite_paths:
           db_config = {
               'path': db_path,
               'replicas': [{
                   'url': f"s3://{self.state['bucket']}/litestream/{os.path.basename(db_path)}",
                   'endpoint': os.environ.get('S3_ENDPOINT', ''),
                   'access-key-id': os.environ['S3_ACCESS_KEY_ID'],
                   'secret-access-key': os.environ['S3_SECRET_ACCESS_KEY'],
               }]
           }
           config['dbs'].append(db_config)
       
       with open(path, 'w') as f:
           pyyaml.dump(config, f)
   
   def run_rclone_loop(self):
       """Run Rclone sync loop with validation."""
       fs_paths = self.state.get('filesystem', {}).get('paths', [])
       if not fs_paths:
           return
       
       # Generate excludes file
       excludes = self.generate_rclone_excludes()
       excludes_path = "/tmp/rclone-excludes.txt"
       with open(excludes_path, 'w') as f:
           for pattern in excludes:
               f.write(pattern + '\n')
       
       while self.running:
           # Validate YAML if needed
           if time.time() - self.last_full_validation > self.validation_interval:
               if not self.validate_yaml_cached(fs_paths):
                   self.metrics['rclone_errors'] += 1
                   time.sleep(60)
                   continue
               self.last_full_validation = time.time()
           
           # Run rclone
           try:
               self.rclone_proc = subprocess.run(
                   [
                       'rclone', 'sync',
                       *fs_paths,
                       f"s3:{self.state['bucket']}/filesystem",
                       '--exclude-from', excludes_path,
                       '--checksum',
                       '--min-age', '30s',
                       '--transfers', '4',
                       '--checkers', '8',
                       '--stats', '0',
                   ],
                   capture_output=True,
                   timeout=300  # 5 min max per sync
               )
               
               if self.rclone_proc.returncode == 0:
                   self.metrics['rclone_last_sync'] = int(time.time())
               else:
                   self.metrics['rclone_errors'] += 1
                   logger.error(f"Rclone failed: {self.rclone_proc.stderr}")
               
           except subprocess.TimeoutExpired:
               self.metrics['rclone_errors'] += 1
               logger.error("Rclone timeout")
           except Exception as e:
               self.metrics['rclone_errors'] += 1
               logger.error(f"Rclone exception: {e}")
           
           # Sleep before next sync
           for _ in range(60):
               if not self.running:
                   break
               time.sleep(1)
   
   def get_metrics(self) -> str:
       """Generate Prometheus metrics."""
       lines = [
           "# HELP data_guard_litestream_up Litestream replication status",
           "# TYPE data_guard_litestream_up gauge",
           f"data_guard_litestream_up {self.metrics['litestream_up']}",
           "",
           "# HELP data_guard_rclone_last_sync_timestamp Last successful Rclone sync",
           "# TYPE data_guard_rclone_last_sync_timestamp gauge",
           f"data_guard_rclone_last_sync_timestamp {self.metrics['rclone_last_sync']}",
           "",
           "# HELP data_guard_rclone_errors_total Rclone error count",
           "# TYPE data_guard_rclone_errors_total counter",
           f"data_guard_rclone_errors_total {self.metrics['rclone_errors']}",
           "",
           "# HELP data_guard_yaml_validation_duration_seconds YAML validation time",
           "# TYPE data_guard_yaml_validation_duration_seconds gauge",
           f"data_guard_yaml_validation_duration_seconds {self.metrics['yaml_validation_time']}",
       ]
       return '\n'.join(lines)
   
   def run(self):
       """Main entry point."""
       # Start metrics server
       server = HTTPServer(('0.0.0.0', 9090), MetricsHandler)
       server.orchestrator = self
       threading.Thread(target=server.serve_forever, daemon=True).start()
       
       # Start Litestream in thread
       litestream_thread = threading.Thread(target=self.run_litestream, daemon=True)
       litestream_thread.start()
       
       # Run Rclone in main thread (blocking)
       self.run_rclone_loop()

if __name__ == '__main__':
   orchestrator = SyncOrchestrator()
   orchestrator.run()

5.5 CLI Tools

#!/usr/bin/env python3
"""
data_guard/cli.py - Command-line tools for troubleshooting
"""

import argparse
import json
import sys
import boto3
from botocore.config import Config

def cmd_restore(args):
   """Force restore from specific generation."""
   print(f"Restoring {args.db_path} from generation {args.generation}")
   # Implementation: litestream restore with specific generation
   import subprocess
   result = subprocess.run([
       'litestream', 'restore',
       '-generation', str(args.generation),
       '-config', '/tmp/litestream.yml',
       args.db_path
   ])
   sys.exit(result.returncode)

def cmd_verify(args):
   """Verify backup integrity without starting app."""
   # Download and verify DB from S3
   print(f"Verifying backup in bucket {args.bucket}")
   # Implementation: download to temp, integrity_check, cleanup
   pass

def cmd_list_generations(args):
   """List available Litestream generations."""
   print(f"Generations for {args.bucket}:")
   # Implementation: litestream generations
   pass

def cmd_check_lock(args):
   """Check S3 lock status."""
   s3 = boto3.client('s3', 
       endpoint_url=os.environ.get('S3_ENDPOINT'),
       config=Config(connect_timeout=5, read_timeout=10)
   )
   
   try:
       obj = s3.head_object(Bucket=args.bucket, Key='.data-guard/lock')
       print(f"Lock held by: {obj['Metadata'].get('instance', 'unknown')}")
       print(f"Since: {obj['Metadata'].get('timestamp', 'unknown')}")
   except s3.exceptions.ClientError as e:
       if e.response['Error']['Code'] == '404':
           print("No lock active")
       else:
           raise

def cmd_force_release_lock(args):
   """Force release orphaned lock (DANGER)."""
   if not args.confirm:
       print("WARNING: This may cause split-brain. Use --confirm")
       sys.exit(1)
   
   s3 = boto3.client('s3',
       endpoint_url=os.environ.get('S3_ENDPOINT'),
       config=Config(connect_timeout=5, read_timeout=10)
   )
   
   s3.delete_object(Bucket=args.bucket, Key='.data-guard/lock')
   print("Lock released")

def main():
   parser = argparse.ArgumentParser(prog='dataangel-cli')
   subparsers = parser.add_subparsers(dest='command')
   
   # restore
   p_restore = subparsers.add_parser('restore')
   p_restore.add_argument('--bucket', required=True)
   p_restore.add_argument('--db-path', required=True)
   p_restore.add_argument('--generation', type=int, required=True)
   p_restore.set_defaults(func=cmd_restore)
   
   # verify
   p_verify = subparsers.add_parser('verify')
   p_verify.add_argument('--bucket', required=True)
   p_verify.set_defaults(func=cmd_verify)
   
   # list-generations
   p_list = subparsers.add_parser('list-generations')
   p_list.add_argument('--bucket', required=True)
   p_list.set_defaults(func=cmd_list_generations)
   
   # check-lock
   p_check = subparsers.add_parser('check-lock')
   p_check.add_argument('--bucket', required=True)
   p_check.set_defaults(func=cmd_check_lock)
   
   # force-release-lock
   p_release = subparsers.add_parser('force-release-lock')
   p_release.add_argument('--bucket', required=True)
   p_release.add_argument('--confirm', action='store_true')
   p_release.set_defaults(func=cmd_force_release_lock)
   
   args = parser.parse_args()
   if not args.command:
       parser.print_help()
       sys.exit(1)
   
   args.func(args)

if __name__ == '__main__':
   main()

5.6 Entrypoint Script

#!/bin/sh
# entrypoint.sh - Main entrypoint for data-guard image

set -e

COMMAND="${1:-init}"

case "$COMMAND" in
   init)
       exec python3 -m data_guard.init
       ;;
   sync)
       exec python3 -m data_guard.orchestrator
       ;;
   cli)
       shift
       exec python3 -m data_guard.cli "$@"
       ;;
   *)
       echo "Usage: $0 {init|sync|cli ...}"
       exit 1
       ;;
esac

5.7 Dockerfile

# syntax=docker/dockerfile:1

# Stage 1: Litestream binary
FROM litestream/litestream:0.5.9 AS litestream

# Stage 2: Rclone binary  
FROM rclone/rclone:1.73 AS rclone

# Stage 3: Python dependencies
FROM python:3.11-alpine3.19 AS python-deps

WORKDIR /app
RUN pip install --user --no-cache-dir \
   boto3 \
   botocore \
   pyyaml \
   prometheus-client

# Stage 4: Final image
FROM alpine:3.19 AS final

RUN apk add --no-cache \
   ca-certificates \
   sqlite \
   libstdc++ \
   python3 \
   py3-pip \
   && rm -rf /var/cache/apk/*

# Copy binaries
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/
COPY --from=rclone /usr/local/bin/rclone /usr/local/bin/

# Copy Python environment
COPY --from=python-deps /root/.local /home/data-guard/.local

# Copy application code
COPY src/ /app/
COPY entrypoint.sh /app/

# Setup non-root user
RUN adduser -D -u 1000 -h /home/data-guard data-guard && \
   chown -R data-guard:data-guard /home/data-guard /app

ENV PATH=/home/data-guard/.local/bin:/usr/local/bin:$PATH \
   HOME=/home/data-guard \
   PYTHONPATH=/app

USER data-guard
WORKDIR /app

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
   CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:9090/metrics')" || exit 1

ENTRYPOINT ["./entrypoint.sh"]
CMD ["init"]



6. Operational Procedures

6.1 Normal Operations

Operation

Command

Check init logs

kubectl logs deployment/app -c data-guard-init

Check sidecar logs

kubectl logs deployment/app -c data-guard

View metrics

kubectl port-forward pod/app-xxx 9090:9090 → curl localhost:9090/metrics

Force restore

kubectl exec -it pod/app-xxx -c data-guard -- dataangel-cli restore --bucket X --generation N --db-path /config/db

Check lock status

kubectl exec -it pod/app-xxx -c data-guard -- dataangel-cli check-lock --bucket X

Release orphaned lock

kubectl exec -it pod/app-xxx -c data-guard -- dataangel-cli force-release-lock --bucket X --confirm

6.2 Troubleshooting Scenarios

Pod stuck in Init:Error

# 1. Check init logs for specific error
kubectl logs pod/app-xxx -c data-guard-init

# 2. Check if S3 is reachable
kubectl exec -it pod/app-xxx -c data-guard-init -- \
   python3 -c "import boto3; s3 = boto3.client('s3'); s3.list_buckets()"

# 3. Check lock status
kubectl exec -it pod/app-xxx -c data-guard-init -- \
   dataangel-cli check-lock --bucket myapp

# 4. Force release if orphaned (after verifying no other pod is running)
kubectl exec -it pod/app-xxx -c data-guard-init -- \
   dataangel-cli force-release-lock --bucket myapp --confirm

Corruption detected (metrics alert)

# 1. Identify affected pod
kubectl get pods -l app=homeassistant

# 2. Check init logs for restore details
kubectl logs pod/app-xxx -c data-guard-init | grep -i "corrupt\|restore"

# 3. If in crash loop (>3 restores), scale to 0 and investigate
kubectl scale deployment/homeassistant --replicas=0

# 4. Manual restore from specific generation if needed
kubectl run debug --rm -it --image=data-guard -- \
   dataangel-cli restore --bucket homeassistant --generation 5 \
   --db-path /tmp/restore.db

# 5. Copy restored DB to PVC and restart
kubectl cp /tmp/restore.db homeassistant-0:/config/home-assistant_v2.db
kubectl scale deployment/homeassistant --replicas=1

6.3 Disaster Recovery

Scenario

Recovery Procedure

Complete cluster loss

Restore PVCs from Velero, data-guard reconciles or restores from S3

S3 data corruption

Manual PITR: identify last good generation via list-generations, restore specific generation

Application bug (corruption loop)

Scale to 0, restore from generation before bug, fix app, scale up

Lock orphaned after node crash

force-release-lock with confirmation, or wait TTL (5min)

Secret rotation failure

Reloader triggers restart, data-guard reconnects automatically



7. Risk Analysis

7.1 Accepted Risks

Risk

Mitigation

Acceptance Criteria

S3 unavailability blocks startup

By design - prefer downtime over data fork

Documented in runbook, monitoring alerts

Crash loop on recurring corruption

Metrics alert after 3 restores, manual intervention

Human in the loop for investigation

Torn writes on large files

--min-age 30s + checksum verification

Risk documented, next sync repairs

YAML validation on first pass

Full validation only on changed files

< 5s for 100 files, then < 50ms

Sidecar compromise

Security context hardening, non-root user

Risk acknowledged, network policies

7.2 Mitigated Risks

Risk

Mitigation

Restore of corrupted S3 data

Validation before backup prevents propagation

Multi-replica corruption

S3 lock with TTL and steal prevents split-brain

Secret rotation outage

Reloader handles pod restart

Image regression

Digest pinning, per-app version testing

State file corruption

Atomic write (temp + rename)

Graceful shutdown failure

SIGTERM handler with 15s Litestream wait

Unmanaged DB file loss

Dynamic Rclone excludes (not wildcard)

S3 blackhole blocking

Aggressive timeouts (5s connect, 10s read)

7.3 Future Risks (v2+)

Risk

Planned Mitigation

DB > 100GB integrity check time

Incremental checksum or online backup

Cross-region DR

Bucket replication, multi-endpoint support

Automated PITR

CLI automation, annotation-based trigger

Non-SQLite databases

Separate component for PostgreSQL native backup



8. Testing Strategy

8.1 Unit Tests

Component

Test Cases

Generation comparison

Local > S3, Local < S3, Local = S3, S3 missing

WAL salt extraction

Valid header, truncated header, no WAL file

Integrity check

Valid DB, corrupted DB, missing DB, locked DB

Lock mechanism

Acquire, release, timeout, steal expired

YAML validation cache

Unchanged file, modified file, new file, deleted file

Atomic file write

Success, failure mid-write, concurrent access

8.2 Integration Tests

Scenario

Setup

Expected Result

Healthy startup

PVC with valid data, S3 older

Skip restore, app starts in <5s

Corruption recovery

PVC with corrupted DB (injected)

Restore from S3, app starts, metrics logged

PVC deletion

Delete PVC, pod rescheduled

Full restore, app starts, data intact

S3 unavailability

Block S3 with network policy

Init blocks, CrashLoopBackOff, alert fires

Multi-replica protection

Scale to 2 replicas

Second pod fails with lock error, no split-brain

Graceful shutdown

Rolling update with active writes

WAL flushed, no data loss, new pod skips restore

Secret rotation

Update secret in Infisical

Reloader restarts pod, seamless reconnect

8.3 End-to-End Tests

Scenario

Validation

Complete data lifecycle

Write → Backup → Corrupt → Restore → Verify integrity

Rolling update resilience

10 successive rollouts, 0 data loss, all skip restore

Node drain survival

Cordon + drain → Pod moves → Data follows → App resumes

Backup corruption isolation

Corrupt local file → Validation fails → S3 backup unchanged



9. CI/CD Specification

9.1 Pipeline Overview (Mermaid)

flowchart LR
   subgraph "Development"
       D[Developer<br/>Local test]
       PR[Pull Request]
   end
   
   subgraph "CI Pipeline"
       T[Unit Tests]
       I[Integration Tests]
       B[Build Image]
       S[Security Scan]
       P[Push to Registry]
   end
   
   subgraph "Staging"
       CAN[Canary Deploy<br/>homeassistant-dev]
       E2E[E2E Tests]
   end
   
   subgraph "Production"
       TAG[Tag Release]
       ROLL[Rolling Update<br/>Per-app digest pin]
   end
   
   D --> PR
   PR --> T
   T --> I
   I --> B
   B --> S
   S --> P
   P --> CAN
   CAN --> E2E
   E2E --> TAG
   TAG --> ROLL
   
   style T fill:#e1f5fe
   style I fill:#e1f5fe
   style S fill:#fff3e0
   style CAN fill:#e8f5e9


9.2 GitHub Actions Workflow

# .github/workflows/data-guard.yml
name: Data-Guard CI/CD

on:
 push:
   branches: [main]
   paths: ['data-guard/**', '.github/workflows/data-guard.yml']
 pull_request:
   paths: ['data-guard/**']
 release:
   types: [published]

env:
 REGISTRY: ghcr.io
 IMAGE_NAME: truxonline/data-guard

jobs:
 test:
   runs-on: ubuntu-latest
   steps:
     - uses: actions/checkout@v4
     
     - name: Set up Python
       uses: actions/setup-python@v5
       with:
         python-version: '3.11'
         
     - name: Install dependencies
       working-directory: ./data-guard
       run: |
         pip install -r requirements.txt
         pip install -r requirements-test.txt
         
     - name: Run unit tests
       working-directory: ./data-guard
       run: pytest tests/unit --cov=src --cov-report=xml
       
     - name: Upload coverage
       uses: codecov/codecov-action@v3

 integration-test:
   runs-on: ubuntu-latest
   needs: test
   services:
     minio:
       image: minio/minio:latest
       env:
         MINIO_ROOT_USER: testuser
         MINIO_ROOT_PASSWORD: testpass
       ports:
         - 9000:9000
       options: >-
         --health-cmd "curl http://localhost:9000/minio/health/live"
         --health-interval 10s
         
   steps:
     - uses: actions/checkout@v4
     
     - name: Setup test environment
       run: |
         docker network create test-net
         
     - name: Run integration tests
       working-directory: ./data-guard
       run: pytest tests/integration -v
       env:
         S3_ENDPOINT: http://localhost:9000
         S3_ACCESS_KEY: testuser
         S3_SECRET_KEY: testpass

 build:
   runs-on: ubuntu-latest
   needs: [test, integration-test]
   if: github.event_name == 'push' || github.event_name == 'release'
   outputs:
     image_digest: ${{ steps.build.outputs.digest }}
   steps:
     - uses: actions/checkout@v4
     
     - name: Set up Docker Buildx
       uses: docker/setup-buildx-action@v3
       
     - name: Login to Registry
       uses: docker/login-action@v3
       with:
         registry: ${{ env.REGISTRY }}
         username: ${{ github.actor }}
         password: ${{ secrets.GITHUB_TOKEN }}
         
     - name: Extract metadata
       id: meta
       uses: docker/metadata-action@v5
       with:
         images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
         tags: |
           type=ref,event=branch
           type=sha,prefix={{branch}}-
           type=semver,pattern={{version}}
           type=raw,value=latest,enable={{is_default_branch}}
           
     - name: Build and push
       id: build
       uses: docker/build-push-action@v5
       with:
         context: ./data-guard
         push: true
         tags: ${{ steps.meta.outputs.tags }}
         labels: ${{ steps.meta.outputs.labels }}
         cache-from: type=gha
         cache-to: type=gha,mode=max
         platforms: linux/amd64,linux/arm64

 security-scan:
   runs-on: ubuntu-latest
   needs: build
   steps:
     - name: Run Trivy vulnerability scanner
       uses: aquasecurity/trivy-action@master
       with:
         image-ref: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ needs.build.outputs.image_digest }}
         format: 'sarif'
         output: 'trivy-results.sarif'
         
     - name: Upload to GitHub Security tab
       uses: github/codeql-action/upload-sarif@v2

 canary-deploy:
   runs-on: ubuntu-latest
   needs: [build, security-scan]
   if: github.ref == 'refs/heads/main'
   environment: staging
   steps:
     - uses: actions/checkout@v4
     
     - name: Update canary deployment
       run: |
         cd apps/homeassistant/overlays/dev
         kustomize edit set image data-guard=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ needs.build.outputs.image_digest }}
         
     - name: Commit and push
       run: |
         git config --local user.email "ci@truxonline.com"
         git config --local user.name "CI Bot"
         git add .
         git commit -m "chore: update data-guard canary to ${{ needs.build.outputs.image_digest }}"
         git push

 e2e-test:
   runs-on: ubuntu-latest
   needs: canary-deploy
   steps:
     - name: Wait for ArgoCD sync
       run: |
         kubectl rollout status deployment/homeassistant -n homeassistant-dev --timeout=300s
         
     - name: Run E2E tests
       run: |
         pytest tests/e2e --target homeassistant-dev

 promote-to-prod:
   runs-on: ubuntu-latest
   needs: e2e-test
   if: github.ref == 'refs/heads/main'
   environment: production
   steps:
     - uses: actions/checkout@v4
     
     - name: Create release tag
       run: |
         VERSION=$(date +%Y%m%d)-$(git rev-parse --short HEAD)
         git tag -a "data-guard-v${VERSION}" -m "Release data-guard v${VERSION}"
         git push origin "data-guard-v${VERSION}"
         
     - name: Update production overlays
       run: |
         for app in homeassistant paperless; do
           cd apps/$app/overlays/prod
           kustomize edit set image data-guard=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ needs.build.outputs.image_digest }}
         done
         
     - name: Create promotion PR
       uses: peter-evans/create-pull-request@v5
       with:
         title: "chore: promote data-guard to production"
         body: |
           Digest: ${{ needs.build.outputs.image_digest }}
           Tests: ${{ needs.e2e-test.result }}
           
           Manual approval required before merge.

9.3 Kustomize Component

# components/data-guard/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

images:
 - name: data-guard
   newName: ghcr.io/truxonline/data-guard
   newTag: v1.0.0  # Pin by digest in production overlays

patches:
 - target:
     kind: Deployment
   patch: |-
     apiVersion: apps/v1
     kind: Deployment
     metadata:
       name: ignored
     spec:
       template:
         spec:
           initContainers:
             - name: data-guard-init
               image: data-guard
               args: ["init"]
               envFrom:
                 - secretRef:
                     name: s3-credentials
               volumeMounts:
                 - name: data
                   mountPath: /data
                 - name: data-guard-state
                   mountPath: /tmp/data-guard
           containers:
             - name: data-guard
               image: data-guard
               args: ["sync"]
               ports:
                 - containerPort: 9090
                   name: metrics
               envFrom:
                 - secretRef:
                     name: s3-credentials
               volumeMounts:
                 - name: data
                   mountPath: /data
                 - name: data-guard-state
                   mountPath: /tmp/data-guard
           volumes:
             - name: data-guard-state
               emptyDir: {}



10. Appendices

Appendix A: Decision Log

Date

Decision

Rationale

2025-03-16

Mono-image vs multi-image

Simpler maintenance, single version to track

2025-03-16

Blocking on S3 failure

Prevent data fork, prefer explicit downtime

2025-03-16

No automated PITR

Complexity vs usage frequency, manual CLI sufficient

2025-03-16

EmptyDir for state

Volatile by design, fresh start on node change

2025-03-16

Circuit breaker at 3 restores

Balance between auto-recovery and alert fatigue

2025-03-16

Unified SyncOrchestrator

Memory optimization, single metrics endpoint

2025-03-16

YAML cache mtime+size

KISS principle, sufficient performance

2025-03-16

Dynamic Rclone excludes

Prevent silent data loss from unmanaged DBs

2025-03-16

Aggressive S3 timeouts

Fail-fast for availability, 5s/10s/2 retries

2025-03-16

SIGTERM with 15s Litestream wait

Ensure WAL flush, prevent data loss

Appendix B: Metrics Reference

Metric

Type

Description

data_guard_litestream_up

Gauge

Litestream replication status (1=up)

data_guard_rclone_last_sync_timestamp

Gauge

Unix timestamp of last successful sync

data_guard_rclone_errors_total

Counter

Cumulative Rclone error count

data_guard_yaml_validation_duration_seconds

Gauge

Time spent in YAML validation

data_guard_init_restore_total

Counter

Restores performed by init container

data_guard_init_duration_seconds

Histogram

Init container execution time

data_guard_s3_lock_held

Gauge

S3 lock ownership (1=held)

Appendix C: File Structure

data-guard/
├── Dockerfile
├── entrypoint.sh
├── requirements.txt
├── requirements-test.txt
├── src/
│   └── data_guard/
│       ├── __init__.py
│       ├── init.py
│       ├── orchestrator.py
│       ├── cli.py
│       └── utils.py
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── README.md

Appendix D: Example Application Overlay

# apps/homeassistant/overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
 - ../../base

components:
 - ../../../../components/data-guard

images:
 - name: data-guard
   digest: sha256:abc123...  # Pinned after canary validation

patches:
 - target:
     kind: Deployment
     name: homeassistant
   patch: |-
     - op: add
       path: /spec/template/metadata/annotations
       value:
         dataangel.io/enabled: "true"
         dataangel.io/bucket: "homeassistant-prod"
         dataangel.io/sqlite-paths: "/config/home-assistant_v2.db"
         dataangel.io/fs-paths: "/config/configuration.yaml,/config/automations.yaml,/config/scripts.yaml"
         dataangel.io/full-logs: "false"



