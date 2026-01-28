# Enterprise Integrations Guide

This guide covers ebot's enterprise integrations with major platforms used in organizations.

## Overview

ebot integrates with:
- **Microsoft Teams** - Chat, collaboration, bots
- **Salesforce** - CRM, leads, opportunities
- **ServiceNow** - ITSM, incidents, changes
- **Jira + Confluence** - Project management, documentation
- **Google Workspace** - Docs, Sheets, Drive, Calendar

## Microsoft Teams Integration

### Setup

```go
import "github.com/chad-atexpedient/ebot/pkg/integrations/teams"

client := teams.NewClient(botID, botPassword)
```

### Send Message

```go
msg := &teams.Message{
    Type: "message",
    Text: "Hello from ebot!",
    Conversation: teams.Conversation{
        ID: "conversation-id",
    },
}

err := client.SendMessage(ctx, msg)
```

### Send Adaptive Card

```go
card := &teams.AdaptiveCard{
    Type: "AdaptiveCard",
    Version: "1.4",
    Body: []interface{}{
        map[string]string{
            "type": "TextBlock",
            "text": "ebot Alert",
            "weight": "Bolder",
            "size": "Large",
        },
    },
}

err := client.SendAdaptiveCard(ctx, conversationID, card)
```

## Salesforce Integration

### Setup

```go
import "github.com/chad-atexpedient/ebot/pkg/integrations/salesforce"

client := salesforce.NewClient(instanceURL, accessToken)
```

### Create Lead

```go
lead := &salesforce.Lead{
    FirstName: "John",
    LastName: "Doe",
    Company: "Acme Corp",
    Email: "john@acme.com",
    Status: "New",
}

leadID, err := client.CreateLead(ctx, lead)
```

### Query Records

```go
soql := "SELECT Id, Name FROM Account WHERE Industry = 'Technology'"
records, err := client.Query(ctx, soql)
```

### Create Opportunity

```go
opp := &salesforce.Opportunity{
    Name: "Q1 Enterprise Deal",
    AccountID: accountID,
    Stage: "Prospecting",
    Amount: 50000.00,
    CloseDate: time.Now().AddDate(0, 3, 0),
    Probability: 25,
}

oppID, err := client.CreateOpportunity(ctx, opp)
```

## ServiceNow Integration

### Setup

```go
import "github.com/chad-atexpedient/ebot/pkg/integrations/servicenow"

client := servicenow.NewClient(baseURL, username, password)
```

### Create Incident

```go
incident := &servicenow.Incident{
    ShortDescription: "Server Down",
    Description: "Production server is not responding",
    Urgency: "1",
    Impact: "1",
    State: "1", // New
}

sysID, err := client.CreateIncident(ctx, incident)
```

### Update Incident

```go
incident.State = "2" // In Progress
incident.AssignedTo = "admin"

err := client.UpdateIncident(ctx, sysID, incident)
```

### Create Change Request

```go
change := &servicenow.ChangeRequest{
    ShortDescription: "Upgrade Database",
    Description: "Upgrade production DB to v12",
    Type: "Normal",
    Risk: "Medium",
    StartDate: time.Now().AddDate(0, 0, 7),
    EndDate: time.Now().AddDate(0, 0, 8),
    State: "1",
}

changeID, err := client.CreateChangeRequest(ctx, change)
```

## Jira + Confluence Integration

### Setup

```go
import "github.com/chad-atexpedient/ebot/pkg/integrations/atlassian"

client := atlassian.NewClient(baseURL, email, apiToken)
```

### Create Jira Issue

```go
issue := &atlassian.Issue{
    Fields: atlassian.IssueFields{
        Project: atlassian.Project{Key: "PROJ"},
        Summary: "Fix login bug",
        Description: "Users cannot login with SSO",
        IssueType: atlassian.IssueType{Name: "Bug"},
        Priority: atlassian.Priority{Name: "High"},
    },
}

issueKey, err := client.CreateIssue(ctx, issue)
```

### Create Confluence Page

```go
page := &atlassian.Page{
    Type: "page",
    Title: "Architecture Documentation",
    Space: atlassian.Space{Key: "DEV"},
    Body: atlassian.PageBody{
        Storage: atlassian.Storage{
            Value: "<p>Architecture content here</p>",
            Representation: "storage",
        },
    },
}

pageID, err := client.CreatePage(ctx, page)
```

## Google Workspace Integration

### Setup

```go
import "github.com/chad-atexpedient/ebot/pkg/integrations/google"

client := google.NewWorkspaceClient(accessToken)
```

### Create Google Doc

```go
docID, err := client.CreateDocument(ctx, "Project Plan")
```

### Create Google Sheet

```go
sheetID, err := client.CreateSpreadsheet(ctx, "Q1 Budget")

// Update cells
values := [][]interface{}{
    {"Item", "Cost"},
    {"Software", 10000},
    {"Hardware", 25000},
}
err = client.UpdateSheet(ctx, sheetID, "A1:B3", values)
```

### Upload to Google Drive

```go
file, _ := os.Open("report.pdf")
defer file.Close()

fileID, err := client.UploadFile(ctx, "report.pdf", "application/pdf", file)
```

### Create Calendar Event

```go
event := &google.CalendarEvent{
    Summary: "Team Meeting",
    Description: "Weekly sync",
    Start: google.EventDateTime{
        DateTime: "2026-02-01T10:00:00-05:00",
        TimeZone: "America/New_York",
    },
    End: google.EventDateTime{
        DateTime: "2026-02-01T11:00:00-05:00",
        TimeZone: "America/New_York",
    },
    Attendees: []google.Attendee{
        {Email: "team@example.com"},
    },
}

eventID, err := client.CreateCalendarEvent(ctx, "primary", event)
```

## Use Cases

### Automated Incident Creation

```go
// When ebot detects an issue, create ServiceNow incident
incident := &servicenow.Incident{
    ShortDescription: "High Memory Usage Detected",
    Description: fmt.Sprintf("Server %s memory at 95%%", serverName),
    Urgency: "1",
    Impact: "2",
}
sysID, _ := snowClient.CreateIncident(ctx, incident)

// Notify team via Teams
msg := &teams.Message{
    Text: fmt.Sprintf("Incident %s created: High memory on %s", sysID, serverName),
    Conversation: teams.Conversation{ID: teamChannelID},
}
teamsClient.SendMessage(ctx, msg)
```

### Lead to Opportunity Workflow

```go
// Create lead in Salesforce
lead := &salesforce.Lead{
    FirstName: "Jane",
    LastName: "Smith",
    Company: "Tech Corp",
    Email: "jane@techcorp.com",
    Status: "New",
}
leadID, _ := sfClient.CreateLead(ctx, lead)

// Create tracking issue in Jira
issue := &atlassian.Issue{
    Fields: atlassian.IssueFields{
        Project: atlassian.Project{Key: "SALES"},
        Summary: fmt.Sprintf("Follow up: %s %s", lead.FirstName, lead.LastName),
        IssueType: atlassian.IssueType{Name: "Task"},
    },
}
jiraClient.CreateIssue(ctx, issue)
```

### Documentation Generation

```go
// Generate project documentation in Confluence
page := &atlassian.Page{
    Type: "page",
    Title: "Q1 Project Summary",
    Space: atlassian.Space{Key: "PROJ"},
    Body: atlassian.PageBody{
        Storage: atlassian.Storage{
            Value: generateProjectReport(),
            Representation: "storage",
        },
    },
}
confluenceClient.CreatePage(ctx, page)

// Also create in Google Docs for external sharing
docID, _ := googleClient.CreateDocument(ctx, "Q1 Project Summary")
```

## Authentication

### OAuth 2.0 Flow

Most integrations use OAuth 2.0. Store tokens securely in ebot's credential system.

### Service Accounts

For automated workflows, use service accounts:
- **Google Workspace**: Service account with domain-wide delegation
- **Salesforce**: Connected app with JWT bearer flow
- **ServiceNow**: Basic auth or OAuth 2.0
- **Jira**: API tokens or OAuth 2.0

## Best Practices

1. **Rate Limiting**: Respect API rate limits
2. **Error Handling**: Retry on transient errors
3. **Logging**: Log all integration activities
4. **Security**: Never log credentials
5. **Testing**: Test integrations in sandbox environments

## Troubleshooting

### Authentication Errors

- Verify tokens haven't expired
- Check token scopes/permissions
- Ensure service accounts have correct roles

### Rate Limiting

- Implement exponential backoff
- Cache responses when possible
- Batch operations when supported

### Network Issues

- Set appropriate timeouts
- Handle connection errors gracefully
- Use circuit breakers for failing services

## Support

For integration issues:
- Check integration-specific documentation
- Review API logs in ebot admin UI
- Contact Expedient support
