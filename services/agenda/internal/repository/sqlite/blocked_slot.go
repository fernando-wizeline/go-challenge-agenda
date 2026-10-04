package sqlite

import (
	"context"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/repository/models"

	"gorm.io/gorm"
)

type BlockedSlotRepository struct {
	db *gorm.DB
}

func NewBlockedSlotRepository(db *gorm.DB) *BlockedSlotRepository {
	return &BlockedSlotRepository{db: db}
}

func (r *BlockedSlotRepository) CreateBlockedSlot(ctx context.Context, b *domain.BlockedSlot) error {
	m := models.BlockedSlotToModel(b)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *BlockedSlotRepository) GetBlockedSlot(ctx context.Context, id string) (*domain.BlockedSlot, error) {
	var m models.BlockedSlot
	res := r.db.WithContext(ctx).First(&m, "id = ?", id)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return models.BlockedSlotFromModel(&m), nil
}

// ListBlockedSlots returns candidate slots for [from, to]. Recurring slots are
// included whenever their series could reach the window; callers must expand
// them with BlockedSlot.Occurrences, which does the exact filtering.
func (r *BlockedSlotRepository) ListBlockedSlots(ctx context.Context, doctorID string, from, to time.Time) ([]*domain.BlockedSlot, error) {
	var ms []models.BlockedSlot
	err := r.db.WithContext(ctx).
		Where(`doctor_id = ? AND starts_at <= ? AND (
			(recurrence_type = 0 AND ends_at >= ?)
			OR (recurrence_type != 0 AND (recurrence_until IS NULL OR recurrence_until >= ?))
		)`, doctorID, to.UTC(), from.UTC(), from.UTC()).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	slots := make([]*domain.BlockedSlot, len(ms))
	for i, m := range ms {
		m := m
		slots[i] = models.BlockedSlotFromModel(&m)
	}
	return slots, nil
}

func (r *BlockedSlotRepository) DeleteBlockedSlot(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.BlockedSlot{}, "id = ?", id).Error
}
