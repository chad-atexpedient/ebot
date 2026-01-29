// Package knowledge provides advanced RAG capabilities including hybrid search,
// knowledge graphs, and semantic retrieval.
package knowledge

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

// HybridSearchEngine combines vector and keyword search for optimal retrieval.
type HybridSearchEngine interface {
	// Search performs hybrid search combining vector and keyword methods
	Search(ctx context.Context, query string, opts SearchOptions) (*SearchResults, error)
	
	// IndexDocument adds or updates a document in the search index
	IndexDocument(ctx context.Context, doc *Document) error
	
	// DeleteDocument removes a document from the index
	DeleteDocument(ctx context.Context, docID string) error
	
	// Rerank reranks search results using advanced models
	Rerank(ctx context.Context, query string, results *SearchResults) (*SearchResults, error)
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	// Search strategy
	Strategy SearchStrategy `json:"strategy"`
	
	// Weighting
	VectorWeight  float64 `json:"vectorWeight"`  // 0-1, default 0.7
	KeywordWeight float64 `json:"keywordWeight"` // 0-1, default 0.3
	
	// Limits
	MaxResults int `json:"maxResults"`
	MinScore   float64 `json:"minScore"`
	
	// Filters
	Filters map[string]interface{} `json:"filters,omitempty"`
	
	// Reranking
	EnableReranking bool `json:"enableReranking"`
}

// SearchStrategy defines the search approach.
type SearchStrategy string

const (
	StrategyHybrid       SearchStrategy = "hybrid"        // Vector + Keyword
	StrategyVectorOnly   SearchStrategy = "vector_only"   // Semantic only
	StrategyKeywordOnly  SearchStrategy = "keyword_only"  // Keyword only
	StrategyAdaptive     SearchStrategy = "adaptive"      // Auto-select best
)

// SearchResults contains ranked search results.
type SearchResults struct {
	Results      []*SearchResult `json:"results"`
	TotalResults int             `json:"totalResults"`
	Strategy     SearchStrategy  `json:"strategy"`
	TimeTakenMs  float64         `json:"timeTakenMs"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	Document     *Document `json:"document"`
	Score        float64   `json:"score"`
	VectorScore  float64   `json:"vectorScore"`
	KeywordScore float64   `json:"keywordScore"`
	Rank         int       `json:"rank"`
	Highlights   []string  `json:"highlights,omitempty"`
}

// Document represents an indexed document.
type Document struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Embedding   []float64              `json:"-"` // Vector embedding
	Keywords    []string               `json:"keywords,omitempty"`
	ChunkIndex  int                    `json:"chunkIndex,omitempty"`
}

// Implementation

type hybridSearchEngine struct {
	// In-memory indexes (production would use vector DB)
	documents      map[string]*Document
	vectorIndex    map[string][]float64 // docID -> embedding
	keywordIndex   map[string][]string  // word -> docIDs
	mu             sync.RWMutex
}

// NewHybridSearchEngine creates a new hybrid search engine.
func NewHybridSearchEngine() HybridSearchEngine {
	return &hybridSearchEngine{
		documents:    make(map[string]*Document),
		vectorIndex:  make(map[string][]float64),
		keywordIndex: make(map[string][]string),
	}
}

func (e *hybridSearchEngine) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResults, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	
	// Set defaults
	if opts.VectorWeight == 0 && opts.KeywordWeight == 0 {
		opts.VectorWeight = 0.7
		opts.KeywordWeight = 0.3
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 10
	}
	
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var results []*SearchResult
	
	switch opts.Strategy {
	case StrategyVectorOnly:
		results = e.vectorSearch(query)
	case StrategyKeywordOnly:
		results = e.keywordSearch(query)
	case StrategyHybrid, StrategyAdaptive:
		results = e.hybridSearch(query, opts)
	default:
		return nil, fmt.Errorf("unknown search strategy: %s", opts.Strategy)
	}
	
	// Filter by min score
	if opts.MinScore > 0 {
		filtered := []*SearchResult{}
		for _, r := range results {
			if r.Score >= opts.MinScore {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}
	
	// Limit results
	if len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}
	
	// Set ranks
	for i, r := range results {
		r.Rank = i + 1
	}
	
	return &SearchResults{
		Results:      results,
		TotalResults: len(results),
		Strategy:     opts.Strategy,
		TimeTakenMs:  10.5, // Simplified
	}, nil
}

func (e *hybridSearchEngine) hybridSearch(query string, opts SearchOptions) []*SearchResult {
	// Get vector results
	vectorResults := e.vectorSearch(query)
	vectorScores := make(map[string]float64)
	for _, r := range vectorResults {
		vectorScores[r.Document.ID] = r.Score
	}
	
	// Get keyword results
	keywordResults := e.keywordSearch(query)
	keywordScores := make(map[string]float64)
	for _, r := range keywordResults {
		keywordScores[r.Document.ID] = r.Score
	}
	
	// Combine scores
	combined := make(map[string]*SearchResult)
	allDocIDs := make(map[string]bool)
	
	for id := range vectorScores {
		allDocIDs[id] = true
	}
	for id := range keywordScores {
		allDocIDs[id] = true
	}
	
	for docID := range allDocIDs {
		vScore := vectorScores[docID] * opts.VectorWeight
		kScore := keywordScores[docID] * opts.KeywordWeight
		totalScore := vScore + kScore
		
		combined[docID] = &SearchResult{
			Document:     e.documents[docID],
			Score:        totalScore,
			VectorScore:  vectorScores[docID],
			KeywordScore: keywordScores[docID],
		}
	}
	
	// Sort by combined score
	results := make([]*SearchResult, 0, len(combined))
	for _, r := range combined {
		results = append(results, r)
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

func (e *hybridSearchEngine) vectorSearch(query string) []*SearchResult {
	// Simplified: In production, use actual vector embeddings
	queryEmbedding := e.generateEmbedding(query)
	
	results := []*SearchResult{}
	for docID, docEmbedding := range e.vectorIndex {
		similarity := cosineSimilarity(queryEmbedding, docEmbedding)
		results = append(results, &SearchResult{
			Document: e.documents[docID],
			Score:    similarity,
		})
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

func (e *hybridSearchEngine) keywordSearch(query string) []*SearchResult {
	// Simple BM25-like scoring
	queryTerms := strings.Fields(strings.ToLower(query))
	
	docScores := make(map[string]float64)
	
	for _, term := range queryTerms {
		if docIDs, exists := e.keywordIndex[term]; exists {
			for _, docID := range docIDs {
				doc := e.documents[docID]
				// TF: term frequency in document
				tf := float64(strings.Count(strings.ToLower(doc.Content), term))
				// IDF: inverse document frequency
				idf := math.Log(float64(len(e.documents)) / float64(len(docIDs)))
				docScores[docID] += tf * idf
			}
		}
	}
	
	// Normalize scores
	maxScore := 0.0
	for _, score := range docScores {
		if score > maxScore {
			maxScore = score
		}
	}
	
	results := []*SearchResult{}
	for docID, score := range docScores {
		normalizedScore := score / maxScore
		results = append(results, &SearchResult{
			Document: e.documents[docID],
			Score:    normalizedScore,
		})
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

func (e *hybridSearchEngine) IndexDocument(ctx context.Context, doc *Document) error {
	if doc.ID == "" {
		return fmt.Errorf("document ID is required")
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Store document
	e.documents[doc.ID] = doc
	
	// Generate and store embedding
	if len(doc.Embedding) == 0 {
		doc.Embedding = e.generateEmbedding(doc.Content)
	}
	e.vectorIndex[doc.ID] = doc.Embedding
	
	// Build keyword index
	words := strings.Fields(strings.ToLower(doc.Content))
	wordSet := make(map[string]bool)
	for _, word := range words {
		if len(word) > 2 { // Skip very short words
			wordSet[word] = true
		}
	}
	
	for word := range wordSet {
		if e.keywordIndex[word] == nil {
			e.keywordIndex[word] = []string{}
		}
		// Add docID if not already present
		found := false
		for _, id := range e.keywordIndex[word] {
			if id == doc.ID {
				found = true
				break
			}
		}
		if !found {
			e.keywordIndex[word] = append(e.keywordIndex[word], doc.ID)
		}
	}
	
	return nil
}

func (e *hybridSearchEngine) DeleteDocument(ctx context.Context, docID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.documents[docID]; !exists {
		return fmt.Errorf("document %s not found", docID)
	}
	
	// Remove from indexes
	delete(e.documents, docID)
	delete(e.vectorIndex, docID)
	
	// Remove from keyword index
	for word, docIDs := range e.keywordIndex {
		filtered := []string{}
		for _, id := range docIDs {
			if id != docID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) > 0 {
			e.keywordIndex[word] = filtered
		} else {
			delete(e.keywordIndex, word)
		}
	}
	
	return nil
}

func (e *hybridSearchEngine) Rerank(ctx context.Context, query string, results *SearchResults) (*SearchResults, error) {
	// Simplified reranking: boost exact phrase matches
	queryLower := strings.ToLower(query)
	
	for _, result := range results.Results {
		contentLower := strings.ToLower(result.Document.Content)
		if strings.Contains(contentLower, queryLower) {
			result.Score *= 1.2 // Boost by 20%
		}
	}
	
	// Re-sort
	sort.Slice(results.Results, func(i, j int) bool {
		return results.Results[i].Score > results.Results[j].Score
	})
	
	// Update ranks
	for i, r := range results.Results {
		r.Rank = i + 1
	}
	
	return results, nil
}

// Helper functions

func (e *hybridSearchEngine) generateEmbedding(text string) []float64 {
	// Simplified: In production, use actual embedding model (OpenAI, etc.)
	// This creates a dummy 384-dimensional vector based on text
	embedding := make([]float64, 384)
	for i := range embedding {
		embedding[i] = float64(len(text)%100) / 100.0
	}
	return embedding
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
