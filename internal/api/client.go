package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for AgentAPI
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Status represents the agent status response
type Status struct {
	Status string `json:"status"` // "running" or "stable"
}

// Message represents a message to send to the agent
type Message struct {
	Content string `json:"content"`
	Type    string `json:"type"` // "user" or "raw"
}

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
}

// NewClient creates a new AgentAPI client
func NewClient(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://localhost:%d", port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetStatus returns the current agent status
func (c *Client) GetStatus() (*Status, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status request failed: %s", string(body))
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}

	return &status, nil
}

// SendMessage sends a message to the agent
func (c *Client) SendMessage(content string, msgType string) error {
	msg := Message{
		Content: content,
		Type:    msgType,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/message",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("message request failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetMessages returns the conversation history
func (c *Client) GetMessages() ([]ConversationMessage, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/messages")
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("messages request failed: %s", string(body))
	}

	var messages []ConversationMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// WaitForStable waits until the agent is in stable state
func (c *Client) WaitForStable(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := c.GetStatus()
		if err != nil {
			// Agent might not be ready yet, continue waiting
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if status.Status == "stable" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for stable state")
}

// WaitForHealthy waits until the agent responds to health checks
func (c *Client) WaitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		_, err := c.GetStatus()
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for agent to be healthy")
}

// IsRunning returns true if the agent is currently processing
func (c *Client) IsRunning() (bool, error) {
	status, err := c.GetStatus()
	if err != nil {
		return false, err
	}
	return status.Status == "running", nil
}
