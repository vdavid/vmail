# Roadmap

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

## Milestone 1

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

## Milestone 2

### **2/1. 🏗️ Backend: Server foundation**

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

* [x] **Set up the React project:**
    * In the root `/frontend` folder, run `pnpm create vite . --template react-ts`.
    * Install all your core dependencies:
      ```bash
      pnpm install react-router-dom @tanstack/react-query zustand dompurify
      pnpm install -D tailwindcss postcss autoprefixer
      ```
    * Initialize Tailwind (`pnpm tailwindcss init -p`).
* [x] **Create the basic layout:**
    * In `/frontend/src`, create a `components/Layout.tsx`.
    * `Layout.tsx` should have a static `Sidebar.tsx` (left), `Header.tsx` (top, for search), and a main content area that renders `{children}`.
* [x] **Set up routing:**
    * In `main.tsx`, wrap your app in `<BrowserRouter>`.
    * Create `App.tsx` to define your routes:
        * `/` (goes to `Inbox.page.tsx`)
        * `/thread/:threadId` (goes to `Thread.page.tsx`)
        * `/settings` (goes to `Settings.page.tsx`)
* [x] **Create Auth/Onboarding flow:**
    * Create an "auth" store in `store/auth.store.ts` (using Zustand). It should hold `isSetupComplete` (boolean, default `false`).
    * Create a `components/AuthWrapper.tsx` component.
        * This component uses `useEffect` on mount to `fetch` your `GET /api/v1/auth/status` endpoint.
        * When it gets the response, it sets the `isSetupComplete` state in your Zustand store.
        * It should render `{children}` *only if* `isSetupComplete` is true.
        * If `isSetupComplete` is `false`, it should render `<Navigate to="/settings" />` (from `react-router-dom`).
    * Wrap your main `<Layout />` in `AuthWrapper.tsx`.
* [x] **Build Settings Page:**
    * Create `pages/Settings.page.tsx`.
    * This page should be a simple form with fields for all the `user_settings` (IMAP server, username, password, folder names, etc.).
    * Use `TanStack Query` to fetch data from `GET /api/v1/settings` to populate the form.
    * Use `TanStack Query`'s `useMutation` hook to `POST` the form data to `POST /api/v1/settings` on submit.

### **2/4. 📨 Backend: Read-only email API**

* [x] **Refactor <code>spike</code> code:** Move your `connectToIMAP`, `login`, `runThreadCommand`, etc. from the `spike` into reusable functions in `/backend/internal/imap`.
    * Create an `imap/client.go` that manages a **connection pool** (as discussed). This is complex, so start simple: just a `map[string]*client.Client` to hold one connection per user.
* [x] **Create API: <code>folders</code> endpoint:**
    * Add the `GET /api/v1/folders` route and handler.
    * The handler should:
        1. Get the user's IMAP credentials from the DB.
        2. Get an IMAP connection from your pool.
        3. Run the IMAP `LIST` command to get all folders.
        4. Return them as a JSON array: `[{"name": "INBOX"}, {"name": "Sent"}]`.
* [x] **Create API: <code>threads</code> endpoint:**
    * Add the `GET /api/v1/threads` route (it needs a query param, e.g., `?folder=INBOX`).
    * This is the most complex handler:
        1. Get user credentials.
        2. Check the DB cache (as discussed, based on a TTL).
        3. If cache is stale: Get IMAP connection, run `THREAD` (like in your spike), then `FETCH` headers for all messages in those threads.
        4. Parse the messages (using `enmime` for headers).
        5. Save the data to your `threads` and `messages` tables.
        6. Return the list of threads from your **database**.
* [x] **Create API: <code>thread/:thread_id</code> endpoint:**
    * Add the `GET /api/v1/thread/:thread_id` route.
    * This handler should:
        1. Query your *database* for the thread (using `stable_thread_id`).
        2. Fetch all messages for that thread from your `messages` table.
        3. (If messages are missing bodies) Fetch the full message bodies from IMAP.
        4. Parse with `enmime`, saving `unsafe_body_html` and `attachments` to the DB.
        5. Return the full thread with all messages and attachments as JSON.

### **2/5. 🖥️ Frontend: Read-only email UI**

* [ ] **Render folders:**
    * In `Sidebar.tsx`, use `TanStack Query` (`useQuery`) to fetch from `GET /api/v1/folders`.
    * Render the list of folders as links (`<Link to="/?folder=INBOX">Inbox</Link>`).
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

#### **Testing: Unit Tests (Jest + React Testing Library)**

* [ ] **<code>Message.tsx</code> (Security):**
    * Test that the component *always* calls `DOMPurify.sanitize()` with the `unsafe_body_html` prop.
    * Test that the output of `DOMPurify.sanitize` is what's actually rendered via `dangerouslySetInnerHTML`.
* [ ] **<code>hooks/useKeyboardShortcuts.ts</code>:**
    * Mock `window.addEventListener` and `window.removeEventListener` to test that they are called on mount/unmount.
    * Test that pressing "j" calls the function to increment the selected index.
    * Test that pressing "k" calls the function to decrement the selected index.
    * Test that pressing "o" or "Enter" calls the `react-router` `Maps` function.
    * Test that pressing "u" calls the `Maps` function to go back.

#### **Testing: Integration Tests (React Testing Library + <code>msw</code>)**

* [ ] **Mock API:** Set up `msw` (Mock Service Worker) to intercept and mock all API calls (`GET /api/v1/folders`, `GET /api/v1/threads`, `GET /api/v1/thread/:threadId`).
* [ ] **<code>Sidebar.tsx</code>:**
    * Test that it renders a "Loading..." state.
    * Test that it calls `GET /api/v1/folders`.
    * Test that it renders a list of links (e.g., "Inbox", "Sent") based on the mock API response.
    * Test that clicking the "Sent" link navigates the user to `/?folder=Sent`.
* [ ] **<code>Inbox.page.tsx</code> (Thread List):**
    * Test that it renders a "Loading..." state.
    * Test that it reads the `?folder=INBOX` URL parameter and calls the correct API: `GET /api/v1/threads?folder=INBOX`.
    * Test that it renders the list of `EmailListItem` components based on the mock response.
    * Test that clicking an `EmailListItem` navigates the user to the correct thread (e.g., `/thread/thread-id-123`).
* [ ] **<code>Thread.page.tsx</code> (Thread View):**
    * Test that it renders a "Loading..." state.
    * Test that it reads the `:threadId` URL parameter and calls the correct API: `GET /api/v1/thread/thread-id-123`.
    * Test that it correctly renders all messages, sender names, subjects, and attachment filenames from the mock response.

### **2/6. 🧪 Test Plan: Milestone 2 (End-to-End)**

This plan uses **Playwright** to test the entire read-only flow, assuming the backend and frontend are running.

* [ ] **Test 1: New User Onboarding Flow**
    * Mock your Authelia login to succeed for a *new user*.
    * Start at the app's root URL.
    * **Assert** the app redirects to the `/settings` page.
    * Fill in all the form fields (IMAP server, user, pass, etc.).
    * Click "Save".
    * **Assert** the app redirects to the Inbox (`/`).
    * (Optional DB check): `SELECT` from `user_settings` and `users` to verify the user was created and the passwords are *encrypted*.
* [ ] **Test 2: Existing User Read-Only Flow**
    * Log in as an *existing* (already set up) user.
    * **Assert** the app lands on the Inbox (`/`).
    * **Assert** the sidebar populates with folders (e.g., "Inbox", "Sent").
    * **Assert** the main view populates with a list of email threads.
    * Note the subject of the first email, then click it.
    * **Assert** the URL changes to `/thread/some-id`.
    * **Assert** the full email body and any attachment names are visible on the screen.
* [ ] **Test 3: Navigation**
    * From the thread view, press the "u" key.
    * **Assert** the app navigates back to the Inbox (`/`).
    * Press the "j" key.
    * **Assert** the visual focus/selection moves to the *second* email in the list.
    * Press the "o" key.
    * **Assert** the app navigates to the *second* thread's page.

### **Front end cleanup**

* [ ] Do the stuff from frontend/README.md and get rid of that file.

## Milestone 3: Actions

- Goal: Be able to manage email.
- Tasks: Implement Archive, Star, Trash (both frontend and backend). Implement the search bar UI to call the search API.

This milestone is all about adding *actions*.
We'll build them one by one.
The "Star" action is the simplest, so we'll start there.
We'll use the `action_queue` for all of them to make the UI feel instant.

### **3/1. 🏗️ Backend: Action worker**

Before we can *queue* any actions, we need to build the worker that *processes* them.

* [ ] **Create the worker service:**
    * In `/backend/internal/sync`, create a new `worker.go` file.
    * Create a `struct` for your worker, e.g., `type ActionWorker struct { db *pgxpool.Pool, imapService *imap.Service }`.
      (You'll need to create an `imap.Service` struct in your `/internal/imap` folder to hold your IMAP logic).
* [ ] **Create the worker loop:**
    * In `worker.go`, create a `Start()` method. This method should be run as a goroutine from your `main.go`.
    * Inside `Start()`, use a `time.NewTicker` (e.g., every 5 seconds) to wake up and poll for jobs.
    * In the loop, call a `processJobs()` function.
* [ ] **Create the job processor:**
    * In `worker.go`, create `processJobs()`.
    * It should query the DB: `SELECT id, action_type, payload FROM action_queue WHERE process_at &lt;= NOW() FOR UPDATE SKIP LOCKED`.
    * It should loop over the returned rows.
    * Use a `switch item.action_type` to handle different jobs. For now, it will be empty.
    * After a job is processed *successfully*, it must `DELETE FROM action_queue WHERE id = $1`.
* [ ] **Add unit tests:**
    * Test the `processJobs` function in isolation.
    * Mock the database: Make your `SELECT ... FOR UPDATE` mock return a list of 2-3 sample jobs (e.g., one "star", one "move").
    * Mock the IMAP service.
    * **Assert** that the correct IMAP service methods (e.g., `StarThread`, `MoveThread`) are called with the exact payloads from the mock jobs.
    * **Assert** that the `DELETE FROM action_queue` command is called for each *successfully* processed job.
* [ ] **Add integration tests (Go + Testcontainers):**
    * Start a real Postgres DB using `testcontainers-go`.
    * Mock *only* the IMAP service.
    * Manually `INSERT` 2-3 jobs into the `action_queue` table.
    * Start your `ActionWorker`.
    * Wait ~100ms.
    * **Assert** your mock IMAP service methods were called.
    * Query the database: **Assert** the processed jobs have been deleted from the `action_queue` table.

### **3/2. ⭐️ Feature: Star/Unstar**

#### **Backend (Star)**

* [ ] **Create the "Star" IMAP logic:**
    * In `/backend/internal/imap`, create a new `actions.go` file.
    * Create a function: `func (s *Service) StarThread(ctx context.Context, userID string, threadStableID string, starStatus bool) error`.
    * This function needs to:
        1. Get the user's settings (for credentials) from the DB.
        2. Get an IMAP connection from your pool.
        3. Find all messages in our DB for that thread: `SELECT imap_uid, imap_folder_name FROM messages WHERE thread_id = (SELECT id FROM threads WHERE stable_thread_id = $1 AND user_id = $2)`.
        4. Loop through each message, `c.Select(folderName, ...)`
        5. Build a `seqSet` for the `imap_uid`.
        6. Run the IMAP command: `c.UidStore(seqSet, flagOp, []interface{}{imap.FlaggedFlag})` (where `flagOp` is `"+FLAGS.SILENT"` or `"-FLAGS.SILENT"`).
        7. Update our *own* DB to match: `UPDATE messages SET is_starred = $1 WHERE ...`.
* [ ] **Create the <code>actions</code> API endpoint:**
    * In `/backend/internal/api/routes.go`, add the route: `router.Post("/api/v1/actions", app.actionsHandler)`.
    * Create a new `/backend/internal/api/actions_handler.go` file.
    * The `actionsHandler` should:
        1. Read the JSON body, e.g., `{"action": "star_thread", "payload": {"thread_stable_id": "...", "star_status": true}}`.
        2. Check the `action` type.
        3. Call a new function `db.QueueAction(...)` to save the job to the `action_queue` table with `process_at = NOW()`.
        4. Return `202 Accepted` to the frontend immediately.
* [ ] **Hook up the worker:**
    * In `worker.go`, add a `case "star_thread":` to your `switch` statement.
    * It should parse the JSON payload, then call your new `imapService.StarThread(...)`.

#### **Frontend (Star)**

* [ ] **Create the <code>starThread</code> mutation:**
    * In `/frontend/src/api/` (or a new `api/email.api.ts`), create a `useStarThreadMutation` hook using `TanStack Query`'s `useMutation`.
    * This mutation will `POST` to `/api/v1/actions` with the `star_thread` payload.
* [ ] **Add optimistic updates:**
    * This is key for a fast UI. In your `useStarThreadMutation`, use the `onMutate` option to *optimistically update* the query cache *before* the API call runs.
    * You'll use `queryClient.setQueryData(...)` to find the thread in the `'threads'` query cache and manually set its `is_starred` status.
    * Add `onError` to roll back the change if the mutation fails.
    * Add `onSettled` to `queryClient.invalidateQueries(['threads'])` to refetch the real data.
* [ ] **Add the UI button:**
    * In `EmailListItem.tsx`, add a star icon button.
    * `onClick` should call `starThreadMutation.mutate({ thread_stable_id: ..., star_status: !thread.is_starred })`.
    * The star's appearance (filled vs. outline) should be based on the `thread.is_starred` prop.

#### **Testing**

* [ ] **Backend Unit (Go):**
    * **<code>actions_handler</code>:** Test that a `POST /api/v1/actions` request with a "star_thread" body results in a new row being inserted into `action_queue` with the correct `action_type` and `payload`.
    * **<code>imapService</code>:** Test the `StarThread` function. Mock the DB and the IMAP client.
        * **Assert** it correctly fetches messages from the mock DB.
        * **Assert** the mock IMAP client's `UidStore` method is called with `+FLAGS.SILENT` for `starStatus: true`.
        * **Assert** the mock IMAP client's `UidStore` method is called with `-FLAGS.SILENT` for `starStatus: false`.
        * **Assert** it updates the `messages` table in the mock DB.
* [ ] **Frontend Unit (RTL):**
    * Test the `EmailListItem` component.
    * Mock the `useStarThreadMutation` hook.
    * **Assert** the star icon is "outline" when `is_starred` is false.
    * Click the star button.
    * **Assert** the `mutate` function from the mock hook was called with `{ ..., star_status: true }`.
* [ ] **Frontend Integration (RTL + <code>msw</code>):**
    * **Test Optimistic Update:**
        * Mock the `POST /api/v1/actions` API to have a 1-second delay.
        * Render the `EmailListItem` (with `is_starred: false`) inside a `QueryClientProvider`.
        * Click the star button.
        * **Immediately assert** that the star icon *changes to "filled"* (even before the API mock resolves).
        * Wait for the API to resolve.
        * **Assert** the star icon *remains "filled"*.

### **3/3. 🗃️ Feature: Archive & Trash**

These are identical, just with a different destination. We'll build them as a generic "move" action.

#### **Backend (Move)**

* [ ] **Create the "Move" IMAP logic:**
    * In `/backend/internal/imap/actions.go`, create: `func (s *Service) MoveThread(ctx context.Context, userID string, threadStableID string, destinationFolder string) error`.
    * Logic:
        1. Get user settings and IMAP connection.
        2. Get all messages for the thread from *our* DB (`SELECT imap_uid, imap_folder_name FROM messages WHERE ...`).
        3. Loop through each message, `c.Select(folderName, ...)`.
        4. Run the IMAP command: `c.UidMove(seqSet, destinationFolder)`.
        5. Update our DB: `UPDATE messages SET imap_folder_name = $1 WHERE ...`.
* [ ] **Update the <code>actionsHandler</code>:**
    * Add logic to handle a new `action_type: "move_thread"`.
    * The payload should be `{"thread_stable_id": "...", "destination_folder": "Archive"}` (or "Trash").
    * It should queue this in the `action_queue` table.
* [ ] **Hook up the worker:**
    * In `worker.go`, add `case "move_thread":` to your `switch`.
    * It should parse the payload and call `imapService.MoveThread(...)`.

#### **Frontend (Move)**

* [ ] **Create the <code>moveThread</code> mutation:**
    * In `email.api.ts`, create a `useMoveThreadMutation` hook.
    * It `POST`s to `/api/v1/actions` with the `move_thread` payload.
* [ ] **Add UI buttons:**
    * In `EmailListItem.tsx` (e.g., on hover) and `ThreadView.tsx`, add "Archive" and "Trash" icon buttons.
    * `onClick` on "Archive" calls `moveThreadMutation.mutate({ ..., destination_folder: 'Archive' })`.
    * `onClick` on "Trash" calls `moveThreadMutation.mutate({ ..., destination_folder: 'Trash' })`.
* [ ] **Update cache on success:**
    * This is *not* an optimistic update.
    * In the `onSuccess` callback of your `useMoveThreadMutation`, you must **invalidate the cache** for the list you're looking at.
    * `queryClient.invalidateQueries({ queryKey: ['threads', currentFolder] })`. This will trigger a refetch, and the item will disappear from the list.

#### **Testing**

* [ ] **Backend Unit (Go):**
    * **<code>actions_handler</code>:** Test that a `POST` request with a "move_thread" body (payload `destination_folder: 'Archive'`) queues the correct job.
    * **<code>imapService</code>:** Test the `MoveThread` function. Mock the DB and IMAP client.
        * **Assert** it correctly fetches messages.
        * **Assert** the mock IMAP client's `UidMove` method is called with the correct `seqSet` and destination folder.
        * **Assert** it updates the `messages` table in the mock DB with the new `imap_folder_name`.
* [ ] **Frontend Integration (RTL + <code>msw</code>):**
    * **Test Cache Invalidation:**
        * Mock the `GET /api/v1/threads?folder=INBOX` to return 3 items.
        * Render the `Inbox.page.tsx`. **Assert** 3 items are visible.
        * Mock the `POST /api/v1/actions` API to succeed.
        * Mock `queryClient.invalidateQueries` to track calls.
        * Click the "Archive" button on the first item.
        * **Assert** `invalidateQueries` was called with the `['threads', 'INBOX']` query key.

### **3/4. 🔎 Feature: Search**

This is a read-only feature, so it doesn't use the action queue.

#### **Backend (Search)**

* [ ] **Create the "Search" IMAP logic:**
    * In `/backend/internal/imap/read.go` (or a new `search.go`), create: `func (s *Service) Search(ctx context.Context, userID string, query string) ([]Thread, error)`.
    * This function needs to:
        1. Get user settings and IMAP connection.
        2. `c.Select("INBOX", ...)` (Keep it simple: only search `INBOX` for now).
        3. Build IMAP criteria: `criteria := imap.NewSearchCriteria()` -> `criteria.Text = []string{query}`.
        4. Run `uids, err := c.UidSearch(criteria)`.
        5. **This is the hard part:** You now have UIDs, but the frontend needs *threads*.
        6. You'll have to `FETCH` the headers for these UIDs, find their `Message-ID`s, and then group them by `stable_thread_id` (similar to your `threads` endpoint logic, but based on a `SEARCH` result, not a `THREAD` result).
* [ ] **Create the <code>search</code> API endpoint:**
    * In `routes.go`, add: `router.Get("/api/v1/search", app.searchHandler)`.
    * Create `api/search_handler.go`.
    * The `searchHandler` should:
        1. Get the query: `q := r.URL.Query().Get("q")`.
        2. Call your new `imapService.Search(...)`.
        3. Return the resulting list of threads (in the *same JSON format* as `GET /api/v1/threads`).

#### **Frontend (Search)**

* [ ] **Create the search results page:**
    * Create a new page: `pages/Search.page.tsx`.
    * Add the route in `App.tsx`: `&lt;Route path="/search" element={&lt;SearchPage />} />`.
* [ ] **Hook up the search bar:**
    * In `Header.tsx`, make the search input a controlled component.
    * On form submit (or `Enter` key), use `useNavigate` from `react-router-dom` to navigate: `Maps(`/search?q=${query}`)`.
* [ ] **Fetch and display results:**
    * In `Search.page.tsx`, use the `useSearchParams` hook to get the `q` param from the URL.
    * Use `TanStack Query`'s `useQuery` to fetch from the backend: `useQuery({ queryKey: ['search', q], queryFn: () => fetchSearchResults(q) })`.
    * **Re-use your component:** The page should just map over the results and render your existing `EmailListItem.tsx` component for each thread.

#### **Testing**

* [ ] **Backend Unit (Go):**
    * **<code>search_handler</code>:** Test that a `GET /api/v1/search?q=test` call correctly calls `imapService.Search("test")`.
    * **<code>imapService</code>:** Test the `Search` function. Mock the DB and IMAP client.
        * **Assert** the IMAP client's `UidSearch` method is called with the correct criteria.
        * **Assert** the logic for fetching/grouping UIDs into threads works.
* [ ] **Frontend Integration (RTL + <code>msw</code>):**
    * **<code>Header.tsx</code>:**
        * Mock `react-router`'s `useNavigate` hook.
        * Simulate typing "hello" into the search input and pressing "Enter".
        * **Assert** `Maps` was called with `/search?q=hello`.
    * **<code>Search.page.tsx</code>:**
        * Mock `useSearchParams` to return `q=hello`.
        * Mock the `GET /api/v1/search?q=hello` API.
        * **Assert** the page calls the API and renders the list of `EmailListItem` components from the mock response.

### **3/5. 🧪 Test plan: Milestone 3 (end-to-end)**

This **Playwright** plan tests the full "action" loop. It assumes a pre-populated mock IMAP server.

* [ ] **Test 1: Star a Thread (Full Loop)**
    * Log in and go to the Inbox.
    * Find a thread and **assert** its star is "outline".
    * Click the star.
    * **Assert** the star *immediately* turns "filled" (optimistic UI).
    * Wait 6 seconds (for the worker to run).
    * **Reload the entire page**.
    * Log in again (if needed).
    * **Assert** the *same thread* still shows a "filled" star (proves the backend IMAP & DB update worked).
* [ ] **Test 2: Archive a Thread (Full Loop)**
    * Log in and go to the Inbox.
    * Find a thread and note its subject (e.g., "Project Budget").
    * Click the "Archive" button.
    * **Assert** the "Project Budget" thread *disappears* from the Inbox (tests cache invalidation).
    * Click the "Archive" folder in the sidebar.
    * **Assert** the "Project Budget" thread *appears* in the Archive folder list.
* [ ] **Test 3: Search (Full Loop)**
    * Log in. (Assume a message with subject "Special Report Q3" exists on the mock IMAP server).
    * Type "Special Report" into the search bar and press Enter.
    * **Assert** the URL changes to `/search?q=Special%20Report`.
    * **Assert** the thread "Special Report Q3" is visible in the search results.

## Milestone 4: Composing

- Goal: Be able to send email.
- Tasks: Build composer UI. Implement SMTP logic on the backend. Implement Reply/Forward. Implement "Undo Send."

This milestone is built around the `action_queue` you created in M3.
The "Send" button doesn't send an email; it *queues* a "send" job.
This is what makes "Undo Send" possible.

Breakdown:

### **4/1. 🏗️ Backend: "Send" & "Undo" API endpoints**

We'll start by creating the two API endpoints the frontend needs: one to *queue* a send, and one to *cancel* it.

* [ ] **Create the "Send" API endpoint:**
    * Add the route: `router.Post("/api/v1/send", app.sendHandler)`.
    * Create a new `/backend/internal/api/send_handler.go`.
    * **Handler Logic:**
        * Read the JSON body (to, subject, body, etc.).
        * Get the user's `undo_send_delay_seconds` from their `user_settings` in the DB.
        * Calculate `process_at = NOW() + (delay * time.Second)`.
        * Create the `payload` (a JSONB object of the full email).
        * `INSERT` the job into `action_queue` (`action_type = 'send_email'`, `payload`, `process_at`).
        * **Crucially:** Use `RETURNING id` on your `INSERT` to get the new job's UUID.
        * Return `202 Accepted` with a JSON body: `{"job_id": "..."}`. The frontend needs this ID for the "Undo" button.
* [ ] **Create the "Undo" API endpoint:**
    * Add the route: `router.Delete("/api/v1/send/undo/:jobID", app.undoHandler)`.
    * Create a new `/backend/internal/api/undo_handler.go`.
    * **Handler Logic:**
        * Get `jobID` from the URL parameters.
        * Get `userID` from the auth middleware.
        * Run `DELETE FROM action_queue WHERE id = $1 AND user_id = $2`. (Checking `user_id` is a critical security step).
        * Return `200 OK`.
* [ ] **Backend Unit Tests:**
    * **<code>send_handler</code>:**
        * Test that `POST /api/v1/send` with a valid body inserts a row into `action_queue`.
        * **Assert** the `action_type` is `send_email`.
        * **Assert** the `process_at` time is correct (e.g., `now + 20s`).
        * **Assert** the handler returns a `202` status and the `job_id`.
    * **<code>undo_handler</code>:**
        * `INSERT` a test job for `user_a`.
        * Test that `DELETE /api/v1/send/undo/:jobID` (as `user_a`) deletes the job.
        * Test that `DELETE /api/v1/send/undo/:jobID` (as `user_b`) does **not** delete the job (and returns an error or `404`).

### **4/2. ⚙️ Backend: SMTP & "Sent" folder worker**

Now, we'll teach our M3 `ActionWorker` how to process the `send_email` jobs.

* [ ] **Create SMTP logic:**
    * Create a new folder `/backend/internal/smtp`.
    * In `smtp/client.go`, create a `SendEmail` function: `func SendEmail(settings *UserSettings, payload *EmailPayload) error`.
    * This function should:
        * Use `github.com/go-mail/mail` to build the email message (set "From", "To", "Subject", "HTMLBody").
        * Use `net/smtp` to connect: `smtp.Dial(settings.smtp_server_hostname + ":587")`.
        * Authenticate: `smtp.PlainAuth(...)`.
        * Send: `mail.Send(client, msg)`.
* [ ] **Create IMAP "Append" logic:**
    * In `/backend/internal/imap/actions.go`, create a new function: `func (s *Service) AppendToSent(ctx context.Context, userID string, rawEmailBytes []byte) error`.
    * This function should:
        * Get user settings (for credentials and `sent_folder_name`).
        * Get an IMAP connection.
        * `c.Append(settings.sent_folder_name, ...)` to upload the raw email bytes.
* [ ] **Hook up the worker:**
    * In `/backend/internal/sync/worker.go`, add `case "send_email":` to your `switch`.
    * **Logic:**
        * Parse the `payload`.
        * Call `smtp.SendEmail(...)`.
        * If SMTP fails, log the error and *don't* delete the job (it will retry).
        * If SMTP succeeds, call `imap.AppendToSent(...)` to save a copy.
        * If *both* succeed, `DELETE` the job from `action_queue`.
* [ ] **Backend Unit Tests:**
    * **<code>smtp.SendEmail</code>:** Mock `smtp.Dial` and `smtp.Auth`. Test that the function attempts to connect and send.
    * **<code>imap.AppendToSent</code>:** Mock the IMAP client. Test that `c.Append` is called with the correct `sent_folder_name`.
    * **<code>worker</code>:** Test the `case "send_email"` logic.
        * Mock the SMTP and IMAP services.
        * **Assert** that on SMTP success, *both* SMTP and IMAP methods are called.
        * **Assert** that on SMTP failure, the IMAP method is *not* called.

### **4/3. 🎨 Frontend: "Compose" UI & "Undo" snackbar**

Now, the frontend to create and cancel jobs.

* [ ] **Create <code>composer</code> store:**
    * In `/frontend/src/store`, create `composer.store.ts` (using Zustand).
    * It should hold state: `isOpen` (boolean), `to` (string[]), `subject` (string), `body` (string), `inReplyTo` (Message, optional).
    * Add actions: `openCompose()`, `openReply(msg)`, `openForward(msg)`, `close()`.
* [ ] **Create <code>Composer</code> component:**
    * Create `/frontend/src/components/composer/Composer.tsx`.
    * It renders as a modal or pop-up (fixed to the bottom corner) only if `composer.store.isOpen` is true.
    * It should have `input` fields for "To" and "Subject," and a rich text editor (or `textarea`) for "Body," all bound to the Zustand store.
* [ ] **Create "Send" mutation:**
    * In `api/email.api.ts`, create `useSendEmailMutation`.
    * It `POST`s to `/api/v1/send` with the data from the `composer.store`.
    * `onClick` for the "Send" button calls `sendEmailMutation.mutate()`.
* [ ] **Create "Undo" snackbar/toast:**
    * Create a `store/ui.store.ts` (Zustand) to hold `undoJobId` (string | null) and `showUndo` (boolean).
    * In the `onSuccess` callback of `useSendEmailMutation`:
        1. Call `composer.store.close()`.
        2. Set `ui.store.showUndo = true` and `ui.store.undoJobId = response.job_id`.
        3. Start a `setTimeout` for `undo_send_delay_seconds`. When it fires, set `showUndo = false`.
    * Create `components/UndoSnackbar.tsx` that renders if `showUndo` is true.
    * Create `useUndoSendMutation` that `DELETE`s `/api/v1/send/undo/:jobID`.
    * The "Undo" button in the snackbar calls this mutation and hides the snackbar.
* [ ] **Frontend Integration Tests (RTL + <code>msw</code>):**
    * Mock `POST /api/v1/send` to return `{"job_id": "123"}`.
    * Click "Compose" button. **Assert** composer is visible.
    * Fill inputs, click "Send". **Assert** the API was called with the correct data.
    * **Assert** the composer closes.
    * **Assert** the "Undo Snackbar" is now visible.
    * Mock `DELETE /api/v1/send/undo/123`.
    * Click the "Undo" button. **Assert** the `DELETE` API was called and the snackbar hides.

### **4/4. ↩️ Frontend: "Reply/Forward" logic**

This part just pre-fills the composer you just built.

* [ ] **Add "Reply/Forward" buttons:**
    * In `Message.tsx` (inside `ThreadView.tsx`), add "Reply," "Reply All," and "Forward" buttons.
* [ ] **Implement store actions:**
    * **<code>composer.store.openReply(msg)</code>:**
        * Sets `isOpen = true`.
        * Sets `to = [msg.from_address]`.
        * Sets `subject = "Re: " + msg.subject`.
        * Sets `body = "On [date], [sender] wrote:\n\n> [quoted body]"`.
        * Sets `inReplyTo = msg`.
    * **<code>composer.store.openReplyAll(msg)</code>:**
        * Same, but `to = [msg.from_address, ...msg.to_addresses, ...msg.cc_addresses]` (filtering out the user's own email).
    * **<code>composer.store.openForward(msg)</code>:**
        * Same, but `to = []`, `subject = "Fwd: " + msg.subject`, and `body = "Forwarded message:\n\n..."`.
* [ ] **Frontend Integration Tests (RTL):**
    * Render `ThreadView.tsx` with a mock message.
    * Click "Reply". **Assert** the `Composer` component opens.
    * **Assert** the "To" and "Subject" fields are pre-filled correctly with "Re: ...".

### **4/5. 🧪 Test plan: Milestone 4 (end-to-end)**

This **Playwright** plan tests the full send-and-undo loop.



* [ ] **Test 1: Compose and Send (Full Loop)**
    * Log in.
    * Click "Compose."
    * Fill in "To" (with an external email you can check), "Subject" ("E2E Test Send"), and "Body".
    * Click "Send".
    * **Assert** the composer closes.
    * **Assert** the "Undo" snackbar appears.
    * **Do not** click Undo. Wait for the `undo_send_delay_seconds` (e.g., 20s) *plus* the worker poll time (e.g., 5s).
    * **Check external email:** **Assert** the email ("E2E Test Send") was received.
    * **Check V-Mail UI:** Click the "Sent" folder. **Assert** the "E2E Test Send" email now appears in the "Sent" list (proves the IMAP `APPEND` worked).
* [ ] **Test 2: Compose and Undo (Full Loop)**
    * Log in.
    * Click "Compose."
    * Fill in "To" (with an external email), "Subject" ("E2E Test Undo"), and "Body".
    * Click "Send".
    * **Assert** the "Undo" snackbar appears.
    * **Immediately click "Undo"**.
    * **Assert** the snackbar disappears.
    * Wait 30 seconds (longer than your undo delay).
    * **Check external email:** **Assert** the email ("E2E Test Undo") was **NOT** received.
    * **Check V-Mail UI:** Click the "Sent" folder. **Assert** the email does **NOT** appear in the "Sent" list.

## Milestone 5: Quality of life

- Goal: Polish the MVP.
- Tasks: Auto-save drafts. Add keyboard shortcuts. Add pagination. Add IDLE and WebSocket connection.

Breakdown:

### **5/1. 💾 Feature: Auto-save drafts**

This plan implements a fast, reliable auto-save that saves to your Postgres DB first,
and then (as a bonus) syncs to your IMAP server in the background.


#### **Backend**

* [ ] **Update <code>drafts</code> table:**
    * Add a new column: `imap_uid BIGINT NOT NULL DEFAULT 0`. This will store the `UID` of the draft on the IMAP server once it's synced.
* [ ] **Create <code>POST /api/v1/drafts</code> endpoint:**
    * This will be your auto-save endpoint. It needs to be fast.
    * Create a `SaveDraft(ctx, draft)` function in `/backend/internal/db/drafts.go`.
    * This function should use `INSERT ... ON CONFLICT (id) DO UPDATE ...` to create *or* update the draft in your Postgres `drafts` table.
    * The `actions_handler` for this route should:
        1. Read the draft payload (to, subject, body, etc.) from the JSON body.
        2. Call `db.SaveDraft(...)`.
        3. Return the `draft.id` to the frontend: `{"draft_id": "..."}`.
* [ ] **(Optional/Bonus) Create background IMAP sync:**
    * After saving to the DB, have the `POST /api/v1/drafts` handler queue a *new* job: `db.QueueAction(..., "sync_draft", payload{"draft_id": "..."})`
    * In your `/backend/internal/sync/worker.go`, add a `case "sync_draft":`
    * In `/backend/internal/imap/actions.go`, create `func (s *Service) SyncDraft(ctx, draftID string) error`.
    * This function should:
        1. Fetch the draft from your Postgres `drafts` table.
        2. Get the `imap_uid` and `drafts_folder_name` (from `user_settings`).
        3. Get an IMAP connection.
        4. If `imap_uid == 0`: `APPEND` the draft to the `drafts_folder_name` and save the new `imap_uid` (from the `APPEND` response) back to your Postgres `drafts` table.
        5. (Harder/v2) If `imap_uid > 0`: `DELETE` the old message (`STORE +FLAGS.SILENT \Deleted`) and `APPEND` the new one (and update the `imap_uid`). For now, just `APPEND`ing is fine.
* [ ] **Create <code>GET /api/v1/drafts</code> endpoint:**
    * This route should query your Postgres `drafts` table (not IMAP) and return all saved drafts for the user.
* [ ] **Update <code>send_email</code> worker:**
    * The payload for `send_email` now needs an *optional* `draft_id_to_delete`.
    * After the `send_email` job successfully sends (SMTP) and appends to "Sent" (IMAP), it must **also**:
        1. `DELETE FROM drafts WHERE id = $1` (the Postgres draft).
        2. (If you did the bonus sync) Queue a new job to delete the draft from the IMAP `Drafts` folder.

#### **Frontend**

* [ ] **Create <code>useAutoSave</code> hook:**
    * Create `hooks/useAutoSave.ts`.
    * This hook takes the current composer state (to, subject, body) as an argument.
    * It uses `useEffect` and `setTimeout` to create a "debounce" (e.g., trigger 2 seconds *after* the user stops typing).
    * When the timeout fires, it calls `saveDraftMutation.mutate(...)`.
* [ ] **Create <code>useSaveDraftMutation</code>:**
    * In `api/email.api.ts`, create this mutation. It `POST`s to `/api/v1/drafts`.
    * In its `onSuccess` callback, it must save the returned `draft_id` into your `composer.store` (which needs a new `draft_id` field).
* [ ] **Integrate with <code>Composer</code>:**
    * Call `useAutoSave(composerState)` from your `Composer.tsx` component.
* [ ] **Update "Send" button:**
    * When the "Send" button is clicked, the `useSendEmailMutation` must now include the `draft_id` from the `composer.store` in its payload, so the backend knows which draft to delete.
* [ ] **Create "Drafts" page:**
    * Add a "Drafts" link to your `Sidebar.tsx` (it should point to `/drafts`).
    * Create `pages/Drafts.page.tsx`.
    * This page uses `useQuery` to `GET /api/v1/drafts`.
    * It renders a list of drafts (you can reuse `EmailListItem.tsx`).
    * When a draft is clicked, it calls `composer.store.openDraft(draftData)` (a new action you'll create) to open the composer and pre-fill it.

#### **Testing**

* [ ] **Backend Unit:** Test the `POST /api/v1/drafts` handler (asserts DB `INSERT`/`UPDATE` and returns an `id`). Test the `GET /api/v1/drafts` handler.
* [ ] **Frontend Unit:** Test `useAutoSave` hook. Mock `setTimeout` and the mutation. Assert `mutate` is (or is not) called based on typing/pausing.
* [ ] **Frontend Integration:** Mock `POST /api/v1/drafts`. Type in the composer, pause for 3 seconds. **Assert** the `POST` API was called.
* [ ] **E2E:**
    1. Click "Compose," type "test subject."
    2. Reload the page.
    3. Go to the "Drafts" page. **Assert** "test subject" is in the list.
    4. Click it. **Assert** the composer opens with "test subject".
    5. Add a recipient and click "Send."
    6. Wait 30 seconds. **Assert** the draft is now gone from the "Drafts" page.

### **5/2. ⌨️ Feature: Add keyboard shortcuts**

This is a frontend-only task that expands on the hook you built in M2.

* [ ] **Expand <code>useKeyboardShortcuts.ts</code>:**
    * Add logic to listen for key presses.
    * `c`: Call `composer.store.openCompose()`.
    * `/`: Find the search bar `ref` and call `ref.current.focus()`.
    * `g` then `i`: `Maps('/?folder=INBOX')` (this needs a simple state machine in the hook).
    * `g` then `s`: `Maps('/?folder=Sent')` (or your starred view).
    * `g` then `d`: `Maps('/drafts')`.
* [ ] **Add shortcuts for selected items:**
    * This requires a "selected item" state (e.g., `ui.store.selectedThreadId`). Your `j`/`k` keys should update this ID.
    * `e`: If `selectedThreadId` exists, call `moveThreadMutation.mutate({ ..., destination_folder: 'Archive' })`.
    * `s`: If `selectedThreadId` exists, call `starThreadMutation.mutate(...)`.
    * `#` (Shift+3): If `selectedThreadId` exists, call `moveThreadMutation.mutate({ ..., destination_folder: 'Trash' })`.
* [ ] **Add shortcuts for thread view:**
    * When on a `/thread/:threadId` page:
    * `r`: Call `composer.store.openReply(currentThreadData)`.
    * `a`: Call `composer.store.openReplyAll(currentThreadData)`.
    * `f`: Call `composer.store.openForward(currentThreadData)`.
* [ ] **Add "Undo" shortcut:**
    * `z`: If `ui.store.showUndo` is true, call `undoSendMutation.mutate()`.

#### **Testing**

* [ ] **Frontend Unit:**
    * Write extensive tests for `useKeyboardShortcuts.ts`.
    * Simulate `keydown` events (`c`, `/`, `g`+`i`, `e`, `r`, etc.).
    * Mock all the store actions and mutations.
    * **Assert** that the *correct* mock function is called for each key press.

### **5/3. 📄 Feature: Add pagination**

This makes the app usable with large inboxes.

#### **Backend**

* [ ] **Update <code>GET /api/v1/threads</code> handler:**
    * It *must* read `page` and `limit` query params (e.g., `?page=2&limit=50`). Default to `page=1, limit=100`.
    * Update your DB query for threads to use `LIMIT $1 OFFSET $2`.
    * Run a *second* DB query: `SELECT COUNT(*) FROM ...` with the same `WHERE` clause (to get the total count).
    * Change the API response to a new object: `{"threads": [...], "pagination": {"total_count": 1234, "page": 2, "per_page": 50}}`
* [ ] **Update <code>GET /api/v1/search</code> handler:**
    * Apply the exact same `page` and `limit` logic.
    * Return the same `{"threads": [...], "pagination": {...}}` object.

#### **Frontend**

* [ ] **Create <code>EmailListPagination.tsx</code> component:**
    * This component receives the `pagination` object as a prop.
    * It calculates `totalPages = total_count / per_page`.
    * It renders "Page [page] of [totalPages]" and "Next >" / "&lt; Prev" links.
* [ ] **Update <code>Inbox.page.tsx</code> and <code>Search.page.tsx</code>:**
    * Read the `page` from `useSearchParams`.
    * Pass the `page` to the `useQuery` hook to fetch the correct data.
    * Get the `pagination` object from the API response.
    * Render `&lt;EmailListPagination pagination={data.pagination} />` at the bottom.
    * The "Next" link should use `Maps` to go to `/?folder=INBOX&page={page + 1}`.
    * The "Prev" link should use `Maps` to go to `/?folder=INBOX&page={page - 1}`.

#### **Testing**

* [ ] **Backend Unit:** Test the `GET /api/v1/threads` handler. Assert `?page=2&limit=50` results in `LIMIT 50 OFFSET 50` in the SQL query. Assert the `pagination` object in the JSON response is correct.
* [ ] **Frontend Integration:**
    * Mock the API to return `{"threads": [...], "pagination": {"total_count": 300, "page": 1, "per_page": 100}}`.
    * **Assert** the pagination component renders "Page 1 of 3".
    * Mock `Maps`. Click the "Next" button.
    * **Assert** `Maps` was called with the new URL (`?page=2`).

### **5/4. ⚡ Feature: Add real-time updates**

This is the most complex but most rewarding "quality of life" feature.

#### **Backend**

* [ ] **Add WebSocket library:**
    * `go get github.com/gorilla/websocket`
* [ ] **Create WebSocket Hub:**
    * Create `/backend/internal/websocket/hub.go`.
    * The `Hub` struct will manage active connections: `clients map[string]*websocket.Conn` (mapping `userID` to their connection).
    * It needs methods: `Register(userID, conn)`, `Unregister(userID)`, and `Send(userID, message []byte)`.
* [ ] **Create <code>GET /api/v1/ws</code> endpoint:**
    * Add this route in `routes.go`.
    * The handler (`wsHandler`) upgrades the HTTP connection to a WebSocket.
    * It gets the `userID` from the auth context.
    * It calls `hub.Register(userID, conn)`.
    * It must handle disconnects by calling `hub.Unregister(userID)`.
* [ ] **Create IMAP IDLE listener:**
    * In `/backend/internal/imap/idle.go`, create `func (s *Service) StartIdleListener(ctx context.Context, userID string, hub *websocket.Hub)`.
    * **Launch it:** When a user successfully connects to the WebSocket (`hub.Register`), launch this function in a **new goroutine** for that `userID`.
    * **Logic:**
        1. Get a *dedicated* IMAP connection (do not use the pool).
        2. Run `SELECT INBOX`.
        3. Start a `for` loop (to handle disconnects).
        4. Inside the loop, run `client.Idle()`.
        5. Listen for updates. When an update arrives (e.g., `* 1 EXISTS`), call `hub.Send(userID, []byte('{"type": "new_email", "folder": "INBOX"}'))`.
        6. If `client.Idle()` returns an error (e.g., timeout), `log.Println` and `time.Sleep(10 * time.Second)` before the loop retries.

#### **Frontend**

* [ ] **Create <code>useWebSocket</code> hook:**
    * Create `hooks/useWebSocket.ts`.
    * It should be called *once* from your main `Layout.tsx`.
    * `useEffect` on mount:
        1. `const socket = new WebSocket('ws://localhost:8080/api/v1/ws')` (use wss in prod).
        2. `socket.onmessage = (event) => { ... }`
        3. `socket.onclose = () => { ... }`
    * The `onmessage` handler parses the `event.data`.
    * `if (message.type === 'new_email') { ... }`
* [ ] **Invalidate cache on message:**
    * Inside the `onmessage` handler:
    * Get the `queryClient` using `useQueryClient()`.
    * Call `queryClient.invalidateQueries({ queryKey: ['threads', message.folder] })`.
    * This will automatically make `TanStack Query` refetch the thread list, and the new email will appear.

#### **Testing**

* [ ] **Frontend Integration (RTL + Mock WebSocket):**
    * You'll need a library like `mock-socket`.
    * Render the `Inbox.page.tsx` (which is inside `Layout.tsx`, so the hook runs).
    * Simulate a message from the mock socket: `mockSocket.send('{"type": "new_email", "folder": "INBOX"}')`.
    * **Assert** that `queryClient.invalidateQueries` was called with `['threads', 'INBOX']`.
* [ ] **E2E:**
    * This is the only true test.
    * Log in to V-Mail. Have the Inbox page open.
    * Use a *different* email client (or your `spike` script!) to send a new email to your test account.
    * **Assert** the new email appears in the V-Mail inbox *without* a page reload.

## Milestone 6: Offline

- Goal: Basic offline support.
- Tasks: Implement IndexedDB caching for recently viewed emails. Build the sync logic.

Breakdown:

### **6/1. 💾 Frontend: Set up local database**

First, we need a place to store the emails in the browser. We'll use `dexie.js`, which is a powerful and easy-to-use wrapper for IndexedDB.

* [ ] **Install Dexie: `pnpm install dexie dexie-react-hooks`
* [ ] **Define the local DB schema:**
    * Create a new file: `/frontend/src/lib/db.ts`.
    * In this file, define your Dexie database. The schema should *mirror* your Postgres tables, as this will make syncing much easier.
```typescript
import Dexie, { Table } from 'dexie'

// Define interfaces for your tables
// (You can move these to a /types file)
export interface IThread {
  id: string // This is your stable_thread_id
  subject?: string
  // ... other thread properties
}

export interface IMessage {
  id: string // This is your message_id_header
  thread_id: string
  imap_folder_name: string
  from_address?: string
  subject?: string
  unsafe_body_html?: string
  body_text?: string
  is_read: boolean
  is_starred: boolean
  // ... other message properties
}

export class VMailDB extends Dexie {
  threads!: Table&lt;IThread>
  messages!: Table&lt;IMessage>

  constructor() {
    super('vmailDB')
    this.version(1).stores({
      // 'id' is the primary key
      // 'thread_id' and 'imap_folder_name' are indexes
      threads: 'id',
      messages: 'id, thread_id, imap_folder_name',
    })
  }
}

export const db = new VMailDB()
```

### **6/2. 📥 Frontend: Cache viewed emails**

This part implements "caching for recently viewed emails." The logic is simple: anything you successfully fetch from the API, you save a copy of in IndexedDB.

* [ ] **Cache thread list:**
    * In your `Inbox.page.tsx` (or wherever you fetch threads), find your `useQuery` for `GET /api/v1/threads`.
    * Add an `onSuccess` callback to the `useQuery` options.
    * In `onSuccess(data)`, call `db.threads.bulkPut(data.threads)`. This will "upsert" (insert or update) all the threads you just fetched.
* [ ] **Cache full thread:**
    * In your `Thread.page.tsx`, find your `useQuery` for `GET /api/v1/thread/:threadId`.
    * Add an `onSuccess` callback.
    * In `onSuccess(data)`, save *both* the thread *and* its messages: \
      TypeScript \
      db.threads.put(data.thread)
    * db.messages.bulkPut(data.thread.messages)

### **6/3. 🔌 Frontend: Read from cache when offline**

Now, we'll change your queries to *always* read from the local cache first. This gives an "offline-first" feel.

* [ ] **Modify <code>GET /api/v1/threads</code> query:**
    * In your `useQuery` for `GET /api/v1/threads`, change the `queryFn`.
    * The `queryFn` should *first* try to get data from Dexie:
```typescript
queryFn: async () => {
  // 1. Try to get data from the local cache
  const cachedThreads = await db.threads
    .where('imap_folder_name') // Assuming you add this to the threads table
    .equals(folder)
    .toArray()

  // 2. If online, fetch from API in the background
  if (navigator.onLine) {
    try {
      const freshData = await api.getThreads(folder) 
      // The onSuccess (from 6/2) will auto-cache this
      return freshData.threads
    } catch (error) {
      // If API fails, return cached data so app still works
      return cachedThreads
    }
  }

  // 3. If offline, just return cached data
  return cachedThreads
}
```
* [ ] **Modify <code>GET /api/v1/thread/:threadId</code> query:**
    * Apply the same logic. The `queryFn` should first `await db.messages.where('thread_id').equals(threadId).toArray()` and return that if the user is offline.

### **6/4. 🔄 Backend: Create "delta" sync endpoints**

Your `action_queue` already handles syncing *actions* (writes). This is for syncing *reads* (changes from other clients, like your phone).

To do this efficiently, we need a "give me what's changed" endpoint.

* [ ] **Create a new migration:**
    * `migrate create -ext sql -dir backend/migrations -seq add_timestamps`
* [ ] **Modify schema (<code>.up.sql</code>):**
    * We need `updated_at` on our main tables.
```postgresql
-- Create a trigger function to auto-update timestamps
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add the column and trigger to 'threads'
ALTER TABLE "threads" ADD COLUMN "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON "threads"
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

-- Add the column and trigger to 'messages'
ALTER TABLE "messages" ADD COLUMN "updated_at" TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON "messages"
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
```

* [ ] **Create new "sync" API endpoints:**
    * `GET /api/v1/sync/threads?since=&lt;timestamp>`
    * `GET /api/v1/sync/messages?since=&lt;timestamp>`
    * These handlers read the `since` query param.
    * They query Postgres: `SELECT * FROM threads WHERE user_id = $1 AND updated_at > $2`.
    * They return a list of updated/new threads and messages.

### **6/5. ⚡ Frontend: Build the sync logic**

This logic will run in the background to keep the local database fresh.

* [ ] **Create <code>useSync</code> hook:**
    * Create `hooks/useSync.ts`.
    * This hook will be called *once* from your main `Layout.tsx`.
* [ ] **Implement the <code>sync</code> function:**
    * Inside the hook, create a `sync()` function.
    * It should:
        1. Get `lastSyncTimestamp` from `localStorage`.
        2. `fetch('/api/v1/sync/threads?since=' + lastSyncTimestamp)`.
        3. `fetch('/api/v1/sync/messages?since=' + lastSyncTimestamp)`.
        4. Take the results and `bulkPut` them into your Dexie `db.threads` and `db.messages` tables.
        5. Save `new Date().toISOString()` into `localStorage` as the new `lastSyncTimestamp`.
* [ ] **Trigger the sync:**
    * Use `useEffect` to call `sync()` once on app load.
    * Use `setInterval` to call `sync()` every 5 minutes.
    * Add a `window.addEventListener('online', sync)` to trigger an immediate sync when the user's connection returns.

### **6/6. 🧪 Test plan: Milestone 6 (end-to-end)**

Offline mode is notoriously difficult to test. Use **Playwright** for this.

* [ ] **Test 1: Offline read (cache population)**
    * Log in while **online**.
    * Open the "Inbox".
    * Click the first thread (Subject: "Meeting Notes").
    * Click back to the "Inbox".
    * **Turn network offline** using Playwright's `context.setOffline(true)`.
    * **Reload the page**.
    * **Assert** the "Inbox" list *still* loads (from Dexie).
    * **Assert** you can click "Meeting Notes" and *read the full email* (from Dexie).
    * **Assert** that clicking the *second* thread (which you never opened) shows a loading spinner or an "Offline" message (because its body isn't cached).
* [ ] **Test 2: Offline action (action_queue)**
    * Log in while **online**.
    * Open the "Inbox".
    * **Turn network offline**.
    * Find a thread ("Meeting Notes") and click the "Star" button.
    * **Assert** the star *optimistically* turns "filled".
    * **Reload the page** (still offline).
    * **Assert** the star is *still* "filled" (this tests that your optimistic UI state is also saved, or that you're reading the `action_queue`).
    * **Turn network online** (`context.setOffline(false)`).
    * Wait 10-15 seconds (for the worker and sync logic to run).
    * **Reload the page** (now online).
    * **Assert** the "Meeting Notes" email is *still* starred (proving the action was synced to the server).
* [ ] **Test 3: Background sync (delta sync)**
    * Log in to V-Mail in Playwright. Have the "Inbox" open.
    * Use your `spike` script (or another email client) to send a **new email** to your account.
    * **Do not** reload the Playwright browser.
    * Wait for your sync interval (or manually trigger sync via a debug button).
    * **Assert** the new email *appears* at the top of the list in V-Mail *without* a page reload.

## Later

* [ ] Write a doc for how to create a daily DB backup, e.g., via a `pg_dump` cron job.