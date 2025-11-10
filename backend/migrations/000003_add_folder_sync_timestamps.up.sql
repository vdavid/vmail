-- Table to track when we synced each folder for each user.
-- This enables efficient cache TTL checking without scanning all messages.
CREATE TABLE "folder_sync_timestamps"
(
    "user_id"     UUID        NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
    "folder_name" TEXT        NOT NULL,
    "synced_at"   TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY ("user_id", "folder_name")
);

CREATE INDEX idx_folder_sync_timestamps_user_id ON "folder_sync_timestamps" ("user_id");

COMMENT ON TABLE "folder_sync_timestamps" IS 'Table to track when we synced each folder for each user. This enables efficient cache TTL checking without scanning all messages.';
