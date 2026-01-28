// Package finance provides trade surveillance for market manipulation detection
package finance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TradeSurveillance monitors trading activity for suspicious patterns
type TradeSurveillance interface {
	MonitorTrade(ctx context.Context, trade *Trade) ([]*Alert, error)
	AnalyzePattern(ctx context.Context, symbol string, window time.Duration) (*PatternAnalysis, error)
	GenerateCase(ctx context.Context, alert *Alert) (*RegulatoryCase, error)
	GetAlerts(ctx context.Context, filters AlertFilters) ([]*Alert, error)
}

type Trade struct {
	ID        string
	Timestamp time.Time
	Symbol    string
	Side      string // "buy" or "sell"
	Price     float64
	Volume    int64
	Account   string
	UserID    string
	OrderID   string
}

type Alert struct {
	ID          string
	Timestamp   time.Time
	Pattern     PatternType
	Severity    AlertSeverity
	Symbol      string
	Trades      []string
	Description string
	Confidence  float64
}

type PatternType string

const (
	PatternWashTrading      PatternType = "wash_trading"
	PatternSpoofing         PatternType = "spoofing"
	PatternLayering         PatternType = "layering"
	PatternFrontRunning     PatternType = "front_running"
	PatternPumpAndDump      PatternType = "pump_and_dump"
	PatternInsiderTrading   PatternType = "insider_trading"
	PatternMarketManip      PatternType = "market_manipulation"
	PatternQuoteStuffing    PatternType = "quote_stuffing"
	PatternMarkingTheClose  PatternType = "marking_the_close"
	PatternMomentumIgnition PatternType = "momentum_ignition"
)

type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

type PatternAnalysis struct {
	Symbol             string
	TimeWindow         time.Duration
	TotalTrades        int
	SuspiciousPatterns map[PatternType]int
	RiskScore          float64
	Recommendations    []string
}

type RegulatoryCase struct {
	ID          string
	CreatedAt   time.Time
	Alert       *Alert
	Status      string
	Assignee    string
	Evidence    []string
	Report      string
	Submitted   bool
	Regulator   string // "SEC", "FINRA", "FCA", etc.
}

type AlertFilters struct {
	StartDate time.Time
	EndDate   time.Time
	Symbol    string
	Pattern   PatternType
	Severity  AlertSeverity
}

// tradeSurveillanceManager implements TradeSurveillance
type tradeSurveillanceManager struct {
	mu     sync.RWMutex
	trades map[string][]*Trade
	alerts []*Alert
	cases  map[string]*RegulatoryCase
	logger Logger
}

func NewTradeSurveillance(logger Logger) TradeSurveillance {
	return &tradeSurveillanceManager{
		trades: make(map[string][]*Trade),
		alerts: []*Alert{},
		cases:  make(map[string]*RegulatoryCase),
		logger: logger,
	}
}

func (t *tradeSurveillanceManager) MonitorTrade(ctx context.Context, trade *Trade) ([]*Alert, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Store trade
	t.trades[trade.Symbol] = append(t.trades[trade.Symbol], trade)
	
	var alerts []*Alert
	
	// Check for wash trading
	if washAlert := t.detectWashTrading(trade); washAlert != nil {
		alerts = append(alerts, washAlert)
		t.alerts = append(t.alerts, washAlert)
	}
	
	// Check for spoofing
	if spoofAlert := t.detectSpoofing(trade); spoofAlert != nil {
		alerts = append(alerts, spoofAlert)
		t.alerts = append(t.alerts, spoofAlert)
	}
	
	// Check for pump and dump
	if pumpAlert := t.detectPumpAndDump(trade); pumpAlert != nil {
		alerts = append(alerts, pumpAlert)
		t.alerts = append(t.alerts, pumpAlert)
	}
	
	if len(alerts) > 0 {
		t.logger.Warn("Suspicious trading pattern detected",
			"symbol", trade.Symbol,
			"patterns", len(alerts))
	}
	
	return alerts, nil
}

func (t *tradeSurveillanceManager) detectWashTrading(trade *Trade) *Alert {
	// Detect buying and selling same security to create artificial volume
	symbolTrades := t.trades[trade.Symbol]
	if len(symbolTrades) < 2 {
		return nil
	}
	
	// Look for matching buy/sell from same account within short timeframe
	for i := len(symbolTrades) - 1; i >= 0 && i > len(symbolTrades)-10; i-- {
		prev := symbolTrades[i]
		if prev.Account == trade.Account &&
			prev.Side != trade.Side &&
			prev.Volume == trade.Volume &&
			trade.Timestamp.Sub(prev.Timestamp) < 5*time.Minute {
			
			return &Alert{
				ID:          fmt.Sprintf("alert-%d", time.Now().Unix()),
				Timestamp:   time.Now(),
				Pattern:     PatternWashTrading,
				Severity:    SeverityHigh,
				Symbol:      trade.Symbol,
				Trades:      []string{prev.ID, trade.ID},
				Description: "Potential wash trading: matching buy/sell from same account",
				Confidence:  0.85,
			}
		}
	}
	
	return nil
}

func (t *tradeSurveillanceManager) detectSpoofing(trade *Trade) *Alert {
	// Detect placing large orders to manipulate price, then canceling
	// Simplified implementation
	symbolTrades := t.trades[trade.Symbol]
	
	// Check for unusually large orders relative to recent volume
	if len(symbolTrades) > 10 {
		avgVolume := t.calculateAverageVolume(symbolTrades[len(symbolTrades)-10:])
		if float64(trade.Volume) > avgVolume*5 {
			return &Alert{
				ID:          fmt.Sprintf("alert-%d", time.Now().Unix()),
				Timestamp:   time.Now(),
				Pattern:     PatternSpoofing,
				Severity:    SeverityMedium,
				Symbol:      trade.Symbol,
				Trades:      []string{trade.ID},
				Description: "Potential spoofing: unusually large order",
				Confidence:  0.65,
			}
		}
	}
	
	return nil
}

func (t *tradeSurveillanceManager) detectPumpAndDump(trade *Trade) *Alert {
	// Detect artificially inflating price then selling
	symbolTrades := t.trades[trade.Symbol]
	if len(symbolTrades) < 20 {
		return nil
	}
	
	// Look for rapid price increase followed by large sell
	recent := symbolTrades[len(symbolTrades)-20:]
	priceIncrease := t.calculatePriceChange(recent)
	
	if priceIncrease > 0.15 && trade.Side == "sell" && trade.Volume > 10000 {
		return &Alert{
			ID:          fmt.Sprintf("alert-%d", time.Now().Unix()),
			Timestamp:   time.Now(),
			Pattern:     PatternPumpAndDump,
			Severity:    SeverityCritical,
			Symbol:      trade.Symbol,
			Trades:      []string{trade.ID},
			Description: "Potential pump and dump: rapid price increase followed by large sell",
			Confidence:  0.78,
		}
	}
	
	return nil
}

func (t *tradeSurveillanceManager) calculateAverageVolume(trades []*Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	
	total := int64(0)
	for _, trade := range trades {
		total += trade.Volume
	}
	
	return float64(total) / float64(len(trades))
}

func (t *tradeSurveillanceManager) calculatePriceChange(trades []*Trade) float64 {
	if len(trades) < 2 {
		return 0
	}
	
	first := trades[0].Price
	last := trades[len(trades)-1].Price
	
	return (last - first) / first
}

func (t *tradeSurveillanceManager) AnalyzePattern(ctx context.Context, symbol string, window time.Duration) (*PatternAnalysis, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	analysis := &PatternAnalysis{
		Symbol:             symbol,
		TimeWindow:         window,
		SuspiciousPatterns: make(map[PatternType]int),
		Recommendations:    []string{},
	}
	
	cutoff := time.Now().Add(-window)
	symbolTrades := t.trades[symbol]
	
	for _, trade := range symbolTrades {
		if trade.Timestamp.After(cutoff) {
			analysis.TotalTrades++
		}
	}
	
	// Count pattern occurrences
	for _, alert := range t.alerts {
		if alert.Symbol == symbol && alert.Timestamp.After(cutoff) {
			analysis.SuspiciousPatterns[alert.Pattern]++
		}
	}
	
	// Calculate risk score
	analysis.RiskScore = float64(len(analysis.SuspiciousPatterns)) / 10.0 * 100
	
	if analysis.RiskScore > 50 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Increased monitoring recommended",
			"Consider regulatory reporting",
			"Review account permissions")
	}
	
	return analysis, nil
}

func (t *tradeSurveillanceManager) GenerateCase(ctx context.Context, alert *Alert) (*RegulatoryCase, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	caseID := fmt.Sprintf("case-%d", time.Now().Unix())
	
	regulatoryCase := &RegulatoryCase{
		ID:        caseID,
		CreatedAt: time.Now(),
		Alert:     alert,
		Status:    "open",
		Evidence:  []string{},
		Submitted: false,
	}
	
	// Determine appropriate regulator
	switch alert.Pattern {
	case PatternInsiderTrading, PatternMarketManip:
		regulatoryCase.Regulator = "SEC"
	default:
		regulatoryCase.Regulator = "FINRA"
	}
	
	t.cases[caseID] = regulatoryCase
	
	t.logger.Info("Regulatory case generated",
		"caseID", caseID,
		"pattern", alert.Pattern,
		"regulator", regulatoryCase.Regulator)
	
	return regulatoryCase, nil
}

func (t *tradeSurveillanceManager) GetAlerts(ctx context.Context, filters AlertFilters) ([]*Alert, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var result []*Alert
	
	for _, alert := range t.alerts {
		if t.matchesAlertFilters(alert, filters) {
			result = append(result, alert)
		}
	}
	
	return result, nil
}

func (t *tradeSurveillanceManager) matchesAlertFilters(alert *Alert, filters AlertFilters) bool {
	if !filters.StartDate.IsZero() && alert.Timestamp.Before(filters.StartDate) {
		return false
	}
	if !filters.EndDate.IsZero() && alert.Timestamp.After(filters.EndDate) {
		return false
	}
	if filters.Symbol != "" && alert.Symbol != filters.Symbol {
		return false
	}
	if filters.Pattern != "" && alert.Pattern != filters.Pattern {
		return false
	}
	if filters.Severity != "" && alert.Severity != filters.Severity {
		return false
	}
	return true
}
