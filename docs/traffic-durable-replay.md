# Durable pending traffic replay (Phase 2)

TX-Node persists exactly one unacknowledged traffic batch before HTTP delivery.
The batch includes a stable random `traffic_batch_id` and the byte counters.
The on-disk file lives beneath the configured `kernel.config_dir`; this path
**must be mounted on a durable writable volume** (not an ephemeral container
overlay) for replay across container replacement.

On a response timeout, TX-Node retains the same batch for retry, without
flushing or merging new tracker bytes. On process restart, it loads the
pending batch and submits the same ID and payload before any fresh traffic.
A successful HTTP acknowledgement deletes the durable spool file.

Startup fails closed if the spool is corrupt, unreadable or its directory is
not writable. Permissions are owner-only (0600). Avoid running two active
TX-Node processes with the same panel/node identity and state directory.
The application does not yet guarantee traffic sampled but not flushed
before an abrupt crash; only flushed, persisted pending batches are protected.

The matching TXBoard backend requires the `v2_traffic_batch` migration
from TXBoard PR #78. Deploy TXBoard before enabling upgraded TX-Node.
