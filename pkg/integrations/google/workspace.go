// Package google provides Google Workspace integration
package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WorkspaceClient represents a Google Workspace client
type WorkspaceClient struct {
	accessToken string
	httpClient  *http.Client
}

// NewWorkspaceClient creates a new Google Workspace client
func NewWorkspaceClient(accessToken string) *WorkspaceClient {
	return &WorkspaceClient{
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Document represents a Google Doc
type Document struct {
	DocumentID string `json:"documentId,omitempty"`
	Title      string `json:"title"`
	Body       Body   `json:"body,omitempty"`
}

type Body struct {
	Content []Content `json:"content"`
}

type Content struct {
	Paragraph *Paragraph `json:"paragraph,omitempty"`
}

type Paragraph struct {
	Elements []Element `json:"elements"`
}

type Element struct {
	TextRun *TextRun `json:"textRun,omitempty"`
}

type TextRun struct {
	Content string `json:"content"`
}

// Spreadsheet represents a Google Sheet
type Spreadsheet struct {
	SpreadsheetID string  `json:"spreadsheetId,omitempty"`
	Title         string  `json:"properties.title"`
	Sheets        []Sheet `json:"sheets,omitempty"`
}

type Sheet struct {
	Title string     `json:"properties.title"`
	Data  [][]string `json:"data,omitempty"`
}

// DriveFile represents a Google Drive file
type DriveFile struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Parents  []string `json:"parents,omitempty"`
}

// CalendarEvent represents a Google Calendar event
type CalendarEvent struct {
	ID          string        `json:"id,omitempty"`
	Summary     string        `json:"summary"`
	Description string        `json:"description,omitempty"`
	Start       EventDateTime `json:"start"`
	End         EventDateTime `json:"end"`
	Attendees   []Attendee    `json:"attendees,omitempty"`
}

type EventDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type Attendee struct {
	Email string `json:"email"`
}

// CreateDocument creates a new Google Doc
func (c *WorkspaceClient) CreateDocument(ctx context.Context, title string) (string, error) {
	url := "https://docs.googleapis.com/v1/documents"
	
	payload := map[string]interface{}{
		"title": title,
	}
	
	data, _ := json.Marshal(payload)
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
	
	var doc Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	
	return doc.DocumentID, nil
}

// GetDocument retrieves a Google Doc
func (c *WorkspaceClient) GetDocument(ctx context.Context, docID string) (*Document, error) {
	url := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s", docID)
	
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
	
	var doc Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	
	return &doc, nil
}

// CreateSpreadsheet creates a new Google Sheet
func (c *WorkspaceClient) CreateSpreadsheet(ctx context.Context, title string) (string, error) {
	url := "https://sheets.googleapis.com/v4/spreadsheets"
	
	payload := map[string]interface{}{
		"properties": map[string]string{"title": title},
	}
	
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()
	
	var sheet Spreadsheet
	json.NewDecoder(resp.Body).Decode(&sheet)
	
	return sheet.SpreadsheetID, nil
}

// UpdateSheet updates cells in a Google Sheet
func (c *WorkspaceClient) UpdateSheet(ctx context.Context, sheetID, range_ string, values [][]interface{}) error {
	url := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?valueInputOption=RAW", sheetID, range_)
	
	payload := map[string]interface{}{
		"values": values,
	}
	
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "PUT", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()
	
	return nil
}

// UploadFile uploads a file to Google Drive
func (c *WorkspaceClient) UploadFile(ctx context.Context, name, mimeType string, content io.Reader) (string, error) {
	url := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart"
	
	// Simplified upload logic
	req, err := http.NewRequestWithContext(ctx, "POST", url, content)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", mimeType)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var file DriveFile
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return "", err
	}
	
	return file.ID, nil
}

// CreateCalendarEvent creates a new calendar event
func (c *WorkspaceClient) CreateCalendarEvent(ctx context.Context, calendarID string, event *CalendarEvent) (string, error) {
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", calendarID)
	
	data, _ := json.Marshal(event)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()
	
	var createdEvent CalendarEvent
	json.NewDecoder(resp.Body).Decode(&createdEvent)
	
	return createdEvent.ID, nil
}
