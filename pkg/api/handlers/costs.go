package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chad-atexpedient/ebot/pkg/metering"
)

// CostHandler handles cost-related API requests
type CostHandler struct {
	collector metering.Collector
	reporter  metering.Reporter
}

// NewCostHandler creates a new cost handler
func NewCostHandler(collector metering.Collector, reporter metering.Reporter) *CostHandler {
	return &CostHandler{
		collector: collector,
		reporter:  reporter,
	}
}

// GetReport returns cost report for a time range
func (h *CostHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse time range
	rangeParam := r.URL.Query().Get("range")
	var startTime, endTime time.Time
	endTime = time.Now()
	
	switch rangeParam {
	case "7d":
		startTime = endTime.AddDate(0, 0, -7)
	case "30d":
		startTime = endTime.AddDate(0, 0, -30)
	case "90d":
		startTime = endTime.AddDate(0, 0, -90)
	default:
		startTime = endTime.AddDate(0, 0, -7) // Default to 7 days
	}

	// Check permissions
	userID := getUserIDFromRequest(r)
	workspaceID := r.URL.Query().Get("workspaceId")
	
	if workspaceID != "" && !canAccessWorkspace(r, workspaceID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get report
	filters := metering.ReportFilters{
		StartTime:   startTime,
		EndTime:     endTime,
		UserID:      userID,
		WorkspaceID: workspaceID,
	}

	if isAdmin(r) {
		// Admin can see all costs
		filters.UserID = ""
	}

	report, err := h.reporter.GenerateReport(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetUserCosts returns costs for a specific user (admin or self)
func (h *CostHandler) GetUserCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	targetUserID := r.URL.Query().Get("userId")
	requesterID := getUserIDFromRequest(r)
	
	// Users can see their own costs, admins can see anyone's
	if targetUserID != requesterID && !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse time range
	startTime := time.Now().AddDate(0, 0, -30) // Last 30 days
	endTime := time.Now()

	filters := metering.ReportFilters{
		StartTime: startTime,
		EndTime:   endTime,
		UserID:    targetUserID,
	}

	report, err := h.reporter.GenerateReport(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetWorkspaceCosts returns costs for a specific workspace
func (h *CostHandler) GetWorkspaceCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID == "" {
		http.Error(w, "workspaceId parameter required", http.StatusBadRequest)
		return
	}

	if !canAccessWorkspace(r, workspaceID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse time range
	startTime := time.Now().AddDate(0, 0, -30) // Last 30 days
	endTime := time.Now()

	filters := metering.ReportFilters{
		StartTime:   startTime,
		EndTime:     endTime,
		WorkspaceID: workspaceID,
	}

	report, err := h.reporter.GenerateReport(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetBudgetStatus returns current budget status
func (h *CostHandler) GetBudgetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := getUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get current month's spending
	startTime := time.Now().AddDate(0, 0, -30)
	endTime := time.Now()

	filters := metering.ReportFilters{
		StartTime: startTime,
		EndTime:   endTime,
		UserID:    userID,
	}

	report, err := h.reporter.GenerateReport(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Get budget limit from user settings
	budgetLimit := 1000.0 // Default budget
	percentUsed := (report.TotalCostUSD / budgetLimit) * 100

	response := map[string]interface{}{
		"totalCost":   report.TotalCostUSD,
		"budgetLimit": budgetLimit,
		"percentUsed": percentUsed,
		"remaining":   budgetLimit - report.TotalCostUSD,
		"isOverBudget": report.TotalCostUSD > budgetLimit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExportReport exports cost report as CSV
func (h *CostHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse filters similar to GetReport
	startTime := time.Now().AddDate(0, 0, -30)
	endTime := time.Now()
	userID := getUserIDFromRequest(r)
	
	if isAdmin(r) {
		userID = "" // Admin sees all
	}

	filters := metering.ReportFilters{
		StartTime: startTime,
		EndTime:   endTime,
		UserID:    userID,
	}

	report, err := h.reporter.GenerateReport(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate CSV
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=cost-report.csv")
	
	// Write CSV header
	w.Write([]byte("Date,Provider,Model,Tokens,Cost\n"))
	
	// Write data rows
	for _, item := range report.Breakdown {
		row := []byte{}
		row = append(row, []byte(time.Now().Format("2006-01-02"))...)
		row = append(row, ',')
		row = append(row, []byte(item.ModelProvider)...)
		row = append(row, ',')
		row = append(row, []byte(item.ModelName)...)
		row = append(row, ',')
		// Add token and cost data
		row = append(row, '\n')
		w.Write(row)
	}
}

// Helper functions

func canAccessWorkspace(r *http.Request, workspaceID string) bool {
	// TODO: Check if user has access to workspace
	return true // Placeholder
}
