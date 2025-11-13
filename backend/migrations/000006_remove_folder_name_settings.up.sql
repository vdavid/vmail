-- Remove folder name columns from user_settings table.
-- These are no longer needed as we now use IMAP SPECIAL-USE extension (RFC 6154)
-- to automatically detect folder roles from the IMAP server.
ALTER TABLE "user_settings"
    DROP COLUMN IF EXISTS "archive_folder_name",
    DROP COLUMN IF EXISTS "sent_folder_name",
    DROP COLUMN IF EXISTS "drafts_folder_name",
    DROP COLUMN IF EXISTS "trash_folder_name",
    DROP COLUMN IF EXISTS "spam_folder_name";

