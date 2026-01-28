// Package servicenow provides ServiceNow ITSM integration
package servicenow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents a ServiceNow client
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClient creates a new ServiceNow client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Incident represents a ServiceNow incident
type Incident struct {
	SysID             string `json:"sys_id,omitempty"`
	Number            string `json:"number,omitempty"`
	ShortDescription  string `json:"short_description"`
	Description       string `json:"description"`
	Urgency           string `json:"urgency"`
	Impact            string `json:"impact"`
	Priority          string `json:"priority,omitempty"`
	State             string `json:"state"`
	AssignedTo        string `json:"assigned_to,omitempty"`
	AssignmentGroup   string `json:"assignment_group,omitempty"`
}

// Problem represents a ServiceNow problem
type Problem struct {
	SysID            string `json:"sys_id,omitempty"`
	Number           string `json:"number,omitempty"`
	ShortDescription string `json:"short_description"`
	Description      string `json:"description"`
	State            string `json:"state"`
	Priority         string `json:"priority"`
}

// ChangeRequest represents a change request
type ChangeRequest struct {
	SysID            string    `json:"sys_id,omitempty"`
	Number           string    `json:"number,omitempty"`
	ShortDescription string    `json:"short_description"`
	Description      string    `json:"description"`
	Type             string    `json:"type"`
	Risk             string    `json:"risk"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	State            string    `json:"state"`
}

// CreateIncident creates a new incident
func (c *Client) CreateIncident(ctx context.Context, incident *Incident) (string, error) {
	url := fmt.Sprintf("%s/api/now/table/incident", c.baseURL)
	return c.createRecord(ctx, url, incident)
}

// GetIncident retrieves an incident
func (c *Client) GetIncident(ctx context.Context, sysID string) (*Incident, error) {
	url := fmt.Sprintf("%s/api/now/table/incident/%s", c.baseURL, sysID)
	var incident Incident
	err := c.getRecord(ctx, url, &incident)
	return &incident, err
}

// UpdateIncident updates an incident
func (c *Client) UpdateIncident(ctx context.Context, sysID string, incident *Incident) error {
	url := fmt.Sprintf("%s/api/now/table/incident/%s", c.baseURL, sysID)
	return c.updateRecord(ctx, url, incident)
}

// CreateProblem creates a new problem
func (c *Client) CreateProblem(ctx context.Context, problem *Problem) (string, error) {
	url := fmt.Sprintf("%s/api/now/table/problem", c.baseURL)
	return c.createRecord(ctx, url, problem)
}

// CreateChangeRequest creates a new change request
func (c *Client) CreateChangeRequest(ctx context.Context, change *ChangeRequest) (string, error) {
	url := fmt.Sprintf("%s/api/now/table/change_request", c.baseURL)
	return c.createRecord(ctx, url, change)
}

// Helper methods
func (c *Client) createRecord(ctx context.Context, url string, record interface{}) (string, error) {
	data, _ := json.Marshal(map[string]interface{}{"result": record})
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
		Result struct {
			SysID string `json:"sys_id"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Result.SysID, nil
}

func (c *Client) getRecord(ctx context.Context, url string, result interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	c.setHeaders(req)
	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) updateRecord(ctx context.Context, url string, record interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, "PUT", url, nil)
	c.setHeaders(req)
	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
