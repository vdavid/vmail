# IMAP

The `imap` package handles all communication with IMAP servers, including connection pooling, folder listing, message
syncing, and searching.

This is probably the trickiest part of the codebase.

## Components

* **`internal/imap/client.go`**: Connection pool implementation.
    * `Pool`: Manages IMAP connections per user (one connection per user, reused across requests).
    * `getClientConcrete`: Gets or creates an IMAP client, checking connection health.
    * `GetClient`: Public interface that returns an `IMAPClient` wrapper.
    * `RemoveClient`: Removes a broken connection from the pool.
    * `ConnectToIMAP`: Establishes connection with 5-second timeout.
    * `Login`: Authenticates with the IMAP server.

* **`internal/imap/pool_interface.go`**: Interfaces for testability.
    * `IMAPClient`: Interface for IMAP client operations (currently only `ListFolders`).
    * `IMAPPool`: Interface for connection pool operations.
    * `ClientWrapper`: Wraps go-imap client to implement `IMAPClient`.

* **`internal/imap/service.go`**: Main IMAP service implementation.
    * `Service`: Handles IMAP operations and caching.
    * `SyncThreadsForFolder`: Syncs threads from IMAP (incremental or full sync).
    * `SyncFullMessage`: Syncs a single message body.
    * `SyncFullMessages`: Batch syncs multiple message bodies.
    * `Search`: Searches for threads matching a query.
    * `ShouldSyncFolder`: Checks if folder cache is stale.

* **`internal/imap/fetch.go`**: Message fetching operations.
    * `FetchMessageHeaders`: Fetches headers for multiple messages.
    * `FetchFullMessage`: Fetches full message body.
    * `SearchUIDsSince`: Searches for UIDs >= minUID (for incremental sync).

* **`internal/imap/folder.go`**: Folder listing operations.
    * `ListFolders`: Lists folders with SPECIAL-USE attributes.
    * `determineFolderRole`: Maps folder names and attributes to roles.

* **`internal/imap/thread.go`**: Thread structure operations.
    * `RunThreadCommand`: Executes IMAP THREAD command.

* **`internal/imap/parser.go`**: Message parsing.
    * `ParseMessage`: Converts IMAP message to internal model.
    * `parseBody`: Parses email body using enmime library.

* **`internal/imap/search.go`**: Search query parsing and execution.
    * `ParseSearchQuery`: Parses Gmail-like search queries.
    * `Search`: Performs IMAP search and returns threads.

## Connection Pooling

The connection pool is a critical and complex part of the codebase. Key characteristics:

* **Worker connections**: Each user has a pool of 1–3 worker connections for API handlers (SEARCH, FETCH, STORE). These
  connections are reused across requests and managed by a semaphore to limit concurrent connections.
* **Listener connections**: Each user has one dedicated listener connection for the IDLE command (for real-time email
  notifications via WebSocket).
* **Thread safety**:
    * IMAP clients from `go-imap` are **NOT thread-safe**. Each connection is wrapped with a mutex (`clientWithMutex`)
      to ensure thread-safe access.
    * Multiple goroutines can use different connections concurrently, but access to the same connection is serialized by
      the mutex.
    * Folder selection is thread-safe because connections are locked during operations.
* **Connection lifecycle management**:
    * **Idle timeout**: Worker connections are closed after 10 minutes of inactivity. Listener connections have no idle
      timeout (IDLE keeps them alive).
    * **Health checks**: Before reusing a connection that's been idle > 1 minute, a NOOP command is sent to verify the
      connection is alive.
    * **Automatic cleanup**: A background goroutine runs every minute to remove idle connections.
* **Connection limits**: Maximum of 3 worker connections per user (enforced by semaphore). One listener connection per
  user.

## Thread safety guarantees

* **Per-connection mutexes**: Each connection has its own mutex, allowing concurrent access to different connections
  while serializing access to the same connection.
* **Double-check locking**: Used when creating new connections to prevent race conditions where multiple goroutines
  create connections simultaneously.
* **Semaphore-based limiting**: Worker connections are limited by a semaphore (max 3 per user), ensuring proper resource
  management.

## Sync behavior

* **Incremental sync**: If a folder has been synced before, only new messages (UIDs > last synced UID) are fetched.
* **Full sync**: If no sync info exists or incremental sync fails, all messages are fetched using THREAD command (or
  SEARCH as fallback).
* **Thread structure**: Full sync uses IMAP THREAD command to build thread relationships. If THREAD is not supported,
  falls back to processing messages without threading.
* **Lazy loading**: Message bodies are not always synced immediately. They are synced on-demand when a thread is viewed.

## Error handling

* Sync errors are logged but don't fail requests (graceful degradation).
* Broken connections are removed from the pool and recreated on next use.
* Folder selection errors are propagated to the caller.
* Network errors during fetch are propagated to the caller.
