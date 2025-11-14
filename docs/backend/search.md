# Search

The `search` feature provides search for email threads using a Gmail-like query syntax.

The feature is intentionally not organized into a single package so that API-level functions can share helpers, etc.

## Components

* **`internal/api/search_handler.go`**: HTTP handler for the `/api/v1/search` endpoint.
    * `Search`: Handles search requests with query parameter parsing and pagination.
    * `getPaginationLimit`: Gets pagination limit from user settings or defaults.

* **`internal/imap/search.go`**: IMAP search implementation and query parsing.
    * `ParseSearchQuery`: Parses Gmail-like search queries into IMAP SearchCriteria.
    * `Search`: Performs IMAP search and returns paginated threads.
    * `buildThreadMapFromMessages`: Builds thread map from IMAP search results.
    * `sortAndPaginateThreads`: Sorts threads by latest sent_at and applies pagination.
    * `tokenizeQuery`: Tokenizes query string, respecting quoted strings.
    * `parseHeaderFilter`: Parses header filters (from:, to:, subject:).
    * `parseDateFilter`: Parses date filters (after:, before:).
    * `parseFolderFilter`: Parses folder/label filters (folder:, label:).

## Flow

1. Handler extracts user ID from request context.
2. Gets query from `q` query parameter (empty query means return all emails).
3. Parses pagination parameters (page, limit) from query string.
4. Gets pagination limit from user settings if not provided in query.
5. Calls IMAP service to search for matching threads.
6. IMAP service parses query using Gmail-like syntax.
7. IMAP service searches the specified folder (or INBOX if not specified).
8. IMAP service fetches message headers for matching UIDs.
9. IMAP service builds thread map from messages in the database.
10. IMAP service sorts threads by latest sent_at and applies pagination.
11. IMAP service enriches threads with first message's from_address.
12. Returns paginated response with threads and pagination info.

## Search syntax

* **Header filters:**
    * `from:george` - Search by sender
    * `to:alice` - Search by recipient
    * `subject:meeting` - Search by subject
    * Quoted values: `from:"John Doe"` - Search with quoted strings

* **Date filters:**
    * `after:2025-01-01` - Messages after date (YYYY-MM-DD format)
    * `before:2025-12-31` - Messages before date (end of day)

* **Folder filters:**
    * `folder:Inbox` - Search in specific folder
    * `label:Sent` - Alias for folder: (Gmail compatibility)

* **Plain text:**
    * `cabbage` - Full-text search across message content

* **Combinations:**
    * `from:george after:2025-01-01 cabbage` - Multiple filters and text search

## Pagination

* Default page: 1
* Default limit: User's setting from `PaginationThreadsPerPage`, or 100 if not set.
* Query parameters: `page` and `limit` can override defaults.
* Invalid values (non-positive numbers) fall back to defaults.

## Error handling

* Returns 400 for invalid query syntax (e.g., empty filter values, invalid date formats).
* Returns 500 for IMAP connection errors, search failures, or database errors.
* Returns 500 for JSON encoding errors.
* If thread enrichment fails, continues gracefully (threads work without from_address).

## Current limitations

* Search is limited to a single folder (defaults to INBOX if not specified).
* Full-text search uses IMAP's TEXT search criteria (server-dependent behavior).
* Threads are sorted by latest sent_at only (no other sort options).
