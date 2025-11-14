# Thread

The `thread` feature provides a way to retrieve a single email thread with all its messages, attachments, and bodies.

It's intentionally not organized into a single package so that its API-level functions can share helpers, etc.

## Components

* **`internal/api/thread_handler.go`**: HTTP handler for the `/api/v1/thread/{thread_id}` endpoint.
    * `GetThread`: Returns a single thread with all messages, attachments, and bodies.
    * `getStableThreadIDFromPath`: Extracts and URL-decodes the thread ID from the request path.
    * `collectMessagesToSync`: Identifies messages that need body syncing (lazy loading).
    * `syncMissingBodies`: Syncs missing message bodies from IMAP in batch.
    * `assignAttachments`: Assigns batch-fetched attachments to messages.
    * `convertMessagesToThreadMessages`: Converts messages for response, ensuring attachments are never nil.

* **`internal/db/messages.go`**: Database operations for messages and attachments.
    * `GetMessagesForThread`: Retrieves all messages for a thread, ordered by sent_at.
    * `GetMessageByUID`: Retrieves a message by IMAP UID and folder.
    * `GetAttachmentsForMessages`: Batch-fetches attachments for multiple messages (avoids N+1 queries).

## Flow

1. Handler extracts user ID from request context.
2. Extracts and URL-decodes thread ID from the request path.
3. Retrieves thread from database by stable thread ID.
4. Retrieves all messages for the thread.
5. Batch-fetches all attachments for the messages (single query).
6. Identifies messages with missing bodies (lazy loading optimization).
7. Syncs missing bodies from IMAP in batch if needed.
8. Re-fetches synced messages to get updated bodies.
9. Assigns attachments to messages and converts for response.
10. Returns thread with all messages, attachments, and bodies.

## Lazy loading

* Message bodies are not always synced immediately when threads are synced.
* Bodies are synced on-demand when a thread is viewed.
* This optimization reduces initial sync time and storage requirements.
* Bodies are synced in batch for efficiency.

## Error handling

* Returns 400 if thread_id is missing or invalid.
* Returns 404 if thread is not found.
* Returns 500 for database errors.
* If attachment fetching fails, continues with empty attachments.
* If body sync fails, continues with messages without bodies (graceful degradation).
* Returns 500 for JSON encoding errors.

## Performance optimizations

* Batch-fetches attachments in a single query (avoids N+1 queries).
* Batch-syncs missing message bodies.
* Uses efficient UID-to-index mapping for updating synced messages.
