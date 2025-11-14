# Folders

The `folders` back end feature provides a way to list IMAP folders for the authenticated user.

The feature is intentionally not organized into a single package so that API-level functions can share helpers, etc.

## Components

* **`internal/api/folders_handler.go`**: HTTP handler for the `/api/v1/folders` endpoint.
    * `GetFolders`: Lists all IMAP folders for the current user, sorted by role priority.
    * `getUserSettingsAndPassword`: Retrieves user settings and decrypts the IMAP password.
    * `getIMAPClient`: Gets an IMAP client from the pool, with user-friendly error messages for timeouts.
    * `listFoldersWithRetry`: Lists folders with automatic retry on connection errors.
    * `retryListFolders`: Retries listing folders after removing a broken connection from the pool.
    * `writeFoldersResponse`: Writes the sorted folders as JSON.
    * `sortFoldersByRole`: Sorts folders by role priority (inbox, sent, drafts, spam, trash, archive, other), then alphabetically within the same role.

* **`internal/imap/folder.go`**: IMAP folder listing implementation.
    * `ListFolders`: Lists all folders on the IMAP server using SPECIAL-USE attributes (RFC 6154) to determine roles.
    * `determineFolderRole`: Maps folder names and SPECIAL-USE attributes to role strings.

## Flow

1. Handler extracts user ID from request context.
2. Retrieves and decrypts user settings (IMAP credentials).
3. Gets an IMAP client from the connection pool.
4. Lists folders from the IMAP server.
5. If a connection error occurs (broken pipe, connection reset, EOF), removes the broken client from the pool and retries with a fresh connection.
6. Sorts folders by role priority and alphabetically.
7. Returns folders as JSON.

## Error handling

* Returns 404 if user settings are not found.
* Returns 400 if the IMAP server doesn't support SPECIAL-USE extension (required for V-Mail).
* Returns 503 (Service Unavailable) for connection timeout errors with a user-friendly message.
* Returns 500 for other connection or internal errors.
* Automatically retries on transient connection errors (broken pipe, connection reset, EOF).

## Dependencies

* Requires IMAP server support for SPECIAL-USE extension (RFC 6154) to identify folder roles.
* Uses the IMAP connection pool to manage client connections efficiently.
