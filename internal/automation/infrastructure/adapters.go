package infrastructure

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/google/uuid"
	automationdomain "github.com/sasrgita/crm-juridico/internal/automation/domain"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	notifapp "github.com/sasrgita/crm-juridico/internal/notification/application"
	notifdomain "github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// --- LeadMoverAdapter ---

// LeadMoverAdapter wraps funnel's MoveLeadUseCase to implement domain.LeadMover.
type LeadMoverAdapter struct {
	moveLeadUC *funnelapp.MoveLeadUseCase
}

func NewLeadMoverAdapter(uc *funnelapp.MoveLeadUseCase) *LeadMoverAdapter {
	return &LeadMoverAdapter{moveLeadUC: uc}
}

func (a *LeadMoverAdapter) MoveLead(ctx context.Context, tenantID, leadID, columnID, funnelID string) error {
	return a.moveLeadUC.Execute(ctx, funnelapp.MoveLeadInput{
		TenantID: tenantID,
		LeadID:   leadID,
		ColumnID: columnID,
		FunnelID: funnelID,
	})
}

// --- LeadFinderAdapter ---

// leadRow is a minimal struct for querying expired leads.
type leadRow struct {
	ID string `gorm:"column:id"`
}

// LeadFinderAdapter wraps funnel's LeadRepository to implement domain.LeadFinder.
type LeadFinderAdapter struct {
	leadRepo funneldomain.LeadRepository
	db       *gorm.DB
}

func NewLeadFinderAdapter(leadRepo funneldomain.LeadRepository, db *gorm.DB) *LeadFinderAdapter {
	return &LeadFinderAdapter{leadRepo: leadRepo, db: db}
}

func (a *LeadFinderAdapter) FindByID(ctx context.Context, leadID string) (*automationdomain.LeadInfo, error) {
	lead, err := a.leadRepo.FindByID(ctx, leadID)
	if err != nil {
		return nil, err
	}
	return &automationdomain.LeadInfo{
		ID:                lead.ID,
		TenantID:          lead.TenantID,
		FunnelID:          lead.FunnelID,
		ColumnID:          lead.ColumnID,
		ContactID:         lead.ContactID,
		ConversationID:    lead.ConversationID,
		ProductID:         lead.ProductID,
		ResponsibleUserID: lead.ResponsibleUserID,
		ColumnEnteredAt:   lead.ColumnEnteredAt,
	}, nil
}

func (a *LeadFinderAdapter) FindExpiredInColumn(ctx context.Context, columnID string, maxAge time.Duration) ([]string, error) {
	cutoff := time.Now().Add(-maxAge)

	var rows []leadRow
	err := a.db.WithContext(ctx).
		Table("leads").
		Select("id").
		Where("column_id = ? AND column_entered_at < ? AND status = 'open'", columnID, cutoff).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, nil
}

// --- NoteSaverAdapter ---

// NoteSaverAdapter wraps funnel's LeadNoteRepository to implement domain.NoteSaver.
type NoteSaverAdapter struct {
	noteRepo funneldomain.LeadNoteRepository
}

func NewNoteSaverAdapter(noteRepo funneldomain.LeadNoteRepository) *NoteSaverAdapter {
	return &NoteSaverAdapter{noteRepo: noteRepo}
}

func (a *NoteSaverAdapter) SaveNote(ctx context.Context, leadID, tenantID, content, createdBy string) error {
	note, err := funneldomain.NewLeadNote(uuid.New().String(), leadID, tenantID, content, createdBy)
	if err != nil {
		return err
	}
	return a.noteRepo.Create(ctx, note)
}

// --- LostColumnFinderAdapter ---

// LostColumnFinderAdapter wraps funnel's ColumnRepository to implement domain.LostColumnFinder.
type LostColumnFinderAdapter struct {
	columnRepo funneldomain.ColumnRepository
}

func NewLostColumnFinderAdapter(columnRepo funneldomain.ColumnRepository) *LostColumnFinderAdapter {
	return &LostColumnFinderAdapter{columnRepo: columnRepo}
}

func (a *LostColumnFinderAdapter) FindLostColumn(ctx context.Context, funnelID string) (string, error) {
	cols, err := a.columnRepo.FindByFunnelID(ctx, funnelID)
	if err != nil {
		return "", err
	}
	for _, c := range cols {
		if c.Type == funneldomain.ColumnTypeLost {
			return c.ID, nil
		}
	}
	return "", automationdomain.ErrAutomationNotFound
}

// --- LeadDeleterAdapter ---

// LeadDeleterAdapter soft-deletes a lead by setting its status to 'lost'.
type LeadDeleterAdapter struct {
	db *gorm.DB
}

func NewLeadDeleterAdapter(db *gorm.DB) *LeadDeleterAdapter {
	return &LeadDeleterAdapter{db: db}
}

func (a *LeadDeleterAdapter) SoftDelete(ctx context.Context, leadID string) error {
	return a.db.WithContext(ctx).
		Table("leads").
		Where("id = ?", leadID).
		Update("status", "lost").Error
}

// --- NotifierAdapter ---

// NotifierAdapter wraps notification's NotifyService to implement domain.Notifier.
type NotifierAdapter struct {
	notifyService *notifapp.NotifyService
}

func NewNotifierAdapter(notifyService *notifapp.NotifyService) *NotifierAdapter {
	return &NotifierAdapter{notifyService: notifyService}
}

func (a *NotifierAdapter) Notify(ctx context.Context, userID, tenantID, title, body string, metadata map[string]string) error {
	return a.notifyService.Notify(ctx, userID, tenantID, notifdomain.TypeAutomationError, title, body, metadata)
}

// --- MessageSenderStub ---

// MessageSenderStub is a placeholder for the WhatsApp message sender.
// Wire it to the real sender when the whatsapp module exposes the needed interface.
type MessageSenderStub struct {
	log *zap.Logger
}

func NewMessageSenderStub(log *zap.Logger) *MessageSenderStub {
	return &MessageSenderStub{log: log}
}

func (s *MessageSenderStub) SendToContact(ctx context.Context, tenantID, contactID, content string) error {
	s.log.Warn("MessageSenderStub: SendToContact not yet wired",
		zap.String("tenant_id", tenantID),
		zap.String("contact_id", contactID),
	)
	return nil
}

// --- SpecialistSwitcherStub ---

// SpecialistSwitcherStub is a placeholder for the specialist switcher.
type SpecialistSwitcherStub struct {
	log *zap.Logger
}

func NewSpecialistSwitcherStub(log *zap.Logger) *SpecialistSwitcherStub {
	return &SpecialistSwitcherStub{log: log}
}

func (s *SpecialistSwitcherStub) SwitchSpecialist(ctx context.Context, conversationID, specialistID string) error {
	s.log.Warn("SpecialistSwitcherStub: SwitchSpecialist not yet wired",
		zap.String("conversation_id", conversationID),
		zap.String("specialist_id", specialistID),
	)
	return nil
}

// --- ProductRouterStub ---

// ProductRouterStub is a placeholder for the product router.
type ProductRouterStub struct {
	log *zap.Logger
}

func NewProductRouterStub(log *zap.Logger) *ProductRouterStub {
	return &ProductRouterStub{log: log}
}

func (s *ProductRouterStub) FindFunnelForProduct(ctx context.Context, productID string) (string, string, error) {
	s.log.Warn("ProductRouterStub: FindFunnelForProduct not yet wired",
		zap.String("product_id", productID),
	)
	return "", "", nil
}

// --- SpecialistForProductStub ---

// SpecialistForProductStub is a placeholder for the specialist-for-product resolver.
type SpecialistForProductStub struct {
	log *zap.Logger
}

func NewSpecialistForProductStub(log *zap.Logger) *SpecialistForProductStub {
	return &SpecialistForProductStub{log: log}
}

func (s *SpecialistForProductStub) FindByProductID(ctx context.Context, productID string) (string, error) {
	s.log.Warn("SpecialistForProductStub: FindByProductID not yet wired",
		zap.String("product_id", productID),
	)
	return "", nil
}
