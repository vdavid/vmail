package imap

import (
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// FetchMessageHeaders fetches message headers for the given UIDs.
// Returns envelope, body structure, flags, and UID for each message.
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
// First fetches headers and body structure, then fetches the actual body content.
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
		if err := <-bodyDone; err != nil {
			// Log error but don't fail - we still have headers and structure
			// The body fetch failure is non-critical for basic message retrieval
			return nil, fmt.Errorf("failed to fetch message body: %w", err)
		}
	}

	return msg, nil
}

// SearchUIDsSince searches for all UIDs greater than or equal to the given UID.
// This is used for incremental sync to find only new messages.
//
// Performance note: This function fetches all UIDs and filters them client-side.
// While IMAP supports UID SEARCH with ranges (e.g., "UID minUID:*"), the go-imap
// library's SearchCriteria doesn't expose this capability directly. The current
// approach is acceptable because:
// 1. We're only fetching UID numbers (not message content), which is fast
// 2. Client-side filtering is efficient for typical mailbox sizes
// 3. Most mailboxes have < 100k messages, making this approach practical
//
// For very large mailboxes (> 1M messages), consider:
// - Using IMAP's native UID SEARCH with ranges if go-imap adds support
// - Implementing batch fetching with pagination
// - Using server-side filtering if the IMAP server supports extensions
func SearchUIDsSince(c *client.Client, minUID uint32) ([]uint32, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	// Fetch all UIDs from the server
	// Note: go-imap's UidSearch doesn't support UID ranges in SearchCriteria,
	// so we fetch all UIDs and filter client-side. This is efficient for typical
	// mailbox sizes since we're only transferring UID numbers.
	searchCriteria := imap.NewSearchCriteria()
	uids, err := c.UidSearch(searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search for UIDs: %w", err)
	}

	// Early exit if no UIDs or minUID is higher than all UIDs
	if len(uids) == 0 {
		return []uint32{}, nil
	}

	// If minUID is higher than the highest UID, return empty
	if minUID > uids[len(uids)-1] {
		return []uint32{}, nil
	}

	// Filter to only UIDs >= minUID
	// Pre-allocate slice with estimated capacity (assuming UIDs are roughly evenly distributed)
	estimatedSize := len(uids)
	if minUID > 0 {
		// Rough estimate: if minUID is halfway, we'll get about half the UIDs
		estimatedSize = len(uids) / 2
	}
	filteredUIDs := make([]uint32, 0, estimatedSize)
	for _, uid := range uids {
		if uid >= minUID {
			filteredUIDs = append(filteredUIDs, uid)
		}
	}

	return filteredUIDs, nil
}
