// Package atlassian provides Jira and Confluence integration
package atlassian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents an Atlassian client (Jira + Confluence)
type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Atlassian client
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		email:      email,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Issue represents a Jira issue
type Issue struct {
	ID          string            `json:"id,omitempty"`
	Key         string            `json:"key,omitempty"`
	Fields      IssueFields       `json:"fields"`
}

// IssueFields represents issue fields
type IssueFields struct {
	Project     Project   `json:"project"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	IssueType   IssueType `json:"issuetype"`
	Priority    Priority  `json:"priority,omitempty"`
	Assignee    *User     `json:"assignee,omitempty"`
	Status      *Status   `json:"status,omitempty"`
}

type Project struct {
	Key string `json:"key"`
}

type IssueType struct {
	Name string `json:"name"`
}

type Priority struct {
	Name string `json:"name"`
}

type User struct {
	AccountID string `json:"accountId"`
}

type Status struct {
	Name string `json:"name"`
}

// Page represents a Confluence page
type Page struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Title   string      `json:"title"`
	Space   Space       `json:"space"`
	Body    PageBody    `json:"body"`
	Version *Version    `json:"version,omitempty"`
}

type Space struct {
	Key string `json:"key"`
}

type PageBody struct {
	Storage Storage `json:"storage"`
}

type Storage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type Version struct {
	Number int `json:"number"`
}

// CreateIssue creates a new Jira issue
func (c *Client) CreateIssue(ctx context.Context, issue *Issue) (string, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue", c.baseURL)
	
	data, err := json.Marshal(issue)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}
	
	c.setHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	return result.Key, nil
}

// GetIssue retrieves a Jira issue
func (c *Client) GetIssue(ctx context.Context, key string) (*Issue, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, key)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	c.setHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}
	
	return &issue, nil
}

// CreatePage creates a Confluence page
func (c *Client) CreatePage(ctx context.Context, page *Page) (string, error) {
	url := fmt.Sprintf("%s/wiki/rest/api/content", c.baseURL)
	
	data, err := json.Marshal(page)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}
	
	c.setHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	return result.ID, nil
}

// GetPage retrieves a Confluence page
func (c *Client) GetPage(ctx context.Context, id string) (*Page, error) {
	url := fmt.Sprintf("%s/wiki/rest/api/content/%s?expand=body.storage,version", c.baseURL, id)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	c.setHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	
	return &page, nil
}

// UpdateIssue updates a Jira issue
func (c *Client) UpdateIssue(ctx context.Context, key string, fields IssueFields) error {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, key)
	
	update := map[string]interface{}{
		"fields": fields,
	}
	
	data, _ := json.Marshal(update)
	req, _ := http.NewRequestWithContext(ctx, "PUT", url, nil)
	c.setHeaders(req)
	
	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()
	
	return nil
}

// Helper method
func (c *Client) setHeaders(req *http.Request) {
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
