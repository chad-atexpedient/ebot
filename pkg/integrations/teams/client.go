// Package teams provides Microsoft Teams integration for ebot
package teams

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents a Microsoft Teams client
type Client struct {
	botID       string
	botPassword string
	httpClient  *http.Client
	baseURL     string
}

// NewClient creates a new Teams client
func NewClient(botID, botPassword string) *Client {
	return &Client{
		botID:       botID,
		botPassword: botPassword,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://smba.trafficmanager.net/apis",
	}
}

// Message represents a Teams message
type Message struct {
	Type         string       `json:"type"`
	Text         string       `json:"text"`
	TextFormat   string       `json:"textFormat,omitempty"`
	Attachments  []Attachment `json:"attachments,omitempty"`
	ChannelID    string       `json:"channelId,omitempty"`
	Conversation Conversation `json:"conversation"`
}

// Attachment represents a message attachment
type Attachment struct {
	ContentType string      `json:"contentType"`
	Content     interface{} `json:"content"`
}

// Conversation represents a Teams conversation
type Conversation struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// AdaptiveCard represents an Adaptive Card
type AdaptiveCard struct {
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Body    []interface{} `json:"body"`
	Actions []interface{} `json:"actions,omitempty"`
}

// SendMessage sends a message to a Teams channel or conversation
func (c *Client) SendMessage(ctx context.Context, msg *Message) error {
	url := fmt.Sprintf("%s/v3/conversations/%s/activities", c.baseURL, msg.Conversation.ID)
	
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add bot authentication (simplified - real implementation would use JWT)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.botPassword))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("teams API error: %d", resp.StatusCode)
	}

	return nil
}

// SendAdaptiveCard sends an Adaptive Card to Teams
func (c *Client) SendAdaptiveCard(ctx context.Context, conversationID string, card *AdaptiveCard) error {
	msg := &Message{
		Type: "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content:     card,
			},
		},
		Conversation: Conversation{
			ID: conversationID,
		},
	}
	return c.SendMessage(ctx, msg)
}

// CreateConversation creates a new conversation
func (c *Client) CreateConversation(ctx context.Context, members []string) (string, error) {
	// Implementation would create a conversation with specified members
	// Returning dummy ID for now
	return "conversation-id", nil
}

// GetChannels retrieves all channels in a team
func (c *Client) GetChannels(ctx context.Context, teamID string) ([]Channel, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.botPassword))
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channels: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Value []Channel `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Value, nil
}

// Channel represents a Teams channel
type Channel struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// UploadFile uploads a file to a Teams channel
func (c *Client) UploadFile(ctx context.Context, channelID, filePath string) error {
	// Implementation would upload file to Teams
	return nil
}
