package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"context"
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type CampaignEmail struct {
	ID         *uint64
	CampaignID *uint64
	EmailID    *uint64
	Subject    *string
	Ratio      *uint64
}

type CampaignEmailFilter struct {
	Conditions []*Condition
	Pagination *Pagination
}

func (m *CampaignEmail) TableName() string {
	return "campaign_email_tab"
}

var (
	ErrCampaignEmailNotFound = errutil.NotFoundError(errors.New("campaign email not found"))
)

type CampaignEmailRepo interface {
	Get(ctx context.Context, f *CampaignEmailFilter) (*entity.CampaignEmail, error)
	Update(ctx context.Context, f *CampaignEmailFilter, campaignEmail *entity.CampaignEmail) error
	GetMany(ctx context.Context, f *CampaignEmailFilter) ([]*entity.CampaignEmail, error)
	Close(ctx context.Context) error
}

type campaignEmailRepo struct {
	orm *gorm.DB
}

func NewCampaignEmailRepo(_ context.Context, mysqlCfg config.MySQL) (CampaignEmailRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &campaignEmailRepo{orm: orm}, nil
}

func (r *campaignEmailRepo) Get(_ context.Context, f *CampaignEmailFilter) (*entity.CampaignEmail, error) {
	cond, args := ToSqlWithArgs(f.Conditions)

	campaignEmail := new(CampaignEmail)
	if err := r.orm.Where(cond, args...).First(campaignEmail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignEmailNotFound
		}
	}

	return ToCampaignEmail(campaignEmail), nil
}

func (r *campaignEmailRepo) Update(_ context.Context, f *CampaignEmailFilter, campaignEmail *entity.CampaignEmail) error {
	cond, args := ToSqlWithArgs(f.Conditions)
	campaignEmailModel := ToCampaignEmailModel(campaignEmail)
	return r.orm.Model(campaignEmailModel).Where(cond, args...).Updates(ToCampaignEmailModel(campaignEmail)).Error
}

func (r *campaignEmailRepo) GetMany(_ context.Context, f *CampaignEmailFilter) ([]*entity.CampaignEmail, error) {
	cond, args := ToSqlWithArgs(f.Conditions)
	mCampaignEmails := make([]*CampaignEmail, 0)
	if err := r.orm.Where(cond, args...).Find(&mCampaignEmails).Error; err != nil {
		return nil, err
	}

	campaignEmails := make([]*entity.CampaignEmail, len(mCampaignEmails))
	for i, mCampaignEmail := range mCampaignEmails {
		campaignEmails[i] = ToCampaignEmail(mCampaignEmail)
	}

	return campaignEmails, nil
}

func (r *campaignEmailRepo) Close(_ context.Context) error {
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

func ToCampaignEmail(campaignEmail *CampaignEmail) *entity.CampaignEmail {
	return &entity.CampaignEmail{
		ID:         campaignEmail.ID,
		CampaignID: campaignEmail.CampaignID,
		EmailID:    campaignEmail.EmailID,
		Subject:    campaignEmail.Subject,
		Ratio:      campaignEmail.Ratio,
	}
}

func ToCampaignEmailModel(campaignEmail *entity.CampaignEmail) *CampaignEmail {
	return &CampaignEmail{
		CampaignID: campaignEmail.CampaignID,
		EmailID:    campaignEmail.EmailID,
		Subject:    campaignEmail.Subject,
		Ratio:      campaignEmail.Ratio,
	}
}
