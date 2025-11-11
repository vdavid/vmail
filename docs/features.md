# Features

## Back end

* Serves the front end via `http.FileServer`
* Validates JWTs from Authelia
* Validates user credentials in the DB
* Pools IMAP connections
* Uses IMAP commands: `SELECT`, `FETCH`, `THREAD`, `SEARCH`, `STORE`, `APPEND`, `COPY`
* Provides a helper function for generating an encryption key for AES-GCM encryption.
* Uses IMAP's IDLE command as per [RFC 2177](https://datatracker.ietf.org/doc/html/rfc2177). Runs a goroutine
  for each active user to get notified as soon as an email arrives.
* Maintains a connection pool to the IMAP server, making sure connections exist at all times.
  We need two types of connections for each active user:
    * **The "Worker" Pool:** A pool of 1–3 "normal" connections used by the API handlers to run `SEARCH`, `FETCH`,
      `STORE` (star, archive), and so on. These are for short-lived commands.
    * **The "Listener" Connection:** A single, dedicated connection per user that runs in its own persistent goroutine.
      Its only job is to log in, `SELECT` `Inbox`, and run the `IDLE` command.
        * If this connection drops (which it will, due to network timeouts), the `client.Idle()` command in the
          goroutine returns an error. The code catches this error, logs it,
          waits 5–10 seconds (uses exponential backoff), and then reconnects and re-issues the IDLE command.
* Provides WebSocket connections for clients for email push. When the IDLE goroutine gets a push, it finds
  the user's WebSocket connection and sends a JSON message like `{"type": "new_email", "folder": "Inbox"}`.

## Front end

* Onboarding
    * Problem: When a user first logs in via Authelia, the API will validate their token, but `users` is empty for them.
      The API has no IMAP settings to use. The app is unusable.
    * Solution:
        * Go's "get user" endpoint returns a `not_set_up` flag (or `null` if that's unidiomatic).
        * The front end redirects to `/settings`.
        * This page has fields for: IMAP Host, IMAP User, IMAP Password, SMTP Host, SMTP User, SMTP Password,
          Undo Send Delay (20s), Pagination Threads Per Page (100).
        * The back end encrypts the data and creates the `users` and `user_settings` records.
* IMAP email loading and SMTP sending/replying.
* WebSocket-based real-time email fetching
    * The app opens a WebSocket connection to the API.
      When the front end gets a message like `{"type": "new_email", "folder": "Inbox"}`, it invalidates
      the TanStack Query cache for the inbox, triggering TanStack Query to GET `/api/v1/threads?folder=Inbox`.
      The new email appears almost instantly.
* Threaded view with thread-count display, such as `Sender Name (3)`. The server does the threading itself.
* Search. Including some Gmail-like syntax, for example `from:george after:2025`. Search itself happens on the server.
* Pagination (100 emails/page).
* Star, Archive, Trash actions.
* Multi-select with checkboxes.
* "Undo send" (20-second delay).
* Auto-saving drafts.
* Basic offline support (read-only cache of viewed emails via `TanStack Query` persistence).
* Periodic auto-sync and connection status indicator.
* Settings page.
    * Has fields for: IMAP Host, IMAP User, IMAP Password, SMTP Host, SMTP User, SMTP Password, Undo send delay,
      Pagination: threads per page.
* URL-based routing.
* Keyboard shortcuts.
* Logout.


### UI

* **Main layout:** Functionally similar to Gmail but aesthetically distinct (fonts, colors, logos).
* **Top:** Persistent search bar.
* **Left sidebar:** Main navigation links:
    * `Inbox`
    * `Starred`
    * `Sent`
    * `Drafts`
    * `Spam`
    * `Trash`
* **Bottom:** Footer placeholder (e.g., "Copyright 2025 V-Mail").
* **Email list view:** A paginated list of email threads. Each row shows: checkbox, star icon, sender(s), subject,
  thread count, date.
* **Email thread view:** Replaces the list view when the user clicks a thread.
  Shows all messages in the thread, expanded. Displays attachments and Reply/Forward actions.
