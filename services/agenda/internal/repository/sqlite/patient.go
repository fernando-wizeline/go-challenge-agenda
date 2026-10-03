package sqlite

import (
	"context"
	"fmt"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/repository/models"

	"gorm.io/gorm"
)

type PatientRepository struct {
	db *gorm.DB
}

func NewPatientRepository(db *gorm.DB) *PatientRepository {
	return &PatientRepository{db: db}
}

func (r *PatientRepository) GetPatient(ctx context.Context, id string) (*domain.Patient, error) {
	var m models.Patient
	res := r.db.WithContext(ctx).First(&m, "id = ?", id)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("%w: %s", domain.ErrPatientNotFound, id)
	}
	return models.PatientFromModel(&m), res.Error
}

func (r *PatientRepository) GetPatientByPhone(ctx context.Context, phone string) (*domain.Patient, error) {
	var m models.Patient
	res := r.db.WithContext(ctx).Where("phone = ?", phone).First(&m)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return models.PatientFromModel(&m), res.Error
}

func (r *PatientRepository) CreatePatient(ctx context.Context, p *domain.Patient) error {
	m := models.PatientToModel(p)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *PatientRepository) ListPatients(ctx context.Context) ([]*domain.Patient, error) {
	var ms []models.Patient
	if err := r.db.WithContext(ctx).Find(&ms).Error; err != nil {
		return nil, err
	}
	patients := make([]*domain.Patient, len(ms))
	for i, m := range ms {
		m := m
		patients[i] = models.PatientFromModel(&m)
	}
	return patients, nil
}

func (r *PatientRepository) UpdatePatient(ctx context.Context, p *domain.Patient) error {
	m := models.PatientToModel(p)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *PatientRepository) DeletePatient(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Patient{}, "id = ?", id).Error
}
