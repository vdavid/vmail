-- Add columns to folder_sync_timestamps for incremental sync and materialized count
ALTER TABLE "folder_sync_timestamps"
ADD COLUMN "last_synced_uid" BIGINT,
ADD COLUMN "thread_count" INT DEFAULT 0;

COMMENT ON COLUMN "folder_sync_timestamps"."last_synced_uid" IS 'The highest IMAP UID we have synced for this folder. Used for incremental sync.';
COMMENT ON COLUMN "folder_sync_timestamps"."thread_count" IS 'Materialized count of threads in this folder. Recalculated on each sync.';

