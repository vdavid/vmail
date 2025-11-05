# V-Mail

A fast, self-hosted webmail client with a familiar, keyboard-driven UI.

## Overview

V-Mail is a self-hosted, web-based email client designed for personal use.
It uses the layout and keyboard shortcuts of Gmail to make it immediately familiar for ex-Gmail users.
It connects to an IMAP server (tested with mailcow) and provides the web UI to read and send email.

I built V-Mail with the explicit legal constraint to **not** use any of Google's proprietary assets (fonts, icons, logos)
or aesthetic design. The focus is on **functional parity**, not visual imitation, to avoid any brand confusion.

## Running

- Install [Authelia](https://www.authelia.com) and run it. Get its hostname, you'll need it.
- Get your IMAP and SMTP credentials ready.
- Clone this repo by `git clone git@github.com:vdavid/vmail.git && cd vmail`.
- Run `cp .env.example .env` to create a private env file.
- Edit the `.env` file and follow the instructions inside.
- Make sure you have **Docker** and **Docker Compose** installed.
- Run `docker compose up -d --build` to start the services.
- Open `http://localhost:8080` in the browser.
- Log in with your Authelia credentials.

## Non-goals

Compared to Gmail, this project does **not** include:

* Client-side email filters. The user should set these up on the server, typically via [Sieve](http://sieve.info/).
* A visual query builder for the search box. A simple text field is fine.
* A multi-language UI. The UI is English-only.
* 95% of Gmail's settings. V-Mail has some basic settings like emails per page and undo send delay, but that's it.
* Automatic categorization such as primary/social/promotions.
* The ability to collapse the left sidebar.
* Signature management.
* Smiley/emoji reactions to emails. This is Google's proprietary thing.

## Tech stack

V-Mail uses a **Go** back end, a **REST** API, and a **React** front end with **TypeScript**.
It uses a **Postgres** database for caching, drafts, and settings.
V-Mail does **not** handle authentication. A separate, self-hosted [Authelia](https://www.authelia.com) instance is responsible for that.

### IMAP server

V-Mail works with modern IMAP servers, **mailcow** (using Dovecot under the hood) being the primary target.
It has two **hard requirements** for the IMAP server:

1.  **`THREAD` Extension ([RFC 5256](https://datatracker.ietf.org/doc/html/rfc5256)):** Server-side threading is mandatory.
    V-Mail will not implement client-side threading.
2.  **Full-Text Search (FTS):** The server must support fast, server-side `SEARCH` commands.
    Standard IMAP `SEARCH` is part of the core protocol, but V-Mail's performance relies on the server's FTS capabilities,
    like those in Dovecot.

### Authelia

**Authelia** ([authelia.com](https://www.authelia.com/)) is responsible for authentication.
It's an [open-source](https://github.com/authelia/authelia), Go-based single sign-on (SSO) and 2FA server.

**Interaction flow:** The V-Mail front end redirects the user to Authelia for login.
After successful login, Authelia provides a session token, a JWT, which the front end stores in the browser.
After this, all API requests from the front end to the Go back end will include this token.
The back end validates the token before processing requests.

## Security

We designed the project with security in mind.
However, you are responsible for regularly backing up the database to avoid data loss. The emails themselves
live on the IMAP server, but offline drafts and settings are in the database.

## Keyboard shortcuts

We designed the app to be fully usable via a subset of Gmail's shortcuts.

* **Navigation:**
    * `j` / `↓`: Move cursor to next email in list / next message in thread.
    * `k` / `↑`: Move cursor to previous email in list / previous message in thread.
    * `o` / `Enter`: Open the selected thread.
    * `u`: Go back to the list view (from a thread).
    * `g` then `i`: Go to inbox.
    * `g` then `s`: Go to starred.
    * `g` then `t`: Go to sent.
    * `g` then `d`: Go to drafts.
* **Actions:**
    * `c`: Compose new email.
    * `r`: Reply (to sender).
    * `a`: Reply all.
    * `f`: Forward.
    * `e`: Archive selected.
    * `s`: Star/unstar selected.
    * `#` (Shift+3): Move to trash (delete).
    * `z`: Undo last action.
    * `/`: Focus search bar.
* **Selection (in list view):**
    * `x`: Toggle selection on the focused email.
    * `*` then `a`: Select all.
    * `*` then `n`: Select none.
    * `*` then `r`: Select read.
    * `*` then `u`: Select unread.
    * `*` then `s`: Select starred.
    * `*` then `t`: Select unstarred.

## Roadmap

1.  **Milestone 1: The IMAP spike**
    * Goal: Prove the core technology works. 
    * Tasks: Just a Go CLI app. Log in, THREAD, SEARCH, FETCH. No UI.
2.  **Milestone 2: Read-only V-Mail (MVP)**
    * Goal: A read-only, online-only client. 
    * Tasks:
      * Set up auth.
      * Build the Go API for reading (threads, messages).
      * Build the React UI (layout, sidebar, list, thread view).
      * Implement j/k/o/u navigation.
      * No sending, no offline.
      * Create Settings page with reading/writing fields.
      * Build onboarding flow.
3.  **Milestone 3: Actions**
    * Goal: Be able to manage email. 
    * Tasks: Implement Archive, Star, Trash (both frontend and backend). Implement the search bar UI to call the search API.
4.  **Milestone 4: Composing**
    * Goal: Be able to send email. 
    * Tasks: Build composer UI. Implement SMTP logic on the backend. Implement Reply/Forward. Implement "Undo Send."
5.  **Milestone 5: Quality of life**
    * Goal: Polish the MVP. 
    * Tasks: Auto-save drafts. Add keyboard shortcuts. Add pagination. Add IDLE and WebSocket connection.
6.  **Milestone 6: Offline**
    * Goal: Basic offline support.
    * Tasks: Implement IndexedDB caching for recently viewed emails. Build the sync logic.

### Milestone 1

**Goal:** Prove the core technology works. A simple Go CLI app. No UI.

* [x] Set up a new Go module (`go mod init backend`).
* [x] Add `github.com/emersion/go-imap` as a dependency.
* [x] Create a `main.go` file.
* [x] Implement logic to connect to the mailcow IMAP server (using `imap.DialTLS`).
* [x] Implement logic to log in using a username and password (from env vars for now).
* [x] Implement a function to run the `CAPABILITY` command and print the results (to verify `THREAD` support).
* [x] Implement a function to `SELECT` the "Inbox".
* [x] Implement a function to run a `THREAD` command (`THREAD=REFERENCES UTF-8 ALL`) and print the raw response.
* [x] Implement a function to run a `SEARCH` command (e.g., `SEARCH FROM "test"`) and print the resulting UIDs.
* [x] Implement a function to `FETCH` a single message (using a UID from the search) and print its body structure and headers.

Done! 🎉 It works nicely. It's in `/backend/cmd/spike`. See `/backend/README.md` for details on milestone 1.

### Milestone 2

#### **2/1. 🏗️ Backend: Server foundation**

* [x] **Create the main server:** In `/backend/cmd/server`, create a new `main.go`. This will be your *actual* server (unlike the `spike`).
    * It should start a `net/http` server using `http.ListenAndServe`.
    * It should use `http.ServeMux` (as specified in your spec) for routing.
    * Add a simple `http.HandlerFunc` for `/` that responds with "V-Mail API is running".
* [x] **Set up config loading:** In `/backend/internal/config`, create a `config.go`.
    * Create a `struct` that holds all env vars (DB host, master key, etc.).
    * Create a `NewConfig()` function that reads from the `.env` file (using `godotenv` for local dev) and `os.Getenv` (for production).
    * Pass this `Config` struct to your server in `main.go`.
* [x] **Set up database connection:** In `/backend/internal/db`, create a `db.go`.
    * Create a `NewConnection()` function that takes the DB config and returns a `*pgxpool.Pool`.
    * Add this `*pgxpool.Pool` to your server's dependencies.
* [x] **Set up DB migrations:**
    * Install `golang-migrate` (e.g., `brew install golang-migrate`).
    * Create a `/backend/migrations` directory.
    * Create a new migration file (e.g., `migrate create -ext sql -dir backend/migrations -seq init_schema`).
    * Copy-paste the entire SQL schema from `SPEC.md` into the `.up.sql` file.
    * Run the migration (`migrate -database "..." -path backend/migrations up`) to create your tables.

### **2/2. 🛡️ Backend: Auth and onboarding**

* [x] **Create Authelia middleware:** In `/backend/internal/api` (or `/internal/auth`), create a `middleware.go`.
    * Create a `RequireAuth` middleware.
    * This middleware should:
        * Get the `Authorization: Bearer ...` token from the request header.
        * (For now) Log the token. In a later step, you'll validate it.
        * Pass the request to the next handler.
* [x] **Create API: <code>auth/status</code> endpoint:**
    * Add the `GET /api/v1/auth/status` route.
    * Create its handler function. This function should:
        * (For now) Assume auth is okay.
        * Check if a row exists in `user_settings` for this user.
        * Return `{"isAuthenticated": true, "isSetupComplete": [true/false]}`.
* [x] **Create API: <code>settings</code> endpoints:**
    * In `/backend/internal/db`, create `user_settings.go`. Add `GetUserSettings(userID string)` and `SaveUserSettings(settings UserSettings)` functions.
    * Add the `GET /api/v1/settings` route and handler. It should call `GetUserSettings` and return the data (without passwords).
    * Add the `POST /api/v1/settings` route and handler.
        * It must read the JSON body.
        * It must **encrypt** the `imap_password` and `smtp_password` fields (using your standard `crypto/aes` logic).
        * It should call `SaveUserSettings` to save the data to the DB.

### **2/3. 🎨 Frontend: Skeleton and settings page**

* [ ] **Set up the React project:**
    * In the root `/frontend` folder, run `pnpm create vite . --template react-ts`. 
    * Install all your core dependencies:
      ```bash
      pnpm install react-router-dom @tanstack/react-query zustand dompurify
      pnpm install -D tailwindcss postcss autoprefixer
      ```
    * Initialize Tailwind (`pnpm tailwindcss init -p`).
* [ ] **Create the basic layout:**
    * In `/frontend/src`, create a `components/Layout.tsx`.
    * `Layout.tsx` should have a static `Sidebar.tsx` (left), `Header.tsx` (top, for search), and a main content area that renders `{children}`.
* [ ] **Set up routing:**
    * In `main.tsx`, wrap your app in `&lt;BrowserRouter>`.
    * Create `App.tsx` to define your routes:
        * `/` (goes to `Inbox.page.tsx`)
        * `/thread/:threadId` (goes to `Thread.page.tsx`)
        * `/settings` (goes to `Settings.page.tsx`)
* [ ] **Create Auth/Onboarding flow:**
    * Create an "auth" store in `store/auth.store.ts` (using Zustand). It should hold `isSetupComplete` (boolean, default `false`).
    * Create a `components/AuthWrapper.tsx` component.
        * This component uses `useEffect` on mount to `fetch` your `GET /api/v1/auth/status` endpoint.
        * When it gets the response, it sets the `isSetupComplete` state in your Zustand store.
        * It should render `{children}` *only if* `isSetupComplete` is true.
        * If `isSetupComplete` is `false`, it should render `&lt;Navigate to="/settings" />` (from `react-router-dom`).
    * Wrap your main `&lt;Layout />` in `AuthWrapper.tsx`.
* [ ] **Build Settings Page:**
    * Create `pages/Settings.page.tsx`.
    * This page should be a simple form with fields for all the `user_settings` (IMAP server, username, password, folder names, etc.).
    * Use `TanStack Query` to fetch data from `GET /api/v1/settings` to populate the form.
    * Use `TanStack Query`'s `useMutation` hook to `POST` the form data to `POST /api/v1/settings` on submit.

### **2/4. 📨 Backend: Read-only email API**

* [ ] **Refactor <code>spike</code> code:** Move your `connectToIMAP`, `login`, `runThreadCommand`, etc. from the `spike` into reusable functions in `/backend/internal/imap`.
    * Create an `imap/client.go` that manages a **connection pool** (as discussed). This is complex, so start simple: just a `map[string]*client.Client` to hold one connection per user.
* [ ] **Create API: <code>folders</code> endpoint:**
    * Add the `GET /api/v1/folders` route and handler.
    * The handler should:
        1. Get the user's IMAP credentials from the DB.
        2. Get an IMAP connection from your pool.
        3. Run the IMAP `LIST` command to get all folders.
        4. Return them as a JSON array: `[{"name": "INBOX"}, {"name": "Sent"}]`.
* [ ] **Create API: <code>threads</code> endpoint:**
    * Add the `GET /api/v1/threads` route (it needs a query param, e.g., `?folder=INBOX`).
    * This is the most complex handler:
        1. Get user credentials.
        2. Check the DB cache (as discussed, based on a TTL).
        3. If cache is stale: Get IMAP connection, run `THREAD` (like in your spike), then `FETCH` headers for all messages in those threads.
        4. Parse the messages (using `enmime` for headers).
        5. Save the data to your `threads` and `messages` tables.
        6. Return the list of threads from your **database**.
* [ ] **Create API: <code>thread/:thread_id</code> endpoint:**
    * Add the `GET /api/v1/thread/:thread_id` route.
    * This handler should:
        1. Query your *database* for the thread (using `stable_thread_id`).
        2. Fetch all messages for that thread from your `messages` table.
        3. (If messages are missing bodies) Fetch the full message bodies from IMAP.
        4. Parse with `enmime`, saving `unsafe_body_html` and `attachments` to the DB.
        5. Return the full thread with all messages and attachments as JSON.

### **5. 🖥️ Frontend: Read-only email UI**

* [ ] **Render folders:**
    * In `Sidebar.tsx`, use `TanStack Query` (`useQuery`) to fetch from `GET /api/v1/folders`.
    * Render the list of folders as links (`&lt;Link to="/?folder=INBOX">Inbox&lt;/Link>`).
* [ ] **Render thread list:**
    * Create `pages/Inbox.page.tsx`.
    * It should read the `?folder=` query param from the URL (using `react-router`'s `useSearchParams` hook).
    * Use `useQuery` to fetch from `GET /api/v1/threads?folder=...`.
    * Create an `EmailListItem.tsx` component.
    * Render the list of threads using this component, showing sender, subject, date, etc.
* [ ] **Render thread view:**
    * Create `pages/Thread.page.tsx`.
    * It should read the `:threadId` from the URL (using `useParams`).
    * Use `useQuery` to fetch from `GET /api/v1/thread/:threadId`.
    * Create a `Message.tsx` component.
    * Render each message in the thread.
    * **Crucially:** In `Message.tsx`, use `DOMPurify.sanitize()` on the `unsafe_body_html` before rendering it with `dangerouslySetInnerHTML`.
    * Render the list of attachments.
* [ ] **Implement basic keyboard navigation:**
    * Create a `hooks/useKeyboardShortcuts.ts`.
    * This hook should `useEffect` to add a `keydown` event listener.
    * (For now) Just implement `j` (next item) and `k` (previous item) to move a "selected" index, which you'll store in a new Zustand store (`ui.store.ts`).
    * Implement `o` (open) or `Enter` to navigate to the selected thread (using `react-router`'s `useNavigate` hook).
    * Implement `u` (up) to navigate from a thread view back to the inbox (`Maps('/')`).

### Later
* [ ] Write a doc for how to create a daily DB backup, e.g., via a `pg_dump` cron job.