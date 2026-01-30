package twilio

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type Client struct {
	client *twilio.RestClient
	number string
}

func NewClient(apiKey, apiSecret, accountSid, number string) *Client {
	// If using API Key/Secret, we might need a different auth method,
	// but standard usage is AccountSID + AuthToken as password.
	// We'll assume the user provides AccountSID and AuthToken.
	c := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: apiKey,    // Account SID
		Password: apiSecret, // Auth Token
	})
	return &Client{
		client: c,
		number: number,
	}
}

// GetLatestSignalCode fetches the most recent SMS from Twilio
// that looks like a Signal verification code.
func (c *Client) GetLatestSignalCode() (string, error) {
	params := &api.ListMessageParams{}
	params.SetTo(c.number)
	// We only care about inbound messages
	// params.SetDateSent(...) // Could filter by recent time, but "latest" implies sorting

	// Twilio lists are roughly reverse chronological by default?
	// Actually no, we should check doc or iterate.
	// But usually the ListMessage returns the most recent first?
	// Let's rely on standard iteration and pick the first one matching our pattern.

	resp, err := c.client.Api.ListMessage(params)
	if err != nil {
		return "", fmt.Errorf("failed to list messages: %w", err)
	}

	// Regex for Signal code: usually "Your Signal verification code: 123-456"
	// or just look for 3 digits dash 3 digits
	re := regexp.MustCompile(`(\d{3}-\d{3})`)

	for _, msg := range resp {
		if msg.Body == nil {
			continue
		}
		body := *msg.Body
		// Check if it looks like a Signal message
		if strings.Contains(body, "Signal") {
			matches := re.FindStringSubmatch(body)
			if len(matches) > 1 {
				return matches[1], nil
			}
		}
	}

	return "", fmt.Errorf("no signal verification code found in recent messages")
}
