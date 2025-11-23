# Database

The DB's role is **not** to be a copy of the mailbox.

## Primary roles

* **Caching**: Thread/message metadata for a fast UI.
* **Settings**: User settings and **encrypted** IMAP/SMTP credentials.
* **Drafts**: Temporary storage for auto-save.
* **Queue**: Actions (like "Undo Send" or offline operations) to be processed asynchronously.

## Schema summary

### Core identity & settings

- **`users`**: Minimal identity.
    - `email`: From Authelia.
- **`user_settings`**: App-specific configuration (1:1 with users).
    - `encrypted_{imap,smtp}_password`: AES-GCM encrypted credentials.
    - Settings: `undo_send_delay_seconds`, `pagination_threads_per_page`.

### Email cache

- **`threads`**: Folder-agnostic conversation anchor.
    - `stable_thread_id`: The `Message-ID` of the root message. Unique constraint ensures we group replies correctly.
- **`messages`**: The main content cache.
    - `thread_id`: Link to parent thread.
    - `imap_folder_name` & `imap_uid`: Location on the IMAP server.
    - `unsafe_body_html`: Raw HTML (must be sanitized by frontend).
    - `is_read`, `is_starred`: Synced flags.
- **`attachments`**: Metadata for files.
    - `content_id`: For inline images.
- **`folder_sync_timestamps`**: Sync state tracking.
    - `last_synced_uid`: For incremental sync.
    - `thread_count`: Materialized count for sidebar badges.

### Action & state

- **`drafts`**: Fast auto-save storage.
    - Synced to IMAP in the background.
- **`action_queue`**: Deferred operations.
    - `action_type`: e.g., `send_email`, `mark_read`.
    - `process_at`: Timestamps for delayed execution (Undo Send).

## Usage patterns

- **Cache TTL**: We don't keep messages forever if the user has a huge mailbox. The sync logic handles eviction/updates.
- **Encryption**: Credentials in `user_settings` are encrypted at rest using a key from environment variables.
- **Sanitization**: The DB stores *unsafe* HTML. The API delivers it as-is. The Frontend *must* sanitize.
