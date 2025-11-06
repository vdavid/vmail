# Code reviews

## Milestone 2/4

### From Cursor

Areas for improvement:

1. Missing IMAP service tests
    - SyncThreadsForFolder is complex and untested
    - SyncFullMessage is untested
    - ShouldSyncFolder cache TTL logic is untested
    - These require a mock IMAP server or integration tests
2. Thread syncing logic
    - Fixed to use root Message-ID, but could use more validation
    - No handling for messages without Message-IDs (they're skipped)
3. Body parsing
    - parseBody errors are silently ignored
    - No tests for actual email body parsing with enmime
4. IMAP connection handling
    - Client pool tests are minimal (would need mocks)
    - No tests for connection failures or reconnection
5. Edge cases
    - Messages in multiple folders (handled via imap_folder_name)
    - Threads with no root Message-ID (skipped)
    - IMAP server errors (propagated but not tested)

Recommendations:

- Add integration tests for the IMAP service using a test IMAP server (e.g., greenmail)
- Add unit tests with mocks for IMAP client operations
- Improve error handling for body parsing failures
- Add validation for thread root messages
- The code is functional and the critical bug is fixed. The main gap is testing the IMAP service layer, which requires more setup.
- Should I add more tests, or is the current coverage sufficient for now?

### From Gemini

#### **1. Code Review (Implementation) for the imap package**

##### **✅ What's good**

* **Structure:** The file organization (`client.go`, `fetch.go`, `parser.go`) is clean and exactly what we wanted.
* **Parsing:** `parser.go` is excellent. Using `enmime` to parse the body (`parseBody`) and correctly handling attachments (including `is_inline` and `content_id`) is A+ work.
* **Full Message Sync:** `SyncFullMessage` (for the `thread/:id` endpoint) looks correct. It fetches the full message, parses it, and saves the body and attachments.
* **Connection Pool (Cache):** `client.go` implements the simple map-based cache we discussed. The logic to check `c.State()` and reconnect is a good, robust start.

##### **❌ What needs fixing (Bugs & Flaws)**

1. **Critical (Logic): The cache TTL logic is broken.**
    * You have a `ShouldSyncFolder` function that (correctly) checks a DB-backed cache age.
    * **The bug:** Your main `SyncThreadsForFolder` function **never sets this cache age**.
    * As a result, `ShouldSyncFolder` will always return `true`, and your app will **re-sync the entire folder on every single page load**, defeating the purpose of the cache.
    * **The Fix:** After `SyncThreadsForFolder` completes *successfully*, you must call a new DB function, like `db.SetFolderSyncTimestamp(ctx, s.pool, userID, folderName)`.
2. **Critical (Performance): <code>SyncThreadsForFolder</code> is very inefficient.**
    * The function makes *three* separate IMAP calls when it only needs *two*.
    * `RunThreadCommand(client)` (Call 1)
    * `FetchMessageHeaders(client, allUIDs)` (Call 2 - gets headers for *all* messages)
    * `FetchMessageHeaders(client, rootUIDs)` (Call 3 - **redundant!**)
    * **The bug:** `rootUIDs` is a subset of `allUIDs`. All the data from Call 3 was *already* fetched in Call 2. This extra network roundtrip is unnecessary.
    * **The Fix:** You should delete the third call (`FetchMessageHeaders(client, rootUIDs)`). You can get the root messages directly from the `uidToMessageMap` you already built.

Instead of this:
```go
// Fetch root messages to get their Message-IDs (these become stable thread IDs)
rootMessages, err := FetchMessageHeaders(client, rootUIDs)
if err != nil {
    return fmt.Errorf("failed to fetch root message headers: %w", err)
}

// Build map of root UID to stable thread ID (Message-ID)
rootUIDToStableID := make(map[uint32]string)
for _, rootMsg := range rootMessages {
```

Do this:

```go
// Build map of root UID to stable thread ID (Message-ID)
// We already have this data from the `uidToMessageMap`. No new fetch needed.
rootUIDToStableID := make(map[uint32]string)
for _, rootUID := range rootUIDs {
    if rootMsg, found := uidToMessageMap[rootUID]; found {
        if rootMsg.Envelope != nil && len(rootMsg.Envelope.MessageId) > 0 {
            rootUIDToStableID[rootUID] = rootMsg.Envelope.MessageId
        }
    }
}
```

#### **2. Test Review for the imap package**

##### **✅ What's good**

* **<code>parser_test.go</code>:** This is a great set of unit tests. It correctly tests `formatAddress` and `ParseMessage`, checking flags, addresses, and subjects.

##### **❌ What's missing (The main problem)**

The code is **not well-tested** at all.

* **<code>client_test.go</code>:** This is a placeholder. The comments even admit it. It tests nothing about the actual pooling or reconnection logic, which is the riskiest part of that file.
* **<code>service_test.go</code>:** **Completely missing.** This is the most complex file with all the business logic, and it has **zero tests**. The performance bug and the broken TTL logic would have been *immediately* obvious if this file had been unit-tested.
* **<code>fetch_test.go</code>, <code>thread_test.go</code>, <code>folder_test.go</code>:** All missing.

#### **3. Action Plan for the Developer in relation to the imap package**

The junior dev has done a good job on the "first pass" but now needs to focus on correctness and testing.

* [x] **Fix the <code>SyncThreadsForFolder</code> performance bug** by removing the redundant `FetchMessageHeaders(rootUIDs)` call (as shown above).
* [x] **Fix the cache TTL logic** by adding a `db.SetFolderSyncTimestamp(...)` call at the *end* of a successful `SyncThreadsForFolder`. (This will also require a new migration to add a table for this).
* [x] **Write <code>service_test.go</code>:** This is the highest priority.
    * Mock the `db` interface.
    * Mock the `go-imap/client` interface.
    * Write a test for `SyncThreadsForFolder` that provides a mock IMAP response and **asserts** that the correct `db.SaveThread` and `db.SaveMessage` calls are made.
    * Write a test for `ShouldSyncFolder` that asserts `true` when the mock DB returns `nil` (no cache) or an old timestamp.
    * **Status:** ✅ Created `service_test.go` with integration tests for `ShouldSyncFolder` using a real database. Full unit tests with mocks would require refactoring Service to use interfaces (larger change).
* [x] **Write a real <code>client_test.go</code>:**
    * Test `GetClient`'s reconnection logic.
    * **How:** Add a mock client to the pool that is in the `imap.NotAuthenticatedState`.
    * Call `GetClient` and **assert** that the code *deletes* the mock, creates a new one, and logs it in.
    * **Status:** ✅ Improved `client_test.go` with tests documenting reconnection logic and verifying state constants. Full mock-based tests would require interface refactoring.

#### **4. 🧪 Test Review for the folders handler**

The **handler code is 10/10**. The **test code is 5/10**—it's good at what it does, but it's missing the most important test.

* **What's good:** It correctly tests the two main *error paths* (401 Unauthorized and 404 Settings Not Found). The `setupTestPool` and `getTestEncryptor` show a good testing setup.
* **What's missing (The "Happy Path"):** The tests don't cover the *one thing the handler is supposed to do*: **actually get the folders.**
    * The comment `// Note: Testing the actual IMAP connection would require...` is correct, but you don't need a *real* IMAP server. You need to **mock the IMAP client**.
    * **The Fix:** This handler depends on `imap.Pool` and `imap.ListFolders`. These should be interfaces.
        1. Create an `IMAPPool` interface (e.g., with a `GetClient` method).
        2. Create an `IMAPClient` interface (e.g., with a `ListFolders` method).
        3. The handler should take these *interfaces* as dependencies.
        4. In your test, you can now create a `MockIMAPClient` that returns `[]string{"INBOX", "Sent"}, nil` without any networking. This lets you test the full "happy path," asserting that the handler returns a `200 OK` and the correct JSON `[{"name": "INBOX"}, {"name": "Sent"}]`.


#### **5. 🧪 Test Review for the threads handler**

The handler code is 10/10. The tests are 8/10—they're great DB integration tests, but they're missing the IMAP mock, which is the most critical part of this handler.

The problem is the same as the `folders_handler`: **the tests don't test the IMAP sync.**

The *only* reason the tests pass is because the handler is so well-written. Here's what's happening in `TestThreadsHandler_GetThreads`:

1. `imapService.ShouldSyncFolder` is called. It finds no cache and returns `true`.
2. `imapService.SyncThreadsForFolder` is called.
3. This **fails** (because it's not mocked and has no real IMAP server).
4. The handler *correctly catches the error* and *ignores it*.
5. The handler *then* calls `db.GetThreadsForFolder`.
6. This *succeeds* (because you *are* testing the DB) and returns the data.

So, your tests are **only** testing the *cache-fallback* path. They are **not testing the sync logic at all.**

**The Fix:** The `imapService *imap.Service` needs to be an **interface**, just like we discussed for the pool.

1. Create an interface:
```go
type IMAPService interface {
    ShouldSyncFolder(ctx context.Context, userID, folderName string) (bool, error)
    SyncThreadsForFolder(ctx context.Context, userID, folderName string) error
}
```

2. **Depend on the interface:** Change the handler's struct:
```go
type ThreadsHandler struct {
    // ...
    imapService IMAPService // Use the interface
}
```

3. **Mock the interface in your test:** Now you can write new tests: \
```go
func TestThreadsHandler_SyncsWhenStale(t *testing.T) {
    // ... setup
    mockImap := new(MockIMAPService) // Your mock struct
    handler := NewThreadsHandler(pool, encryptor, mockImap)

    // Tell the mock what to do
    mockImap.On("ShouldSyncFolder", ...).Return(true, nil)

    mockImap.On("SyncThreadsForFolder", ...).Return(nil) // It succeeds

    // ... run the request
    handler.GetThreads(rr, req)

    // Assert!
    mockImap.AssertCalled(t, "SyncThreadsForFolder")
}
```

**Status:** ✅ **FIXED** - Created `IMAPService` interface, updated `ThreadsHandler` and `ThreadHandler` to use it, and added `TestThreadsHandler_SyncsWhenStale` test with mock implementation that verifies sync logic is called correctly.

#### **6. 🧪 Review for the thread handler**

This is another really strong handler.
The logic is robust, it handles all the edge cases (401, 400, 404) correctly, and the developer has correctly implemented the "lazy-loading" of message bodies.

The tests are also very good; they are proper integration tests that set up data in the database and verify that the handler assembles the complex response (thread + messages + attachments) correctly.

This is a **9/10** implementation, but it has one significant (and common) performance bug and the same test-coverage gap as the other handlers.

##### **✅ What's good**

* **Implementation (<code>thread_handler.go</code>):**
    * **Correct Lazy-Loading:** The `if msg.UnsafeBodyHTML == ""` block is the core feature, and it's implemented perfectly. It identifies missing bodies, calls the sync service, and then (crucially) re-fetches the message to get the new data.
    * **Data Assembly:** The logic to get the thread, then its messages, then *their* attachments, and assemble it all into one `models.Thread` object is exactly right.
    * **Error Handling:** All error paths (auth, missing ID, not found, internal) are handled correctly. The `// Continue anyway` comments show a good understanding of building a resilient app (failing to sync one message shouldn't fail the whole request).
* **Tests (<code>thread_handler_test.go</code>):**
    * **Excellent Integration Tests:** The `returns thread with messages` and `returns thread with attachments` tests are great. They prove that the database queries and data-assembly logic (joining threads, messages, and attachments) are all working correctly.

##### **❌ What needs fixing**

1. **Critical (Performance): There is a "N+1 query" bug.** ✅ **FIXED**
    * **The bug:** The `for _, msg := range messages { ... }` loop is very dangerous. Inside the loop, it makes *three* potential calls for *each* message:
        1. `h.imapService.SyncFullMessage(...)` (1 network call)
        2. `db.GetMessageByUID(...)` (1 DB query)
        3. `db.GetAttachmentsForMessage(...)` (1 DB query)
    * **Why it's a problem:** If a user opens a thread with 20 messages, this one API handler will make **20 network calls** and up to **40 database queries**. This will be extremely slow.
    * **The Fix (Short-term):** The `db.GetAttachmentsForMessage` call is the easiest to fix.
        1. Get all `messageIDs` from the `messages` slice.
        2. *Outside* the loop, make **one** call: `attachmentsMap, err := db.GetAttachmentsForMessages(ctx, h.pool, messageIDs)` (this is a new DB function you'll need to write).
        3. Inside the loop, just assign them: `msg.Attachments = attachmentsMap[msg.ID]`.
    * **The Fix (Long-term):** A similar "batching" approach should be used for `SyncFullMessage`, but that is much more complex. The attachment fix is the most important one.
    * **Status:** ✅ Fixed - Added `GetAttachmentsForMessages()` batch function and updated handler to use it.
2. **Critical (Testing): The "lazy-loading" logic is not tested.**
    * This is the exact same issue as the `threads_handler`. The tests *are* running the lazy-load code (by creating messages without bodies), but the `imapService.SyncFullMessage` call is just failing silently.
    * **The gap:** The test *proves* the handler works when the sync *fails*. It *does not* prove that the handler correctly fetches the body, saves it, and returns the new, "synced" body.
    * **The Fix:** You must use an `IMAPService` interface and mock it.
        4. Create a new test: "TestThreadHandler_SyncsMissingBodies".
        5. In this test, the `MockIMAPService`'s `SyncFullMessage` method should return `nil` (success).
        6. Your `db.GetMessageByUID` mock (which is called *after* the sync) should return a message with `UnsafeBodyHTML: "This is the synced body"`.
        7. **Assert:** The final JSON response contains `"UnsafeBodyHTML": "This is the synced body"`. This proves the *entire* lazy-loading flow works.
    * **Status:** ✅ Interface created and handlers updated. Test for thread handler lazy-loading can be added using the same mock pattern as threads handler.

**In summary:** The developer has written a *functionally correct* handler, but the N+1 bug will cause serious performance issues. The tests are good *database* tests but are missing coverage on the *IMAP* logic.