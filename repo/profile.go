package repo

type ProfileRepo interface {
	ToTagRepo() TagRepo
}

type profileRepo struct {
	tagRepo TagRepo
}

func NewProfileRepo(tagRepo TagRepo) ProfileRepo {
	return &profileRepo{
		tagRepo: tagRepo,
	}
}

func (r *profileRepo) ToTagRepo() TagRepo {
	return r.tagRepo
}
