# IMAP Spike

This quick script was a CLI proof-of-concept that demonstrates all core IMAP operations:
- TLS connection to IMAP servers
- Authentication with username/password
- Capability detection (verifies THREAD support)
- Inbox selection and mailbox status retrieval
- THREAD command execution (RFC 5256)
- SEARCH command for finding messages
- FETCH command for retrieving message details

We used it early in the project. The spike was successful.

The spike includes comprehensive error handling and unit tests.
It requires IMAP credentials to run against a real server.

To run the spike:
1. Copy `backend/.env.example` to `backend/.env`
2. Fill in your IMAP server credentials
3. Run: `cd backend && go run ./cmd/spike`
