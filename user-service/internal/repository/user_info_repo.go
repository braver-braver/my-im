package repository

//go:generate mockgen -source=user_info_repo.go -destination=../../internal/mocks/user_info_repo_mock.go  -package mock

import (
	"context"
	"fmt"
	"golang.org/x/sync/singleflight"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"user-service/internal/cache"
	"user-service/internal/model"
)

var (
	_tableUserInfoName        = (&model.UserInfoModel{}).TableName()
	_getUserInfoSQL           = "SELECT * FROM %s WHERE id = ?"
	_getUserInfoByUsernameSQL = "SELECT * FROM %s WHERE username = ?"
	_getUserInfoByEmailSQL    = "SELECT * FROM %s WHERE email = ?"
	_getUserInfoByPhoneSQL    = "SELECT * FROM %s WHERE phone = ?"
	_batchGetUserInfoSQL      = "SELECT * FROM %s WHERE id IN (%s)"
)

var (
	g singleflight.Group
)
var _ UserInfoRepo = (*userInfoRepo)(nil)

// UserInfoRepo define a repo interface
type UserInfoRepo interface {
	CreateUserInfo(ctx context.Context, data *model.UserInfoModel) (id int64, err error)
	UpdateUserInfo(ctx context.Context, id int64, data *model.UserInfoModel) error
	GetUserInfo(ctx context.Context, id int64) (ret *model.UserInfoModel, err error)
	GetUserByUsername(ctx context.Context, username string) (ret *model.UserInfoModel, err error)
	GetUserByEmail(ctx context.Context, email string) (ret *model.UserInfoModel, err error)
	GetUserByPhone(ctx context.Context, phone string) (ret *model.UserInfoModel, err error)
	BatchGetUserInfo(ctx context.Context, ids []int64) (ret []*model.UserInfoModel, err error)
}

type userInfoRepo struct {
	db     *gorm.DB
	tracer trace.Tracer
	cache  cache.UserInfoCache
}

func (r *userInfoRepo) GetUserByUsername(ctx context.Context, username string) (ret *model.UserInfoModel, err error) {
	item := new(model.UserInfoModel)
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(_getUserInfoByUsernameSQL, _tableUserInfoName), username).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *userInfoRepo) GetUserByEmail(ctx context.Context, email string) (ret *model.UserInfoModel, err error) {
	item := new(model.UserInfoModel)
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(_getUserInfoByUsernameSQL, _tableUserInfoName), email).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *userInfoRepo) GetUserByPhone(ctx context.Context, phone string) (ret *model.UserInfoModel, err error) {
	item := new(model.UserInfoModel)
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(_getUserInfoByUsernameSQL, _tableUserInfoName), phone).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

// NewUserInfo new a repository and return
func NewUserInfo(db *gorm.DB, cache cache.UserInfoCache) UserInfoRepo {
	return &userInfoRepo{
		db:     db,
		tracer: otel.Tracer("userInfoRepo"),
		cache:  cache,
	}
}

// CreateUserInfo create a item
func (r *userInfoRepo) CreateUserInfo(ctx context.Context, data *model.UserInfoModel) (id int64, err error) {
	err = r.db.WithContext(ctx).Create(&data).Error
	if err != nil {
		return 0, errors.Wrap(err, "[repo] create UserInfo err")
	}

	return data.ID, nil
}

// UpdateUserInfo update item
func (r *userInfoRepo) UpdateUserInfo(ctx context.Context, id int64, data *model.UserInfoModel) error {
	item, err := r.GetUserInfo(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "[repo] update UserInfo err: %v", err)
	}
	err = r.db.Model(&item).Updates(data).Error
	if err != nil {
		return err
	}
	// delete cache
	_ = r.cache.DelUserInfoCache(ctx, id)
	return nil
}

// GetUserInfo get a record
func (r *userInfoRepo) GetUserInfo(ctx context.Context, id int64) (ret *model.UserInfoModel, err error) {
	// read cache
	item, err := r.cache.GetUserInfoCache(ctx, id)
	if err != nil {
		return nil, err
	}
	if item != nil {
		return item, nil
	}
	// read db
	data := new(model.UserInfoModel)
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(_getUserInfoSQL, _tableUserInfoName), id).Scan(&data).Error
	if err != nil {
		return
	}
	// write cache
	if data.ID > 0 {
		err = r.cache.SetUserInfoCache(ctx, id, data, 5*time.Minute)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// BatchGetUserInfo batch get items
func (r *userInfoRepo) BatchGetUserInfo(ctx context.Context, ids []int64) (ret []*model.UserInfoModel, err error) {
	// read cache
	idsStr := cast.ToStringSlice(ids)
	itemMap, err := r.cache.MultiGetUserInfoCache(ctx, ids)
	if err != nil {
		return nil, err
	}
	var missedID []int64
	for _, v := range ids {
		item, ok := itemMap[cast.ToString(v)]
		if !ok {
			missedID = append(missedID, v)
			continue
		}
		ret = append(ret, item)
	}
	// get missed data
	if len(missedID) > 0 {
		var missedData []*model.UserInfoModel
		_sql := fmt.Sprintf(_batchGetUserInfoSQL, _tableUserInfoName, strings.Join(idsStr, ","))
		err = r.db.WithContext(ctx).Raw(_sql).Scan(&missedData).Error
		if err != nil {
			// you can degrade to ignore error
			return nil, err
		}
		if len(missedData) > 0 {
			ret = append(ret, missedData...)
			err = r.cache.MultiSetUserInfoCache(ctx, missedData, 5*time.Minute)
			if err != nil {
				// you can degrade to ignore error
				return nil, err
			}
		}
	}
	return ret, nil
}
