# Threads

The `threads` feature provides a way to list email threads for a folder with pagination support.

It's intentionally not organized into a single package so that API-level functions can share helpers, etc.

## Components

* **`internal/api/threads_handler.go`**: HTTP handler for the `/api/v1/threads` endpoint.
    * `GetThreads`: Returns a paginated list of email threads for a folder.
    * `parsePaginationParams`: Parses page and limit query parameters with validation.
    * `getPaginationLimit`: Gets pagination limit from user settings or defaults.
    * `syncFolderIfNeeded`: Checks if folder needs syncing and syncs if necessary.
    * `buildPaginationResponse`: Builds the paginated response structure.

* **`internal/db/threads.go`**: Database operations for threads.
    * `GetThreadsForFolder`: Retrieves paginated threads for a folder.
    * `GetThreadCountForFolder`: Gets the total count of threads for pagination.
    * `SaveThread`: Saves or updates a thread in the database.

## Flow

1. Handler extracts user ID from request context.
2. Validates that the `folder` query parameter is provided.
3. Parses pagination parameters (page, limit) from query string.
4. Gets pagination limit from user settings if not provided in query.
5. Checks if folder needs syncing and syncs from IMAP if stale.
6. Retrieves threads from the database with pagination.
7. Gets total thread count for pagination metadata.
8. Returns paginated response with threads and pagination info.

## Pagination

* Default page: 1
* Default limit: User's setting from `PaginationThreadsPerPage`, or 100 if not set.
* Query parameters: `page` and `limit` can override defaults.
* Invalid values (non-positive numbers) fall back to defaults.

## Sync behavior

* Automatically checks if folder cache is stale before returning threads.
* If stale, syncs from IMAP server in the background.
* If sync fails, continues and returns cached data (graceful degradation).
* Sync errors are logged but don't fail the request.

## Thread Fields

The `GetThreadsForFolder` function returns threads with the following fields populated for list views:

* **`message_count`**: Number of messages in the thread. Always populated in list views to avoid needing to load the full messages array.
* **`last_sent_at`**: Date/time of the most recent message in the thread. Used for date display in the email list (shows time if today, otherwise shows day).
* **`preview_snippet`**: First 100 characters of the first message's body text, with whitespace normalized. Used for email preview in the list view.
* **`has_attachments`**: Boolean indicating if any messages in the thread have non-inline attachments. Used to display attachment indicator (📎) in the list view.
* **`first_message_from_address`**: Sender address of the first message in the thread. Used to display the sender name in the list view.

The `EnrichThreadsWithPreviewAndAttachments` function also populates these fields for search results and other cases where threads don't have them pre-populated.

## Error handling

* Returns 400 if folder parameter is missing.
* Returns 500 for database errors (getting threads or count).
* Returns 500 for JSON encoding errors.
