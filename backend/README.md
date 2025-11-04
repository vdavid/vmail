# V-Mail Backend - Milestone 1: IMAP Spike

This CLI application demonstrates the core IMAP functionality required for V-Mail.

## Goal

Prove that the core technology works by connecting to an IMAP server and executing basic commands.

## Features

This spike implements the following IMAP operations:

1. **Connect** to an IMAP server using TLS (`imap.DialTLS`)
2. **Login** with username and password (from environment variables)
3. **Check capabilities** to verify THREAD support
4. **Select** the Inbox folder
5. **Run THREAD** command (`THREAD=REFERENCES UTF-8 ALL`)
6. **Run SEARCH** command (searches for unseen messages)
7. **Fetch** a message and display its headers and body structure

## Running the spike

### Prerequisites

- Go 1.24 or later
- Access to an IMAP server (tested with mailcow/Dovecot)
- IMAP credentials

### Setup

1. Copy the environment example file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and add your IMAP credentials:
   ```bash
   IMAP_SERVER=imap.example.com:993
   IMAP_USER=your_email@example.com
   IMAP_PASSWORD=your_password
   ```

3. Load the environment variables:
   ```bash
   source .env
   ```
   
   Or export them directly:
   ```bash
   export IMAP_SERVER=imap.example.com:993
   export IMAP_USER=your_email@example.com
   export IMAP_PASSWORD=your_password
   ```

4. Run the spike:
   ```bash
   go run ./cmd/spike
   ```

## Testing

Run the tests with:

```bash
go test ./cmd/spike
```

## Expected output

When run with valid credentials, you should see:

```
Starting IMAP spike...
Connecting to imap.example.com:993...
Connected to IMAP server
Logged in successfully
Server capabilities:
  - IMAP4rev1
  - THREAD=REFERENCES
    ✓ THREAD support detected
  ...
Inbox selected: 42 messages
Running THREAD command...
THREAD command executed successfully
Running SEARCH command: UNSEEN...
Found 3 messages matching criteria
UIDs: [123 124 125]
Fetching message UID 123...
Message details:
  - UID: 123
  - From: [sender@example.com]
  - Subject: Example Email
  ...
IMAP spike completed successfully
```

## What's next?

This spike proves that the core IMAP operations work. The next milestone (Milestone 2) will build a read-only web-based email client using this foundation.

## Notes

- This is a proof-of-concept CLI application, not production code
- Error handling is basic - production code will need more robust error handling
- The THREAD command uses a generic execute approach since go-imap v1.2.1 doesn't have built-in THREAD support
- Future milestones will add proper connection pooling, caching, and the web interface
