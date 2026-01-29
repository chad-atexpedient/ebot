// Package knowledge provides knowledge graph capabilities for entity extraction
// and relationship mapping.
package knowledge

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// KnowledgeGraphManager manages entities and relationships in a knowledge graph.
type KnowledgeGraphManager interface {
	// AddEntity adds an entity to the knowledge graph
	AddEntity(ctx context.Context, entity *Entity) error
	
	// AddRelationship adds a relationship between entities
	AddRelationship(ctx context.Context, rel *Relationship) error
	
	// QueryGraph queries the knowledge graph
	QueryGraph(ctx context.Context, query GraphQuery) (*GraphResults, error)
	
	// FindPath finds the shortest path between two entities
	FindPath(ctx context.Context, fromID, toID string) ([]*Relationship, error)
	
	// GetNeighbors gets all entities connected to an entity
	GetNeighbors(ctx context.Context, entityID string, depth int) ([]*Entity, error)
}

// Entity represents a node in the knowledge graph.
type Entity struct {
	ID         string            `json:\"id\"`
	Type       string            `json:\"type\"` // person, organization, concept, etc.
	Name       string            `json:\"name\"`
	Properties map[string]string `json:\"properties,omitempty\"`
	CreatedAt  time.Time         `json:\"createdAt\"`
}

// Relationship represents an edge in the knowledge graph.
type Relationship struct {
	ID         string            `json:\"id\"`
	FromID     string            `json:\"fromID\"`
	ToID       string            `json:\"toID\"`
	Type       string            `json:\"type\"` // works_for, located_in, relates_to, etc.
	Properties map[string]string `json:\"properties,omitempty\"`
	Weight     float64           `json:\"weight\"` // Relationship strength
	CreatedAt  time.Time         `json:\"createdAt\"`
}

// GraphQuery defines a knowledge graph query.
type GraphQuery struct {
	EntityType       string   `json:\"entityType,omitempty\"`
	RelationshipType string   `json:\"relationshipType,omitempty\"`
	Properties       map[string]string `json:\"properties,omitempty\"`
	MaxDepth         int      `json:\"maxDepth\"`
	MaxResults       int      `json:\"maxResults\"`
}

// GraphResults contains query results.
type GraphResults struct {
	Entities      []*Entity      `json:\"entities\"`
	Relationships []*Relationship `json:\"relationships\"`
}

// Implementation

type knowledgeGraphManager struct {
	entities      map[string]*Entity
	relationships map[string]*Relationship
	adjacency     map[string][]string // entityID -> []relationshipIDs
	mu            sync.RWMutex
}

// NewKnowledgeGraphManager creates a new knowledge graph manager.
func NewKnowledgeGraphManager() KnowledgeGraphManager {
	return &knowledgeGraphManager{
		entities:      make(map[string]*Entity),
		relationships: make(map[string]*Relationship),
		adjacency:     make(map[string][]string),
	}
}

func (m *knowledgeGraphManager) AddEntity(ctx context.Context, entity *Entity) error {
	if entity.ID == \"\" {
		return fmt.Errorf(\"entity ID is required\")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	entity.CreatedAt = time.Now()
	m.entities[entity.ID] = entity
	
	return nil
}

func (m *knowledgeGraphManager) AddRelationship(ctx context.Context, rel *Relationship) error {
	if rel.ID == \"\" {
		return fmt.Errorf(\"relationship ID is required\")
	}
	if rel.FromID == \"\" || rel.ToID == \"\" {
		return fmt.Errorf(\"fromID and toID are required\")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Verify entities exist
	if _, exists := m.entities[rel.FromID]; !exists {
		return fmt.Errorf(\"from entity %s not found\", rel.FromID)
	}
	if _, exists := m.entities[rel.ToID]; !exists {
		return fmt.Errorf(\"to entity %s not found\", rel.ToID)
	}
	
	rel.CreatedAt = time.Now()
	m.relationships[rel.ID] = rel
	
	// Update adjacency list
	m.adjacency[rel.FromID] = append(m.adjacency[rel.FromID], rel.ID)
	
	return nil
}

func (m *knowledgeGraphManager) QueryGraph(ctx context.Context, query GraphQuery) (*GraphResults, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := &GraphResults{
		Entities:      []*Entity{},
		Relationships: []*Relationship{},
	}
	
	// Simple query implementation
	for _, entity := range m.entities {
		if query.EntityType != \"\" && entity.Type != query.EntityType {
			continue
		}
		results.Entities = append(results.Entities, entity)
	}
	
	if query.MaxResults > 0 && len(results.Entities) > query.MaxResults {
		results.Entities = results.Entities[:query.MaxResults]
	}
	
	return results, nil
}

func (m *knowledgeGraphManager) FindPath(ctx context.Context, fromID, toID string) ([]*Relationship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// BFS to find shortest path
	queue := [][]string{{fromID}}
	visited := make(map[string]bool)
	
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		
		current := path[len(path)-1]
		
		if current == toID {
			// Found path, convert to relationships
			var rels []*Relationship
			for i := 0; i < len(path)-1; i++ {
				for _, relID := range m.adjacency[path[i]] {
					rel := m.relationships[relID]
					if rel.ToID == path[i+1] {
						rels = append(rels, rel)
						break
					}
				}
			}
			return rels, nil
		}
		
		if visited[current] {
			continue
		}
		visited[current] = true
		
		// Add neighbors to queue
		for _, relID := range m.adjacency[current] {
			rel := m.relationships[relID]
			newPath := append([]string{}, path...)
			newPath = append(newPath, rel.ToID)
			queue = append(queue, newPath)
		}
	}
	
	return nil, fmt.Errorf(\"no path found between %s and %s\", fromID, toID)
}

func (m *knowledgeGraphManager) GetNeighbors(ctx context.Context, entityID string, depth int) ([]*Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if depth <= 0 {
		depth = 1
	}
	
	neighbors := make(map[string]*Entity)
	m.getNeighborsRecursive(entityID, depth, neighbors)
	
	result := make([]*Entity, 0, len(neighbors))
	for _, entity := range neighbors {
		result = append(result, entity)
	}
	
	return result, nil
}

func (m *knowledgeGraphManager) getNeighborsRecursive(entityID string, depth int, visited map[string]*Entity) {
	if depth <= 0 {
		return
	}
	
	for _, relID := range m.adjacency[entityID] {
		rel := m.relationships[relID]
		if _, seen := visited[rel.ToID]; !seen {
			visited[rel.ToID] = m.entities[rel.ToID]
			m.getNeighborsRecursive(rel.ToID, depth-1, visited)
		}
	}
}
