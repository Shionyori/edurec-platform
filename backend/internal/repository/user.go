package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("记录不存在")
	ErrDuplicate = errors.New("记录已存在")
)

// UserRepository 用户数据访问接口，便于服务层测试替换
type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	FindByID(id uint) (*model.User, error)
	Update(user *model.User) error
	IsAdmin(userID uint) (bool, error)
}

// UserRepo 使用 GORM 操作用户相关表
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *model.User) error {
	if err := r.db.Create(user).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.Where("username = ?", username).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("按用户名查询用户失败: %w", err)
	}
	return user, nil
}

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.Where("email = ?", email).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("按邮箱查询用户失败: %w", err)
	}
	return user, nil
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	user := &model.User{}
	err := r.db.First(user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("按 ID 查询用户失败: %w", err)
	}
	return user, nil
}

func (r *UserRepo) Update(user *model.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

func (r *UserRepo) IsAdmin(userID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Admin{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("查询管理员状态失败: %w", err)
	}
	return count > 0, nil
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
