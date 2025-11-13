-- Restore folder name columns to user_settings table.
-- This migration reverses the removal of folder name columns.
ALTER TABLE "user_settings"
    ADD COLUMN "archive_folder_name" TEXT NOT NULL DEFAULT 'Archive',
    ADD COLUMN "sent_folder_name" TEXT NOT NULL DEFAULT 'Sent',
    ADD COLUMN "drafts_folder_name" TEXT NOT NULL DEFAULT 'Drafts',
    ADD COLUMN "trash_folder_name" TEXT NOT NULL DEFAULT 'Trash',
    ADD COLUMN "spam_folder_name" TEXT NOT NULL DEFAULT 'Spam';

