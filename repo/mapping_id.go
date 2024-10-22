package repo

import (
	"cdp/config"
	"cdp/entity"
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MappingID struct {
	ID   *uint64
	UdID *string
}

func (m *MappingID) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *MappingID) TableName() string {
	return "mapping_id_tab"
}

type MappingIDRepo interface {
	GetMany(ctx context.Context, udIDs []string) ([]*entity.MappingID, error)
	CreateMany(ctx context.Context, mappingIDs []*entity.MappingID) error
	Close(ctx context.Context) error
}

type mappingIDRepo struct {
	orm *gorm.DB
}

func NewMappingIDRepo(_ context.Context, mysqlCfg config.MySQL) (MappingIDRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()))
	if err != nil {
		return nil, err
	}
	return &mappingIDRepo{orm: orm}, nil
}

func (r *mappingIDRepo) GetMany(_ context.Context, udIDs []string) ([]*entity.MappingID, error) {
	mappingIDModels := make([]*MappingID, 0, len(udIDs))

	if err := r.orm.Where("ud_id IN (?)", udIDs).Find(&mappingIDModels).Error; err != nil {
		return nil, err
	}

	mappingIDs := make([]*entity.MappingID, 0, len(mappingIDModels))
	for _, mappingIDModel := range mappingIDModels {
		mappingIDs = append(mappingIDs, ToMappingID(mappingIDModel))
	}

	return mappingIDs, nil
}

func (r *mappingIDRepo) CreateMany(_ context.Context, mappingIDs []*entity.MappingID) error {
	mappingIDModels := make([]*MappingID, 0, len(mappingIDs))
	for _, mappingID := range mappingIDs {
		mappingIDModels = append(mappingIDModels, ToMappingIDModel(mappingID))
	}

	return r.orm.Create(&mappingIDModels).Error
}

func (r *mappingIDRepo) Close(_ context.Context) error {
	if r.orm != nil {
		sqlDB, err := r.orm.DB()
		if err != nil {
			return err
		}

		err = sqlDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func ToMappingIDModel(mappingID *entity.MappingID) *MappingID {
	return &MappingID{
		ID:   mappingID.ID,
		UdID: mappingID.UdID,
	}
}

func ToMappingID(mappingID *MappingID) *entity.MappingID {
	return &entity.MappingID{
		ID:   mappingID.ID,
		UdID: mappingID.UdID,
	}
}
