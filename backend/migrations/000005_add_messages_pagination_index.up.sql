-- Composite index to optimize pagination queries for threads
-- This index helps the GetThreadsForFolder query which:
-- - Filters by user_id and imap_folder_name (WHERE clause)
-- - Orders by sent_at DESC (ORDER BY clause)
-- - Uses MAX(m2.sent_at) aggregation
CREATE INDEX idx_messages_user_folder_sent_at 
ON "messages" ("user_id", "imap_folder_name", "sent_at" DESC NULLS LAST);

