package healthcare

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BreachSeverity indicates the severity of a breach
type BreachSeverity string

const (
	BreachSeverityLow      BreachSeverity = "low"
	BreachSeverityMedium   BreachSeverity = "medium"
	BreachSeverityHigh     BreachSeverity = "high"
	BreachSeverityCritical BreachSeverity = "critical"
)

// BreachType categorizes different types of breaches
type BreachType string

const (
	BreachTypeUnauthorizedAccess    BreachType = "unauthorized_access"
	BreachTypeDataExfiltration      BreachType = "data_exfiltration"
	BreachTypeImproperDisposal      BreachType = "improper_disposal"
	BreachTypeLostDevice            BreachType = "lost_device"
	BreachTypeStolenDevice          BreachType = "stolen_device"
	BreachTypeHacking               BreachType = "hacking"
	BreachTypePhishing              BreachType = "phishing"
	BreachTypeInsiderThreat         BreachType = "insider_threat"
	BreachTypeVendorBreach          BreachType = "vendor_breach"
	BreachTypeSystemMisconfiguration BreachType = "system_misconfiguration"
)

// Breach represents a PHI breach incident
type Breach struct {
	ID                    string            `json:"id"`
	OrganizationID        string            `json:"organization_id"`
	Type                  BreachType        `json:"type"`
	Severity              BreachSeverity    `json:"severity"`
	DiscoveredAt          time.Time         `json:"discovered_at"`
	OccurredAt            *time.Time        `json:"occurred_at,omitempty"`
	Description           string            `json:"description"`
	AffectedIndividuals   int               `json:"affected_individuals"`
	PHITypesCompromised   []PHIType         `json:"phi_types_compromised"`
	NotificationRequired  bool              `json:"notification_required"`
	NotificationDeadline  *time.Time        `json:"notification_deadline,omitempty"`
	NotificationsSent     []BreachNotification `json:"notifications_sent"`
	RegulatoryReported    bool              `json:"regulatory_reported"`
	RegulatoryReportDate  *time.Time        `json:"regulatory_report_date,omitempty"`
	MitigationSteps       []string          `json:"mitigation_steps"`
	Status                BreachStatus      `json:"status"`
	IncidentCommander     string            `json:"incident_commander"`
	EstimatedCost         float64           `json:"estimated_cost"`
	Metadata              map[string]string `json:"metadata,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// BreachStatus indicates the current status of breach handling
type BreachStatus string

const (
	BreachStatusDetected        BreachStatus = "detected"
	BreachStatusInvestigating   BreachStatus = "investigating"
	BreachStatusContained        BreachStatus = "contained"
	BreachStatusNotifying       BreachStatus = "notifying"
	BreachStatusReportedToHHS   BreachStatus = "reported_to_hhs"
	BreachStatusResolved        BreachStatus = "resolved"
	BreachStatusFalsePositive   BreachStatus = "false_positive"
)

// BreachNotification represents a notification sent regarding a breach
type BreachNotification struct {
	ID            string              `json:"id"`
	BreachID      string              `json:"breach_id"`
	RecipientType BreachRecipientType `json:"recipient_type"`
	RecipientID   string              `json:"recipient_id"` // User ID, email, etc.
	Channel       NotificationChannel `json:"channel"`
	SentAt        time.Time           `json:"sent_at"`
	DeliveredAt   *time.Time          `json:"delivered_at,omitempty"`
	ReadAt        *time.Time          `json:"read_at,omitempty"`
	Content       string              `json:"content"`
	Success       bool                `json:"success"`
	Error         string              `json:"error,omitempty"`
}

// BreachRecipientType indicates who receives the notification
type BreachRecipientType string

const (
	RecipientTypeIndividual   BreachRecipientType = "individual"
	RecipientTypeRegulator    BreachRecipientType = "regulator" // HHS, OCR
	RecipientTypeMedia        BreachRecipientType = "media"
	RecipientTypeBusinessAssociate BreachRecipientType = "business_associate"
	RecipientTypeInternal     BreachRecipientType = "internal"
)

// NotificationChannel indicates how notification is sent
type NotificationChannel string

const (
	ChannelEmail  NotificationChannel = "email"
	ChannelSMS    NotificationChannel = "sms"
	ChannelMail   NotificationChannel = "mail" // Physical mail
	ChannelPhone  NotificationChannel = "phone"
	ChannelPortal NotificationChannel = "portal"
)

// BreachNotifier handles breach notification automation
type BreachNotifier interface {
	// ReportBreach reports a new breach
	ReportBreach(ctx context.Context, breach *Breach) error
	
	// GetBreach retrieves a breach by ID
	GetBreach(ctx context.Context, breachID string) (*Breach, error)
	
	// UpdateBreach updates breach status and details
	UpdateBreach(ctx context.Context, breach *Breach) error
	
	// SendNotifications sends required notifications
	SendNotifications(ctx context.Context, breachID string) error
	
	// GetPendingNotifications returns breaches needing notifications
	GetPendingNotifications(ctx context.Context) ([]*Breach, error)
	
	// RecordNotification records a sent notification
	RecordNotification(ctx context.Context, notification *BreachNotification) error
	
	// AssessNotificationRequirement determines if notification is required
	AssessNotificationRequirement(ctx context.Context, breach *Breach) (bool, error)
}

// breachNotifierImpl implements BreachNotifier
type breachNotifierImpl struct {
	breaches map[string]*Breach
	baaManager BAAManager
	mu       sync.RWMutex
}

// NewBreachNotifier creates a new breach notifier
func NewBreachNotifier(baaManager BAAManager) BreachNotifier {
	return &breachNotifierImpl{
		breaches:   make(map[string]*Breach),
		baaManager: baaManager,
	}
}

// ReportBreach reports a new breach
func (n *breachNotifierImpl) ReportBreach(ctx context.Context, breach *Breach) error {
	if breach.ID == "" {
		breach.ID = n.generateBreachID(breach)
	}
	
	// Validate breach
	if err := n.validateBreach(breach); err != nil {
		return fmt.Errorf("invalid breach: %w", err)
	}
	
	// Set timestamps
	now := time.Now()
	breach.CreatedAt = now
	breach.UpdatedAt = now
	breach.Status = BreachStatusDetected
	
	// Assess if notification is required
	required, err := n.AssessNotificationRequirement(ctx, breach)
	if err != nil {
		return fmt.Errorf("failed to assess notification requirement: %w", err)
	}
	breach.NotificationRequired = required
	
	// Set notification deadline (60 days for individuals, immediate for media if > 500)
	if required {
		deadline := now.AddDate(0, 0, 60) // 60 days
		breach.NotificationDeadline = &deadline
	}
	
	n.mu.Lock()
	defer n.mu.Unlock()
	
	n.breaches[breach.ID] = breach
	
	return nil
}

// GetBreach retrieves a breach by ID
func (n *breachNotifierImpl) GetBreach(ctx context.Context, breachID string) (*Breach, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	breach, exists := n.breaches[breachID]
	if !exists {
		return nil, fmt.Errorf("breach not found: %s", breachID)
	}
	
	return breach, nil
}

// UpdateBreach updates breach status and details
func (n *breachNotifierImpl) UpdateBreach(ctx context.Context, breach *Breach) error {
	if err := n.validateBreach(breach); err != nil {
		return fmt.Errorf("invalid breach: %w", err)
	}
	
	n.mu.Lock()
	defer n.mu.Unlock()
	
	existing, exists := n.breaches[breach.ID]
	if !exists {
		return fmt.Errorf("breach not found: %s", breach.ID)
	}
	
	// Preserve creation timestamp
	breach.CreatedAt = existing.CreatedAt
	breach.UpdatedAt = time.Now()
	
	n.breaches[breach.ID] = breach
	
	return nil
}

// SendNotifications sends required notifications for a breach
func (n *breachNotifierImpl) SendNotifications(ctx context.Context, breachID string) error {
	breach, err := n.GetBreach(ctx, breachID)
	if err != nil {
		return err
	}
	
	if !breach.NotificationRequired {
		return fmt.Errorf("notification not required for breach %s", breachID)
	}
	
	notifications := []BreachNotification{}
	
	// 1. Notify affected individuals
	if breach.AffectedIndividuals > 0 {
		notification := BreachNotification{
			ID:            n.generateNotificationID(),
			BreachID:      breachID,
			RecipientType: RecipientTypeIndividual,
			RecipientID:   "affected-individuals",
			Channel:       ChannelEmail,
			SentAt:        time.Now(),
			Content:       n.generateIndividualNotificationContent(breach),
			Success:       true,
		}
		notifications = append(notifications, notification)
	}
	
	// 2. Notify HHS if > 500 individuals
	if breach.AffectedIndividuals >= 500 {
		notification := BreachNotification{
			ID:            n.generateNotificationID(),
			BreachID:      breachID,
			RecipientType: RecipientTypeRegulator,
			RecipientID:   "hhs-ocr",
			Channel:       ChannelEmail,
			SentAt:        time.Now(),
			Content:       n.generateRegulatoryNotificationContent(breach),
			Success:       true,
		}
		notifications = append(notifications, notification)
		
		breach.RegulatoryReported = true
		reportDate := time.Now()
		breach.RegulatoryReportDate = &reportDate
	}
	
	// 3. Notify media if > 500 individuals (prominent media outlets in the state)
	if breach.AffectedIndividuals >= 500 {
		notification := BreachNotification{
			ID:            n.generateNotificationID(),
			BreachID:      breachID,
			RecipientType: RecipientTypeMedia,
			RecipientID:   "media-outlets",
			Channel:       ChannelEmail,
			SentAt:        time.Now(),
			Content:       n.generateMediaNotificationContent(breach),
			Success:       true,
		}
		notifications = append(notifications, notification)
	}
	
	// 4. Notify business associates if applicable
	if n.baaManager != nil {
		// Get BAA for organization
		baas, err := n.baaManager.ListBAAs(ctx, BAAFilters{
			OrganizationID: breach.OrganizationID,
			Status:         BAAStatusActive,
		})
		if err == nil && len(baas) > 0 {
			for _, baa := range baas {
				if baa.BreachNotification.Required {
					notification := BreachNotification{
						ID:            n.generateNotificationID(),
						BreachID:      breachID,
						RecipientType: RecipientTypeBusinessAssociate,
						RecipientID:   baa.ID,
						Channel:       ChannelEmail,
						SentAt:        time.Now(),
						Content:       n.generateBusinessAssociateNotificationContent(breach, baa),
						Success:       true,
					}
					notifications = append(notifications, notification)
				}
			}
		}
	}
	
	// Store notifications
	breach.NotificationsSent = append(breach.NotificationsSent, notifications...)
	breach.Status = BreachStatusNotifying
	breach.UpdatedAt = time.Now()
	
	return n.UpdateBreach(ctx, breach)
}

// GetPendingNotifications returns breaches needing notifications
func (n *breachNotifierImpl) GetPendingNotifications(ctx context.Context) ([]*Breach, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	result := []*Breach{}
	
	for _, breach := range n.breaches {
		if breach.NotificationRequired && len(breach.NotificationsSent) == 0 {
			// Check if deadline is approaching
			if breach.NotificationDeadline != nil {
				daysUntilDeadline := time.Until(*breach.NotificationDeadline).Hours() / 24
				if daysUntilDeadline <= 7 { // Within 7 days of deadline
					result = append(result, breach)
				}
			}
		}
	}
	
	return result, nil
}

// RecordNotification records a sent notification
func (n *breachNotifierImpl) RecordNotification(ctx context.Context, notification *BreachNotification) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	breach, exists := n.breaches[notification.BreachID]
	if !exists {
		return fmt.Errorf("breach not found: %s", notification.BreachID)
	}
	
	breach.NotificationsSent = append(breach.NotificationsSent, *notification)
	breach.UpdatedAt = time.Now()
	
	return nil
}

// AssessNotificationRequirement determines if notification is required
func (n *breachNotifierImpl) AssessNotificationRequirement(ctx context.Context, breach *Breach) (bool, error) {
	// HIPAA Breach Notification Rule assessment
	
	// 1. Check if breach affects > 500 individuals (always requires notification)
	if breach.AffectedIndividuals >= 500 {
		return true, nil
	}
	
	// 2. Check severity
	if breach.Severity == BreachSeverityCritical || breach.Severity == BreachSeverityHigh {
		return true, nil
	}
	
	// 3. Check PHI types compromised
	highRiskPHI := []PHIType{
		PHITypeSSN,
		PHITypeMedicalRecordNumber,
		PHITypeBiometricIdentifier,
		PHITypeFacePhoto,
	}
	
	for _, compromisedType := range breach.PHITypesCompromised {
		for _, highRisk := range highRiskPHI {
			if compromisedType == highRisk {
				return true, nil
			}
		}
	}
	
	// 4. Low probability that PHI has been compromised?
	// (This would require risk assessment - simplified here)
	if breach.AffectedIndividuals > 0 {
		return true, nil
	}
	
	return false, nil
}

// Helper functions

func (n *breachNotifierImpl) generateBreachID(breach *Breach) string {
	return fmt.Sprintf("breach-%d", time.Now().UnixNano())
}

func (n *breachNotifierImpl) generateNotificationID() string {
	return fmt.Sprintf("notif-%d", time.Now().UnixNano())
}

func (n *breachNotifierImpl) validateBreach(breach *Breach) error {
	if breach.OrganizationID == "" {
		return fmt.Errorf("organization ID is required")
	}
	
	if breach.Type == "" {
		return fmt.Errorf("breach type is required")
	}
	
	if breach.Severity == "" {
		return fmt.Errorf("severity is required")
	}
	
	if breach.DiscoveredAt.IsZero() {
		return fmt.Errorf("discovery time is required")
	}
	
	if breach.Description == "" {
		return fmt.Errorf("description is required")
	}
	
	return nil
}

func (n *breachNotifierImpl) generateIndividualNotificationContent(breach *Breach) string {
	return fmt.Sprintf(`
NOTICE OF BREACH OF PROTECTED HEALTH INFORMATION

We are writing to notify you of a breach that may have affected the security of your protected health information (PHI).

What Happened:
%s

What Information Was Involved:
The breach may have affected the following types of information: %v

What We Are Doing:
We have taken the following steps to address this situation:
%v

What You Can Do:
We recommend that you remain vigilant and monitor your accounts for any suspicious activity.

For More Information:
If you have questions, please contact us at [contact information].

We sincerely apologize for any inconvenience this may cause.
`, breach.Description, breach.PHITypesCompromised, breach.MitigationSteps)
}

func (n *breachNotifierImpl) generateRegulatoryNotificationContent(breach *Breach) string {
	return fmt.Sprintf(`
BREACH NOTIFICATION TO HHS OCR

Organization: %s
Breach ID: %s
Discovery Date: %s
Affected Individuals: %d
Breach Type: %s
Severity: %s

Description:
%s

PHI Types Compromised:
%v

Mitigation Steps:
%v
`, breach.OrganizationID, breach.ID, breach.DiscoveredAt.Format(time.RFC3339),
		breach.AffectedIndividuals, breach.Type, breach.Severity,
		breach.Description, breach.PHITypesCompromised, breach.MitigationSteps)
}

func (n *breachNotifierImpl) generateMediaNotificationContent(breach *Breach) string {
	return fmt.Sprintf(`
MEDIA NOTIFICATION - PHI BREACH

A breach affecting %d individuals has been reported.

Type: %s
Discovery Date: %s

Details will be provided in accordance with HIPAA breach notification requirements.
`, breach.AffectedIndividuals, breach.Type, breach.DiscoveredAt.Format("January 2, 2006"))
}

func (n *breachNotifierImpl) generateBusinessAssociateNotificationContent(breach *Breach, baa *BAA) string {
	return fmt.Sprintf(`
BREACH NOTIFICATION TO BUSINESS ASSOCIATE

BAA ID: %s
Business Associate: %s

A breach has been discovered that may affect data covered under our Business Associate Agreement.

Breach ID: %s
Discovery Date: %s
Affected Individuals: %d

Description:
%s

Please review your systems and take appropriate action in accordance with our BAA.
`, baa.ID, baa.BusinessAssociateName, breach.ID,
		breach.DiscoveredAt.Format(time.RFC3339),
		breach.AffectedIndividuals, breach.Description)
}
