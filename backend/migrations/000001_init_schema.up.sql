-- First, enable the pgcrypto extension to get gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Stores the V-Mail user.
-- Records in this table answer the question "Who is this user?"
-- We keep this table minimal, only storing the core identity.
CREATE TABLE "users"
(
    "id"         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),

    -- The user's login email, which we get from Authelia after a successful login.
    "email"      TEXT        NOT NULL UNIQUE,

    "created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Stores user-specific settings and credentials for the V-Mail app.
-- This is a 1:1 relationship with the "users" table.
-- We keep it separate to follow a clear "separation of concerns":
-- "users" handles *identity*, while "user_settings" handles *application data*.
CREATE TABLE "user_settings"
(
    "user_id"                     UUID PRIMARY KEY REFERENCES "users" ("id") ON DELETE CASCADE,

    "undo_send_delay_seconds"     INT         NOT NULL DEFAULT 20,
    "pagination_threads_per_page" INT         NOT NULL DEFAULT 100,

    -- We store IMAP and SMTP credentials so the Go backend can connect
    -- to the mail server on the user's behalf to send/receive email.
    "imap_server_hostname"        TEXT        NOT NULL,
    "imap_username"               TEXT        NOT NULL,

    -- This *must* be encrypted. We use AES-GCM, with the master encryption key
    -- provided to the backend as an environment variable.
    "encrypted_imap_password"     BYTEA       NOT NULL,

    "smtp_server_hostname"        TEXT        NOT NULL,
    "smtp_username"               TEXT        NOT NULL,

    -- This *must* also be encrypted, using the same method.
    "encrypted_smtp_password"     BYTEA       NOT NULL,

    -- These folder names map V-Mail's actions (like "Archive") to the
    -- user's actual IMAP folder names. IMAP servers can name these differently.
    -- On the first login, the backend should try to auto-detect these using
    -- the IMAP SPECIAL-USE extension (RFC 6154), but the user should
    -- be able to override them in the settings.
    "archive_folder_name"         TEXT        NOT NULL DEFAULT 'Archive',
    "sent_folder_name"            TEXT        NOT NULL DEFAULT 'Sent',
    "drafts_folder_name"          TEXT        NOT NULL DEFAULT 'Drafts',
    "trash_folder_name"           TEXT        NOT NULL DEFAULT 'Trash',
    "spam_folder_name"            TEXT        NOT NULL DEFAULT 'Spam',

    "created_at"                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    "updated_at"                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Caches the "anchor" for a thread.
-- A thread is a folder-agnostic container. This table just proves
-- a thread exists and links it to a subject line.
CREATE TABLE "threads"
(
    "id"               UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    "user_id"          UUID NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,

    -- This is the `Message-ID` header of the thread's root (first) message.
    -- It's our stable, unique key for the whole conversation.
    -- Using this ID allows us to group messages from different folders
    -- (e.g., 'INBOX' and 'Sent') into a single thread.
    "stable_thread_id" TEXT NOT NULL,

    -- The subject from the root message, used for display in the list.
    "subject"          TEXT,

    -- A user can only have one thread with a given stable ID.
    UNIQUE ("user_id", "stable_thread_id")
);

-- Caches individual messages.
-- This is our main workhorse table.
CREATE TABLE "messages"
(
    "id"                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    "thread_id"         UUID    NOT NULL REFERENCES "threads" ("id") ON DELETE CASCADE,

    -- This is a denormalized field (it's redundant).
    -- We include it for performance, so we can query messages
    -- by user without needing to JOIN against the "threads" table.
    "user_id"           UUID    NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,

    -- The IMAP Unique ID. This is a number that's only unique
    -- *within* a specific "imap_folder_name".
    "imap_uid"          BIGINT  NOT NULL,

    -- The IMAP folder (e.g., "INBOX", "Sent") where this specific message lives.
    -- Messages within the same thread will often be in different folders.
    "imap_folder_name"  TEXT    NOT NULL,

    -- The globally unique "<...@...>" header.
    -- This is what IMAP's THREAD command uses to group messages.
    "message_id_header" TEXT    NOT NULL,

    "from_address"      TEXT,
    "to_addresses"      TEXT[],
    "cc_addresses"      TEXT[],
    "sent_at"           TIMESTAMPTZ,
    "subject"           TEXT,

    -- The raw, unsanitized HTML from the email.
    -- The front end *must* sanitize this with DOMPurify before rendering it.
    "unsafe_body_html"  TEXT,
    "body_text"         TEXT,

    -- This boolean mirrors the IMAP `\Seen` flag for this message.
    "is_read"           BOOLEAN NOT NULL DEFAULT false,

    -- This boolean mirrors the IMAP `\Flagged` flag for this message.
    "is_starred"        BOOLEAN NOT NULL DEFAULT false,

    -- A message (identified by its UID) can only exist once per folder per user.
    UNIQUE ("user_id", "imap_folder_name", "imap_uid")
);

-- Stores metadata about attachments.
CREATE TABLE "attachments"
(
    "id"         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    "message_id" UUID    NOT NULL REFERENCES "messages" ("id") ON DELETE CASCADE,

    "filename"   TEXT    NOT NULL,
    "mime_type"  TEXT    NOT NULL,
    "size_bytes" BIGINT  NOT NULL,

    -- True if this attachment is meant to be shown inside the email body
    -- (e.g., a signature image). False if it's a downloadable file.
    "is_inline"  BOOLEAN NOT NULL DEFAULT false,

    -- The "<Content-ID>" header value.
    -- This is used to match an inline attachment to an <img src="cid:...">
    -- tag in the "unsafe_body_html" field of the message.
    "content_id" TEXT
);

-- For auto-saving drafts.
-- This table provides a fast, responsive auto-save experience.
-- The Go backend saves the draft here *first* (for speed),
-- then syncs the draft to the IMAP "drafts_folder_name" in the background.
CREATE TABLE "drafts"
(
    "id"                     UUID PRIMARY KEY     DEFAULT gen_random_uuid(),

    "user_id"                UUID        NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,

    -- The "message_id_header" of the email being replied to.
    -- This is *not* a foreign key to our "messages" table. It's the
    -- globally unique header string.
    "in_reply_to_message_id" TEXT,

    "to_addresses"           TEXT[],
    "cc_addresses"           TEXT[],
    "bcc_addresses"          TEXT[],
    "subject"                TEXT,
    "body_html"              TEXT,
    "last_saved_at"          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A queue for actions that need to be delayed or run reliably.
-- This enables "Undo Send" and a robust "offline mode" for simple actions.
CREATE TABLE "action_queue"
(
    "id"          UUID PRIMARY KEY     DEFAULT gen_random_uuid(),

    "user_id"     UUID        NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,

    -- The type of action to perform.
    -- Examples: 'send_email', 'star_thread', 'mark_read', 'move_thread'
    "action_type" TEXT        NOT NULL,

    -- A JSON blob with the data needed to perform the action.
    -- For 'send_email': {"to_addresses": [...], "subject": "..."}
    -- For 'star_thread': {"thread_stable_id": "...", "star_status": true}
    -- For 'mark_read': {"thread_stable_id": "...", "read_status": true}
    "payload"     JSONB       NOT NULL,

    "created_at"  TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- The time when the action should run.
    -- For 'send_email', this is `NOW() + undo_send_delay_seconds`.
    -- For other actions, it's just `NOW()`.
    -- A background worker in Go polls this table for actions
    -- where `process_at <= NOW()`.
    "process_at"  TIMESTAMPTZ NOT NULL
);
