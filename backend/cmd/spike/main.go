package main

import (
	"fmt"
	"log"
	"os"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
)

// getIMAPCredentials gets IMAP credentials from environment variables.
func getIMAPCredentials() (server, user, password string, err error) {
	server = os.Getenv("IMAP_SERVER")
	user = os.Getenv("IMAP_USER")
	password = os.Getenv("IMAP_PASSWORD")

	if server == "" || user == "" || password == "" {
		return "", "", "", fmt.Errorf("IMAP_SERVER, IMAP_USER, and IMAP_PASSWORD environment variables are required")
	}

	return server, user, password, nil
}

// setupIMAPConnection connects and logs in to the IMAP server.
func setupIMAPConnection(server, user, password string) (*client.Client, error) {
	c, err := connectToIMAP(server)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := login(c, user, password); err != nil {
		err := c.Logout()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to log in: %w", err)
	}

	return c, nil
}

// performIMAPChecks performs capability checks and inbox selection.
func performIMAPChecks(c *client.Client) (*imap.MailboxStatus, error) {
	if err := checkCapabilities(c); err != nil {
		return nil, fmt.Errorf("failed to check capabilities: %w", err)
	}

	mbox, err := selectInbox(c)
	if err != nil {
		return nil, fmt.Errorf("failed to select inbox: %w", err)
	}

	return mbox, nil
}

// fetchUnseenOrFirstMessage fetches an unseen message or falls back to the first message.
func fetchUnseenOrFirstMessage(c *client.Client) error {
	uids, err := runSearchCommand(c, "UNSEEN")
	if err != nil {
		return fmt.Errorf("failed to run SEARCH command: %w", err)
	}

	if len(uids) > 0 {
		return fetchMessage(c, uids[0])
	}

	// Fallback: search for all messages
	log.Println("No unseen messages found - searching for all messages instead")
	allCriteria := imap.NewSearchCriteria()
	allUIDs, err := c.UidSearch(allCriteria)
	if err != nil {
		return fmt.Errorf("failed to search for all messages: %w", err)
	}

	if len(allUIDs) > 0 {
		log.Printf("Found %d total messages, fetching first one\n", len(allUIDs))
		return fetchMessage(c, allUIDs[0])
	}

	log.Println("No messages in inbox to fetch")
	return nil
}

func main() {
	log.Println("Starting IMAP spike...")

	// Get credentials
	imapServer, imapUser, imapPassword, err := getIMAPCredentials()
	if err != nil {
		log.Fatal(err)
	}

	// Connect and login
	c, err := setupIMAPConnection(imapServer, imapUser, imapPassword)
	if err != nil {
		log.Fatal(err)
	}
	defer func(c *client.Client) {
		if err := c.Logout(); err != nil {
			log.Printf("Failed to log out: %v", err)
		}
	}(c)

	log.Println("Connected to IMAP server")
	log.Println("Logged in successfully")

	// Perform checks and select the inbox
	mbox, err := performIMAPChecks(c)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inbox selected: %d messages\n", mbox.Messages)

	// Run THREAD command
	if err := runThreadCommand(c); err != nil {
		log.Fatalf("Failed to run THREAD command: %v\n", err)
	}

	// Fetch messages
	if err := fetchUnseenOrFirstMessage(c); err != nil {
		log.Fatalf("Failed to fetch message: %v\n", err)
	}

	log.Println("IMAP spike completed successfully")
}

// connectToIMAP connects to the IMAP server using TLS
func connectToIMAP(server string) (*client.Client, error) {
	log.Printf("Connecting to %s...\n", server)

	// Connect to the server using TLS
	c, err := client.DialTLS(server, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return c, nil
}

// login authenticates with the IMAP server
func login(c *client.Client, username, password string) error {
	if err := c.Login(username, password); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	return nil
}

// checkCapabilities checks and prints the server capabilities
func checkCapabilities(c *client.Client) error {
	caps, err := c.Capability()
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}

	log.Println("Server capabilities:")
	for capability := range caps {
		log.Printf("  - %s\n", capability)
		if capability == "THREAD=REFERENCES" || capability == "THREAD=ORDEREDSUBJECT" {
			log.Printf("    ✓ THREAD support detected\n")
		}
	}

	return nil
}

// selectInbox selects the Inbox folder
func selectInbox(c *client.Client) (*imap.MailboxStatus, error) {
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	log.Printf("Mailbox status:\n")
	log.Printf("  - Messages: %d\n", mbox.Messages)
	log.Printf("  - Recent: %d\n", mbox.Recent)
	log.Printf("  - Unseen: %d\n", mbox.Unseen)
	log.Printf("  - UIDNext: %d\n", mbox.UidNext)
	log.Printf("  - UIDValidity: %d\n", mbox.UidValidity)

	return mbox, nil
}

// runThreadCommand runs the THREAD command using the sortthread extension
func runThreadCommand(c *client.Client) error {
	if c == nil {
		return fmt.Errorf("client is nil")
	}

	log.Println("Running THREAD command...")

	// Create a thread client using the sortthread extension
	threadClient := sortthread.NewThreadClient(c)

	// Create search criteria for all messages
	searchCriteria := imap.NewSearchCriteria()

	// Execute UID THREAD command with REFERENCES algorithm
	// Note: We use UID THREAD instead of THREAD because UIDs are stable across
	// sessions, while sequence numbers change when a client deletes/moves messages.
	// This is the best practice for production IMAP applications.
	threads, err := threadClient.UidThread(sortthread.References, searchCriteria)
	if err != nil {
		return fmt.Errorf("THREAD command returned error: %w", err)
	}

	log.Println("THREAD command executed successfully")
	log.Printf("Found %d thread(s)\n", len(threads))

	// Print thread structure
	for i, thread := range threads {
		log.Printf("Thread %d: %v\n", i+1, thread)
	}

	return nil
}

// runSearchCommand runs a SEARCH command and returns matching UIDs
func runSearchCommand(c *client.Client, criteria string) ([]uint32, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	log.Printf("Running SEARCH command: %s...\n", criteria)

	// Create search criteria
	searchCriteria := imap.NewSearchCriteria()

	// Add UNSEEN flag to search criteria
	if criteria == "UNSEEN" {
		searchCriteria.WithoutFlags = []string{imap.SeenFlag}
	}

	uids, err := c.UidSearch(searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	log.Printf("Found %d messages matching criteria\n", len(uids))
	if len(uids) > 0 {
		log.Printf("UIDs: %v\n", uids)
	}

	return uids, nil
}

// fetchMessage fetches and prints a message's headers and body structure
func fetchMessage(c *client.Client, uid uint32) error {
	if c == nil {
		return fmt.Errorf("client is nil")
	}

	log.Printf("Fetching message UID %d...\n", uid)

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Fetch envelope, body structure, and flags
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchBodyStructure,
		imap.FetchFlags,
		imap.FetchUid,
	}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.UidFetch(seqSet, items, messages)
	}()

	msg := <-messages
	if msg == nil {
		return fmt.Errorf("server did not return message")
	}

	if err := <-done; err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	log.Println("Message details:")
	log.Printf("  - UID: %d\n", msg.Uid)
	log.Printf("  - Flags: %v\n", msg.Flags)

	if msg.Envelope != nil {
		log.Printf("  - From: %v\n", msg.Envelope.From)
		log.Printf("  - To: %v\n", msg.Envelope.To)
		log.Printf("  - Subject: %s\n", msg.Envelope.Subject)
		log.Printf("  - Date: %v\n", msg.Envelope.Date)
	}

	if msg.BodyStructure != nil {
		log.Printf("  - Body structure: %+v\n", msg.BodyStructure)
	}

	return nil
}
