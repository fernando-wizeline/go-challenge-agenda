package sqlite

import (
	"context"
	"fmt"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/repository/models"

	"gorm.io/gorm"
)

type DoctorRepository struct {
	db *gorm.DB
}

func NewDoctorRepository(db *gorm.DB) *DoctorRepository {
	return &DoctorRepository{db: db}
}

func (r *DoctorRepository) GetDoctor(ctx context.Context, id string) (*domain.Doctor, error) {
	var m models.Doctor
	res := r.db.WithContext(ctx).Preload("WorkingHours").First(&m, "id = ?", id)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("doctor not found: %s", id)
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return models.DoctorFromModel(&m), nil
}

func (r *DoctorRepository) ListDoctors(ctx context.Context) ([]*domain.Doctor, error) {
	var ms []models.Doctor
	if err := r.db.WithContext(ctx).Preload("WorkingHours").Find(&ms).Error; err != nil {
		return nil, err
	}
	doctors := make([]*domain.Doctor, len(ms))
	for i, m := range ms {
		m := m
		doctors[i] = models.DoctorFromModel(&m)
	}
	return doctors, nil
}
