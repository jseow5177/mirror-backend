package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Email struct {
	ID         *uint64
	Name       *string
	EmailDesc  *string
	Blob       *string
	Status     *uint32
	CreateTime *uint64
	UpdateTime *uint64
}

func (m *Email) TableName() string {
	return "email_tab"
}

func (m *Email) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

type EmailFilter struct {
	ID         *uint64
	Name       *string
	EmailDesc  *string
	Status     *uint32
	Pagination *Pagination `gorm:"-"`
}

func (f *EmailFilter) GetName() string {
	if f != nil && f.Name != nil {
		return *f.Name
	}
	return ""
}

func (f *EmailFilter) GetEmailDesc() string {
	if f != nil && f.EmailDesc != nil {
		return *f.EmailDesc
	}
	return ""
}

type EmailRepo interface {
	GetMany(ctx context.Context, f *EmailFilter) ([]*entity.Email, *entity.Pagination, error)
	Create(ctx context.Context, email *entity.Email) (uint64, error)
	Close(ctx context.Context) error
}

type emailRepo struct {
	orm *gorm.DB
}

func NewEmailRepo(_ context.Context, mysqlCfg config.MySQL) (EmailRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &emailRepo{orm: orm}, nil
}

func (r *emailRepo) Create(_ context.Context, email *entity.Email) (uint64, error) {
	emailModel := ToEmailModel(email)

	if err := r.orm.Create(&emailModel).Error; err != nil {
		return 0, err
	}

	return emailModel.GetID(), nil
}

func (r *emailRepo) GetMany(_ context.Context, f *EmailFilter) ([]*entity.Email, *entity.Pagination, error) {
	var (
		cond string
		args = make([]interface{}, 0)
	)
	if f.Name != nil {
		cond += "LOWER(name) LIKE ?"
		args = append(args, fmt.Sprintf("%%%s%%", f.GetName()))
	}

	if f.EmailDesc != nil {
		if cond != "" {
			cond += " OR "
		}
		cond += "LOWER(email_desc) LIKE ?"
		args = append(args, fmt.Sprintf("%%%s%%", f.GetEmailDesc()))
	}

	if cond != "" {
		cond += " AND "
	}
	cond += "status != ?"
	args = append(args, entity.EmailStatusDeleted)

	var count int64
	if err := r.orm.Model(&Email{}).Where(cond, args...).Count(&count).Error; err != nil {
		return nil, nil, err
	}

	var (
		limit = f.Pagination.GetLimit()
		page  = f.Pagination.GetPage()
	)
	if page == 0 {
		page = 1
	}

	var (
		offset  = (page - 1) * limit
		mEmails = make([]*Email, 0)
	)
	query := r.orm.Where(cond, args...).Offset(int(offset))
	if limit > 0 {
		query = query.Limit(int(limit + 1))
	}

	if err := query.Find(&mEmails).Error; err != nil {
		return nil, nil, err
	}

	var hasNext bool
	if limit > 0 && len(mEmails) > int(limit) {
		hasNext = true
		mEmails = mEmails[:limit]
	}

	emails := make([]*entity.Email, len(mEmails))
	for i, mEmail := range mEmails {
		emails[i] = ToEmail(mEmail)
	}

	return emails, &entity.Pagination{
		Page:    goutil.Uint32(page),
		Limit:   f.Pagination.Limit, // may be nil
		HasNext: goutil.Bool(hasNext),
		Total:   goutil.Int64(count),
	}, nil
}

func (r *emailRepo) Close(_ context.Context) error {
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

func ToEmailModel(email *entity.Email) *Email {
	return &Email{
		ID:         email.ID,
		Name:       email.Name,
		EmailDesc:  email.EmailDesc,
		Blob:       email.Blob,
		Status:     email.Status,
		CreateTime: email.CreateTime,
		UpdateTime: email.UpdateTime,
	}
}

func ToEmail(email *Email) *entity.Email {
	return &entity.Email{
		ID:         email.ID,
		Name:       email.Name,
		EmailDesc:  email.EmailDesc,
		Blob:       email.Blob,
		Status:     email.Status,
		CreateTime: email.CreateTime,
		UpdateTime: email.UpdateTime,
	}
}
