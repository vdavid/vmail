
### REST API

**Base path:** `/api/v1`

**Thread ID:** The `thread_id` we use in the API (e.g., `/api/v1/thread/{thread_id}`) is a stable,
unique identifier, such as the `Message-ID` header of the root/first message in the thread.

(The checked items are implemented)

* [x] `GET /auth/status`: Checks the Authelia token and tells the front end if the user has
  completed the setup/onboarding.
    * Response: `{"isSetupComplete": false}`.
    * `isSetupComplete: false` tells the React app to redirect to the `/settings` page for onboarding.
* [x] `GET /folders`: List all IMAP folders (Inbox, Sent, etc.).
    * Response: Array of folder objects with `name` and `role` fields.
    * Folders are sorted by role priority (inbox, sent, drafts, spam, trash, archive, other), then alphabetically within the same role.
* [x] `GET /threads?folder=Inbox&page=1&limit=100`: Get paginated threads for a folder.
    * Response: `{"threads": [...], "pagination": {"total_count": 100, "page": 1, "per_page": 100}}`.
    * Automatically syncs the folder from IMAP if the cache is stale.
    * Uses user's pagination setting from settings if no limit is provided.
* [x] `GET /search?q=from:george&page=1&limit=100`: Get paginated search results.
    * Response: `{"threads": [...], "pagination": {"total_count": 100, "page": 1, "per_page": 100}}`.
    * Supports Gmail-like search syntax (from:, to:, subject:, after:, before:, folder:, label:).
    * Empty query returns all emails in INBOX.
    * Uses user's pagination setting from settings if no limit is provided.
* [x] `GET /thread/{thread_id}`: Get all messages and content for one thread.
    * Response: Thread object with all messages, attachments, and bodies.
    * Automatically syncs missing message bodies from IMAP in batch.
    * Thread ID is URL-encoded Message-ID header.
* [ ] `GET /message/{message_id}/attachment/{attachment_id}`: Download an attachment.
* [x] `GET /settings`: Get user settings.
    * Response: `{"imap_server_hostname": "mail.example.com", "archive_folder_name": "Archive", ...}`
    * It should **not** return the encrypted passwords.
* [ ] `POST /send`: Send a new email (places in `action_queue` for "Undo Send").
* [ ] `POST /drafts`: Create or update a draft.
* [ ] `POST /actions`: Perform bulk actions.
    * Body: `{"action": "archive", "thread_ids": ["id1", "id2"]}`
    * Body: `{"action": "mark_read", "message_ids": ["id3"]}`
    * Body: `{"action": "star", "thread_ids": ["id1"]}`
* [ ] `POST /undo`: Undo the last `send` action.
* [x] `POST /settings`: Save settings.
    * Body:
      `{"imap_server_hostname": "imap.example.com", "imap_username": "user", "imap_password": "pass", "smtp_server_hostname": "smtp.example.com", "smtp_username": "user", "smtp_password": "pass", "undo_send_delay_seconds": 20, "pagination_threads_per_page": 100}`
    * Response: `200 OK`
* [ ] `DELETE /threads`: Move threads to trash.
    * Body: `{"thread_ids": ["id1", "id2"]}`

### Real-time API (WebSockets)

For real-time updates for new emails, the front end opens a WebSocket connection.

* [x] `GET /api/v1/ws`: Upgrades the HTTP connection to a WebSocket.
  The server uses this connection to push updates to the client.
    * The backend maintains a per-process **WebSocket Hub** that:
        * Tracks multiple connections per user (`userID -> set of connections`).
        * Limits the number of concurrent connections per user (default: 10).
        * Sends messages (like new-email notifications) to all active connections for a user.
    * When the first WebSocket connection for a user is established, the backend starts an **IMAP IDLE listener**:
        * Uses a dedicated IMAP listener connection from the pool.
        * Runs `IDLE` on the `INBOX` folder.
        * On new-mail notifications, performs an **incremental sync** for `INBOX` immediately and then pushes an event to the WebSocket hub.
    * **Server-to-client message example:**
        ```json
        {"type": "new_email", "folder": "INBOX"}
        ```
    * The front end listens for `new_email` messages and calls `queryClient.invalidateQueries({ queryKey: ['threads', folder] })`
      so `GET /threads?folder=...` refetches and the new email appears.

**Cache TTL as fallback:**  
The 5‑minute cache TTL used by `GET /threads` is a **backup mechanism**:

* Real-time updates (IDLE + WebSockets) cause immediate incremental syncs for `INBOX`.
* TTL-based sync still runs when:
    * WebSockets are not connected or temporarily unavailable.
    * The IDLE listener fails or is not yet started.
    * A user navigates to a folder without real-time support.
