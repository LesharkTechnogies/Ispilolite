# Database Disaster Recovery

## Backup Policy

- Run `scripts/backup-postgres.ps1` daily with a secret-managed `DATABASE_URL`.
- Retain daily backups for 30 days, weekly copies for 12 weeks, and monthly copies for 12 months.
- Store encrypted copies in a separate region and account with object-lock enabled.
- Enable PostgreSQL WAL archiving for point-in-time recovery between full backups.
- Alert when the last successful backup is older than 26 hours.

## Restore Procedure

1. Provision an isolated PostgreSQL instance at the same or newer compatible version.
2. Restore the latest full backup using `scripts/restore-postgres.ps1`.
3. Apply archived WAL to the selected recovery timestamp when point-in-time recovery is required.
4. Run schema verification and application smoke tests against the isolated instance.
5. Pause application writes, capture a final backup, and switch the database secret/DNS endpoint.
6. Monitor error rate, replication, locks, and queue consumers before resuming normal traffic.

## Targets

- Recovery point objective: 15 minutes with WAL archiving, 24 hours without it.
- Recovery time objective: 60 minutes for the primary database.
- Test restoration quarterly and record duration, backup checksum, row-count checks, and sign-off.

Never overwrite the failed primary during the first restore attempt. Preserve it for investigation and possible incremental recovery.
