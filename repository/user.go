package repository

import (
	"github.com/denisyao1/welsh-academy-api/exception"
	"github.com/denisyao1/welsh-academy-api/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	IsNotCreated(user model.User) (bool, error)
	Create(user *model.User) error
	GetByUsername(user *model.User) error
	GetByID(user *model.User) error
	UpdatePassword(user *model.User) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r userRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r userRepo) IsNotCreated(user model.User) (bool, error) {
	var userB model.User
	result := r.db.Where("username=?", user.Username).Find(&userB)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 0, nil
}

func (r userRepo) GetByUsername(user *model.User) error {
	err := r.db.Where("username=?", user.Username).First(user).Error
	if err == gorm.ErrRecordNotFound {
		return exception.ErrRecordNotFound
	}
	return err
}

func (r userRepo) GetByID(user *model.User) error {
	err := r.db.Where("id=?", user.ID).First(user).Error
	if err == gorm.ErrRecordNotFound {
		return exception.ErrRecordNotFound
	}
	return err
}

func (r userRepo) UpdatePassword(user *model.User) error {
	err := r.db.Model(user).Update("password", user.Password).Error
	if err == gorm.ErrRecordNotFound {
		return exception.ErrRecordNotFound
	}
	return err
}
