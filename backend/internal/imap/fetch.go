package imap

import (
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// FetchMessageHeaders fetches message headers for the given UIDs.
func FetchMessageHeaders(c *client.Client, uids []uint32) ([]*imap.Message, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	if len(uids) == 0 {
		return []*imap.Message{}, nil
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	// Fetch envelope, body structure, flags, and UID
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchBodyStructure,
		imap.FetchFlags,
		imap.FetchUid,
	}

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- c.UidFetch(seqSet, items, messages)
	}()

	var result []*imap.Message
	for msg := range messages {
		result = append(result, msg)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return result, nil
}

// FetchFullMessage fetches the full message body for the given UID.
func FetchFullMessage(c *client.Client, uid uint32) (*imap.Message, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Fetch envelope, body structure, flags, and UID first
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
		return nil, fmt.Errorf("server did not return message")
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	// Now fetch the body using the body structure
	if msg.BodyStructure != nil {
		section := &imap.BodySectionName{}
		bodyItem := section.FetchItem()
		bodyItems := []imap.FetchItem{bodyItem}

		bodyMessages := make(chan *imap.Message, 1)
		bodyDone := make(chan error, 1)

		go func() {
			bodyDone <- c.UidFetch(seqSet, bodyItems, bodyMessages)
		}()

		bodyMsg := <-bodyMessages
		if bodyMsg != nil {
			msg.Body = bodyMsg.Body
		}
		<-bodyDone
	}

	return msg, nil
}

// SearchUIDsSince searches for all UIDs greater than or equal to the given UID.
// This is used for incremental sync to find only new messages.
func SearchUIDsSince(c *client.Client, minUID uint32) ([]uint32, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	// Create a SeqSet with the range minUID:*
	// This represents all UIDs from minUID to the highest UID
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(minUID, 0) // 0 means "highest UID"

	// Use SEARCH to find UIDs in this range
	// We'll use a simple approach: fetch UIDs for all messages in the range
	// Actually, IMAP SEARCH doesn't work with SeqSet directly for UID ranges
	// Instead, we need to use the SEARCH command with UID criteria

	// The go-imap library's UidSearch doesn't directly support UID ranges,
	// but we can fetch all UIDs and filter them, or use a different approach.
	// For now, let's fetch all UIDs and filter - this is still efficient
	// because we're only getting UID numbers, not message content.

	searchCriteria := imap.NewSearchCriteria()
	uids, err := c.UidSearch(searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search for UIDs: %w", err)
	}

	// Filter to only UIDs >= minUID
	var filteredUIDs []uint32
	for _, uid := range uids {
		if uid >= minUID {
			filteredUIDs = append(filteredUIDs, uid)
		}
	}

	return filteredUIDs, nil
}
