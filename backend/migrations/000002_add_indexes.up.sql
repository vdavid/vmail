-- Index all foreign keys for fast JOINs and relation-based lookups
CREATE INDEX idx_threads_user_id ON "threads" ("user_id");
CREATE INDEX idx_messages_thread_id ON "messages" ("thread_id");
CREATE INDEX idx_attachments_message_id ON "attachments" ("message_id");
CREATE INDEX idx_drafts_user_id ON "drafts" ("user_id");
CREATE INDEX idx_action_queue_user_id ON "action_queue" ("user_id");

-- Index for the background worker to poll for new jobs
CREATE INDEX idx_action_queue_process_at ON "action_queue" ("process_at");
