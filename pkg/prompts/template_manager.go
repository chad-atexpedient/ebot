// Package prompts provides prompt engineering tools including templates,
// versioning, testing, and optimization capabilities.
package prompts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TemplateManager manages prompt templates and their versions.
type TemplateManager interface {
	// CreateTemplate creates a new prompt template
	CreateTemplate(ctx context.Context, template *PromptTemplate) error
	
	// GetTemplate retrieves a specific template version
	GetTemplate(ctx context.Context, templateID string, version int) (*PromptTemplate, error)
	
	// ListTemplates lists all templates with optional filtering
	ListTemplates(ctx context.Context, filters TemplateFilters) ([]*PromptTemplate, error)
	
	// UpdateTemplate creates a new version of a template
	UpdateTemplate(ctx context.Context, templateID string, content string, metadata map[string]string) (*PromptTemplate, error)
	
	// DeleteTemplate soft-deletes a template
	DeleteTemplate(ctx context.Context, templateID string) error
	
	// RenderTemplate renders a template with variables
	RenderTemplate(ctx context.Context, templateID string, version int, variables map[string]interface{}) (string, error)
	
	// ShareTemplate makes a template shareable
	ShareTemplate(ctx context.Context, templateID string, shareConfig ShareConfig) error
}

// PromptTemplate represents a reusable prompt template.
type PromptTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Version     int       `json:"version"`
	Content     string    `json:"content"`
	
	// Template variables
	Variables []TemplateVariable `json:"variables"`
	
	// Categorization
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	
	// Usage tracking
	UsageCount int       `json:"usageCount"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	
	// Quality metrics
	AvgRating    float64 `json:"avgRating,omitempty"`
	TotalRatings int     `json:"totalRatings,omitempty"`
	
	// Ownership & sharing
	OwnerID   string      `json:"ownerID"`
	IsPublic  bool        `json:"isPublic"`
	IsShared  bool        `json:"isShared"`
	SharedWith []string   `json:"sharedWith,omitempty"`
	
	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
	
	// Lifecycle
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// TemplateVariable defines a variable that can be substituted in a template.
type TemplateVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean, array
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
}

// TemplateFilters for querying templates.
type TemplateFilters struct {
	OwnerID       string   `json:"ownerID,omitempty"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	IsPublic      *bool    `json:"isPublic,omitempty"`
	IncludeDeleted bool    `json:"includeDeleted"`
}

// ShareConfig defines template sharing configuration.
type ShareConfig struct {
	IsPublic   bool     `json:"isPublic"`
	SharedWith []string `json:"sharedWith,omitempty"` // User IDs
}

// Implementation

type templateManager struct {
	templates map[string]map[int]*PromptTemplate // templateID -> version -> template
	mu        sync.RWMutex
}

// NewTemplateManager creates a new template manager.
func NewTemplateManager() TemplateManager {
	return &templateManager{
		templates: make(map[string]map[int]*PromptTemplate),
	}
}

func (m *templateManager) CreateTemplate(ctx context.Context, template *PromptTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}
	if template.Content == "" {
		return fmt.Errorf("template content is required")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Initialize version map if needed
	if m.templates[template.ID] == nil {
		m.templates[template.ID] = make(map[int]*PromptTemplate)
	}
	
	// Set version to 1 for new template
	if len(m.templates[template.ID]) == 0 {
		template.Version = 1
	} else {
		return fmt.Errorf("template %s already exists, use UpdateTemplate to create new version", template.ID)
	}
	
	// Extract variables from template content
	template.Variables = m.extractVariables(template.Content)
	
	// Set timestamps
	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now
	
	m.templates[template.ID][template.Version] = template
	
	return nil
}

func (m *templateManager) GetTemplate(ctx context.Context, templateID string, version int) (*PromptTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	versions, exists := m.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateID)
	}
	
	template, exists := versions[version]
	if !exists {
		return nil, fmt.Errorf("template %s version %d not found", templateID, version)
	}
	
	if template.DeletedAt != nil {
		return nil, fmt.Errorf("template %s has been deleted", templateID)
	}
	
	return template, nil
}

func (m *templateManager) ListTemplates(ctx context.Context, filters TemplateFilters) ([]*PromptTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result []*PromptTemplate
	
	for _, versions := range m.templates {
		// Get latest version only
		var latestVersion int
		for v := range versions {
			if v > latestVersion {
				latestVersion = v
			}
		}
		
		template := versions[latestVersion]
		
		// Apply filters
		if !filters.IncludeDeleted && template.DeletedAt != nil {
			continue
		}
		if filters.OwnerID != "" && template.OwnerID != filters.OwnerID {
			continue
		}
		if filters.Category != "" && template.Category != filters.Category {
			continue
		}
		if filters.IsPublic != nil && template.IsPublic != *filters.IsPublic {
			continue
		}
		if len(filters.Tags) > 0 {
			hasTag := false
			for _, filterTag := range filters.Tags {
				for _, templateTag := range template.Tags {
					if templateTag == filterTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		
		result = append(result, template)
	}
	
	return result, nil
}

func (m *templateManager) UpdateTemplate(ctx context.Context, templateID string, content string, metadata map[string]string) (*PromptTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	versions, exists := m.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateID)
	}
	
	// Find latest version
	var latestVersion int
	var baseTemplate *PromptTemplate
	for v, t := range versions {
		if v > latestVersion {
			latestVersion = v
			baseTemplate = t
		}
	}
	
	// Create new version
	newVersion := latestVersion + 1
	newTemplate := &PromptTemplate{
		ID:          templateID,
		Name:        baseTemplate.Name,
		Description: baseTemplate.Description,
		Version:     newVersion,
		Content:     content,
		Variables:   m.extractVariables(content),
		Category:    baseTemplate.Category,
		Tags:        baseTemplate.Tags,
		OwnerID:     baseTemplate.OwnerID,
		IsPublic:    baseTemplate.IsPublic,
		IsShared:    baseTemplate.IsShared,
		SharedWith:  baseTemplate.SharedWith,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	versions[newVersion] = newTemplate
	
	return newTemplate, nil
}

func (m *templateManager) DeleteTemplate(ctx context.Context, templateID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	versions, exists := m.templates[templateID]
	if !exists {
		return fmt.Errorf("template %s not found", templateID)
	}
	
	// Soft delete all versions
	now := time.Now()
	for _, template := range versions {
		template.DeletedAt = &now
	}
	
	return nil
}

func (m *templateManager) RenderTemplate(ctx context.Context, templateID string, version int, variables map[string]interface{}) (string, error) {
	template, err := m.GetTemplate(ctx, templateID, version)
	if err != nil {
		return "", err
	}
	
	// Simple variable substitution ({{variable_name}})
	result := template.Content
	
	for _, v := range template.Variables {
		value, exists := variables[v.Name]
		if !exists {
			if v.Required {
				return "", fmt.Errorf("required variable %s not provided", v.Name)
			}
			value = v.DefaultValue
		}
		
		if value != nil {
			placeholder := fmt.Sprintf("{{%s}}", v.Name)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
		}
	}
	
	// Update usage tracking
	m.mu.Lock()
	template.UsageCount++
	now := time.Now()
	template.LastUsedAt = &now
	m.mu.Unlock()
	
	return result, nil
}

func (m *templateManager) ShareTemplate(ctx context.Context, templateID string, shareConfig ShareConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	versions, exists := m.templates[templateID]
	if !exists {
		return fmt.Errorf("template %s not found", templateID)
	}
	
	// Apply sharing to all versions
	for _, template := range versions {
		template.IsPublic = shareConfig.IsPublic
		template.IsShared = len(shareConfig.SharedWith) > 0
		template.SharedWith = shareConfig.SharedWith
		template.UpdatedAt = time.Now()
	}
	
	return nil
}

func (m *templateManager) extractVariables(content string) []TemplateVariable {
	var variables []TemplateVariable
	seen := make(map[string]bool)
	
	// Simple extraction of {{variable}} patterns
	start := 0
	for {
		idx := strings.Index(content[start:], "{{")
		if idx == -1 {
			break
		}
		idx += start
		
		endIdx := strings.Index(content[idx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += idx
		
		varName := strings.TrimSpace(content[idx+2 : endIdx])
		if varName != "" && !seen[varName] {
			variables = append(variables, TemplateVariable{
				Name:     varName,
				Type:     "string",
				Required: true,
			})
			seen[varName] = true
		}
		
		start = endIdx + 2
	}
	
	return variables
}
