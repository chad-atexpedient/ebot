// Package salesforce provides Salesforce CRM integration
package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents a Salesforce client
type Client struct {
	instanceURL string
	accessToken string
	httpClient  *http.Client
}

// NewClient creates a new Salesforce client
func NewClient(instanceURL, accessToken string) *Client {
	return &Client{
		instanceURL: instanceURL,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Lead represents a Salesforce lead
type Lead struct {
	ID        string `json:"Id,omitempty"`
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
	Company   string `json:"Company"`
	Email     string `json:"Email"`
	Phone     string `json:"Phone,omitempty"`
	Status    string `json:"Status"`
}

// Account represents a Salesforce account
type Account struct {
	ID          string  `json:"Id,omitempty"`
	Name        string  `json:"Name"`
	Industry    string  `json:"Industry,omitempty"`
	AnnualRevenue float64 `json:"AnnualRevenue,omitempty"`
	Phone       string  `json:"Phone,omitempty"`
	Website     string  `json:"Website,omitempty"`
}

// Opportunity represents a sales opportunity
type Opportunity struct {
	ID          string    `json:"Id,omitempty"`
	Name        string    `json:"Name"`
	AccountID   string    `json:"AccountId"`
	Stage       string    `json:"StageName"`
	Amount      float64   `json:"Amount"`
	CloseDate   time.Time `json:"CloseDate"`
	Probability int       `json:"Probability"`
}

// Case represents a customer support case
type Case struct {
	ID          string `json:"Id,omitempty"`
	Subject     string `json:"Subject"`
	Description string `json:"Description"`
	Status      string `json:"Status"`
	Priority    string `json:"Priority"`
	AccountID   string `json:"AccountId,omitempty"`
}

// CreateLead creates a new lead
func (c *Client) CreateLead(ctx context.Context, lead *Lead) (string, error) {
	url := fmt.Sprintf("%s/services/data/v57.0/sobjects/Lead", c.instanceURL)
	return c.createRecord(ctx, url, lead)
}

// GetLead retrieves a lead by ID
func (c *Client) GetLead(ctx context.Context, id string) (*Lead, error) {
	url := fmt.Sprintf("%s/services/data/v57.0/sobjects/Lead/%s", c.instanceURL, id)
	var lead Lead
	err := c.getRecord(ctx, url, &lead)
	return &lead, err
}

// CreateAccount creates a new account
func (c *Client) CreateAccount(ctx context.Context, account *Account) (string, error) {
	url := fmt.Sprintf("%s/services/data/v57.0/sobjects/Account", c.instanceURL)
	return c.createRecord(ctx, url, account)
}

// CreateOpportunity creates a new opportunity
func (c *Client) CreateOpportunity(ctx context.Context, opp *Opportunity) (string, error) {
	url := fmt.Sprintf("%s/services/data/v57.0/sobjects/Opportunity", c.instanceURL)
	return c.createRecord(ctx, url, opp)
}

// CreateCase creates a support case
func (c *Client) CreateCase(ctx context.Context, caseObj *Case) (string, error) {
	url := fmt.Sprintf("%s/services/data/v57.0/sobjects/Case", c.instanceURL)
	return c.createRecord(ctx, url, caseObj)
}

// Query executes a SOQL query
func (c *Client) Query(ctx context.Context, soql string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/services/data/v57.0/query?q=%s", c.instanceURL, soql)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result struct {
		Records []map[string]interface{} `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Records, nil
}

// Helper methods
func (c *Client) createRecord(ctx context.Context, url string, record interface{}) (string, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	
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

func (c *Client) getRecord(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return json.NewDecoder(resp.Body).Decode(result)
}
