package imap

import (
	"fmt"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
)

// RunThreadCommand runs the THREAD command and returns the thread structure.
// Uses the REFERENCES algorithm to build thread relationships.
func RunThreadCommand(c *client.Client) ([]*sortthread.Thread, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	// Create a thread client using the sortthread extension
	threadClient := sortthread.NewThreadClient(c)

	// Create search criteria for all messages
	searchCriteria := imap.NewSearchCriteria()

	// Execute UID THREAD command with the REFERENCES algorithm
	threads, err := threadClient.UidThread(sortthread.References, searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("THREAD command returned error: %w", err)
	}

	return threads, nil
}
