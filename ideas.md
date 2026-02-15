# Ideas

## High Availability — Persistent Volume Strategy

Signal-sideband uses a Nomad host volume (`signal-sideband-media` → `/data/media`) for media storage. Host volumes are node-local, so the job can only run on nodes where that volume is pre-configured. This blocks HA across multiple nodes (e.g., local Mac + DO worker).

### Option 1: Object Storage (DO Spaces / S3) — Recommended

Move media storage from local filesystem to S3-compatible object storage.

- DO Spaces is S3-compatible, ~$5/mo for 250GB
- Replace `os.WriteFile`/`os.ReadFile` with `s3.PutObject`/`s3.GetObject`
- Replace `MEDIA_PATH` env var with `S3_BUCKET` / `S3_ENDPOINT` / credentials
- Remove `volumes:` block from `infraspec.yaml`
- App becomes fully stateless and portable across nodes
- Media files are append-mostly blobs — perfect fit for object storage
- DB (PostgreSQL) is the only remaining stateful dependency, with its own replication story

### Option 2: NFS Shared Mount

Mount a shared NFS volume on every Nomad client node. Each node's `host_volume` points at the NFS mount path.

- No code changes to signal-sideband or Norn
- Adds NFS server as infrastructure dependency and SPOF
- Simple but operationally fragile

### Option 3: Nomad CSI with DO Block Storage

Use DigitalOcean's CSI driver to dynamically attach block storage volumes to whichever node the job lands on.

- Volume type changes from `host` to `csi` in Norn's translator
- Requires registering DO CSI plugin as a Nomad job
- **Single-attach only** — one node at a time, so failover but not true HA
