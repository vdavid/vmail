package main

import (
	"fmt"
	"log"
	"os"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
)

func main() {
	log.Println("Starting IMAP spike...")

	// Get credentials from environment variables
	imapServer := os.Getenv("IMAP_SERVER")
	imapUser := os.Getenv("IMAP_USER")
	imapPassword := os.Getenv("IMAP_PASSWORD")

	if imapServer == "" || imapUser == "" || imapPassword == "" {
		log.Fatal("Error: IMAP_SERVER, IMAP_USER, and IMAP_PASSWORD environment variables are required")
	}

	// Connect to the server
	c, err := connectToIMAP(imapServer)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func(c *client.Client) {
		err := c.Logout()
		if err != nil {
			log.Printf("Failed to log out: %v", err)
		}
	}(c)

	log.Println("Connected to IMAP server")

	// Log in
	if err := login(c, imapUser, imapPassword); err != nil {
		log.Fatalf("Failed to log in: %v", err)
	}

	log.Println("Logged in successfully")

	// Check capabilities
	if err := checkCapabilities(c); err != nil {
		log.Fatalf("Failed to check capabilities: %v", err)
	}

	// Select inbox
	mbox, err := selectInbox(c)
	if err != nil {
		log.Fatalf("Failed to select inbox: %v", err)
	}

	log.Printf("Inbox selected: %d messages\n", mbox.Messages)

	// Run THREAD command
	if err := runThreadCommand(c); err != nil {
		log.Fatalf("Failed to run THREAD command: %v\n", err)
	}

	// Run SEARCH command
	uids, err := runSearchCommand(c, "UNSEEN")
	if err != nil {
		log.Fatalf("Failed to run SEARCH command: %v\n", err)
	}

	if len(uids) > 0 {
		// Fetch a message if we found any
		if err := fetchMessage(c, uids[0]); err != nil {
			log.Fatalf("Failed to fetch message: %v\n", err)
		}
	} else {
		log.Println("No unseen messages found - searching for all messages instead")
		// Try searching for all messages
		allCriteria := imap.NewSearchCriteria()
		allUIDs, err := c.UidSearch(allCriteria)
		if err != nil {
			log.Fatalf("Failed to search for all messages: %v\n", err)
		}
		if len(allUIDs) > 0 {
			log.Printf("Found %d total messages, fetching first one\n", len(allUIDs))
			if err := fetchMessage(c, allUIDs[0]); err != nil {
				log.Fatalf("Failed to fetch message: %v\n", err)
			}
		} else {
			log.Println("No messages in inbox to fetch")
		}
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
