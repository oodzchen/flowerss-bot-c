package storage

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/internal/model"
)

type SubscriptionStorageImpl struct {
	db *gorm.DB
}

func NewSubscriptionStorageImpl(db *gorm.DB) *SubscriptionStorageImpl {
	return &SubscriptionStorageImpl{db: db.Model(&model.Subscribe{})}
}

func (s *SubscriptionStorageImpl) Init(ctx context.Context) error {
	return s.db.Migrator().AutoMigrate(&model.Subscribe{})
}

func (s *SubscriptionStorageImpl) AddSubscription(ctx context.Context, subscription *model.Subscribe) error {
	result := s.db.WithContext(ctx).Create(subscription)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *SubscriptionStorageImpl) SubscriptionExist(ctx context.Context, userID int64, sourceID uint) (bool, error) {
	var count int64
	result := s.db.WithContext(ctx).Where("user_id = ? and source_id = ?", userID, sourceID).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return (count > 0), nil
}

func (s *SubscriptionStorageImpl) GetSubscription(ctx context.Context, userID int64, sourceID uint) (
	*model.Subscribe, error,
) {
	subscription := &model.Subscribe{}
	result := s.db.WithContext(ctx).Where("user_id = ? and source_id = ?", userID, sourceID).First(subscription)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, result.Error
	}
	return subscription, nil
}

func (s *SubscriptionStorageImpl) GetSubscriptionsByUserID(
	ctx context.Context, userID int64, opts *GetSubscriptionsOptions,
) (*GetSubscriptionsResult, error) {
	var subscriptions []*model.Subscribe

	count := s.getSubscriptionsCount(opts)
	orderBy := s.getSubscriptionsOrderBy(opts)
	dbResult := s.db.WithContext(ctx).Where(
		&model.Subscribe{UserID: userID},
	).Limit(count).Order(orderBy).Offset(opts.Offset).Find(&subscriptions)
	if dbResult.Error != nil {
		if errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, dbResult.Error
	}

	result := &GetSubscriptionsResult{}
	if opts.Count != -1 && len(subscriptions) > opts.Count {
		result.HasMore = true
		subscriptions = subscriptions[:opts.Count]
	}

	result.Subscriptions = subscriptions
	return result, nil
}

func (s *SubscriptionStorageImpl) GetSubscriptionsBySourceID(
	ctx context.Context, sourceID uint, opts *GetSubscriptionsOptions,
) (*GetSubscriptionsResult, error) {
	var subscriptions []*model.Subscribe

	count := s.getSubscriptionsCount(opts)
	orderBy := s.getSubscriptionsOrderBy(opts)
	dbResult := s.db.WithContext(ctx).Where(
		&model.Subscribe{SourceID: sourceID},
	).Limit(count).Order(orderBy).Offset(opts.Offset).Find(&subscriptions)
	if dbResult.Error != nil {
		if errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, dbResult.Error
	}

	result := &GetSubscriptionsResult{}
	if opts.Count > 0 && len(subscriptions) > opts.Count {
		result.HasMore = true
		subscriptions = subscriptions[:opts.Count]
	}

	result.Subscriptions = subscriptions
	return result, nil
}

func (s *SubscriptionStorageImpl) getSubscriptionsCount(opts *GetSubscriptionsOptions) int {
	count := opts.Count
	if count != -1 {
		count += 1
	}
	return count
}

func (s *SubscriptionStorageImpl) getSubscriptionsOrderBy(opts *GetSubscriptionsOptions) string {
	switch opts.SortType {
	case SubscriptionSortTypeCreatedTimeDesc:
		return "created_at desc"
	}
	return ""
}

func (s *SubscriptionStorageImpl) CountSubscriptions(ctx context.Context) (int64, error) {
	var count int64
	dbResult := s.db.WithContext(ctx).Count(&count)
	if dbResult.Error != nil {
		return 0, dbResult.Error
	}
	return count, nil
}

func (s *SubscriptionStorageImpl) DeleteSubscription(ctx context.Context, userID int64, sourceID uint) (int64, error) {
	result := s.db.WithContext(ctx).Where(
		"user_id = ? and source_id = ?", userID, sourceID,
	).Delete(&model.Subscribe{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *SubscriptionStorageImpl) CountSourceSubscriptions(ctx context.Context, sourceID uint) (int64, error) {
	var count int64
	result := s.db.WithContext(ctx).Where("source_id = ?", sourceID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (s *SubscriptionStorageImpl) UpdateSubscription(
	ctx context.Context, userID int64, sourceID uint, newSubscription *model.Subscribe,
) error {
	result := s.db.WithContext(ctx).Where(
		"user_id = ? and source_id = ?", userID, sourceID,
	).Updates(newSubscription)
	if result.Error != nil {
		return result.Error
	}
	log.Debugf(
		"update %d row, userID %d sourceID %d new %#v", result.RowsAffected, userID, sourceID, newSubscription,
	)
	return nil
}

func (s *SubscriptionStorageImpl) UpsertSubscription(
	ctx context.Context, userID int64, sourceID uint, newSubscription *model.Subscribe,
) error {
	// 不依赖 GORM Save 的分支语义（不同版本对带主键结构体的 Save 行为不一致，
	// 部分版本会退化成 INSERT 并触发主键冲突），这里显式使用 ON CONFLICT upsert，
	// 跨 GORM 版本行为一致：行存在则全列更新，不存在则插入。
	result := s.db.WithContext(ctx).Clauses(
		clause.OnConflict{UpdateAll: true},
	).Create(newSubscription)
	if result.Error != nil {
		return result.Error
	}
	log.Debugf(
		"upsert %d row, userID %d sourceID %d new %#v", result.RowsAffected, userID, sourceID, newSubscription,
	)
	return nil
}

// UpdateSubscriptionLang updates the translate language of one subscription.
// A dedicated column update is used so an empty lang (disabling translation)
// is persisted and the statement can never degrade into an INSERT.
func (s *SubscriptionStorageImpl) UpdateSubscriptionLang(
	ctx context.Context, userID int64, sourceID uint, lang string,
) (int64, error) {
	result := s.db.WithContext(ctx).Where(
		"user_id = ? and source_id = ?", userID, sourceID,
	).Update("translate_lang", lang)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// UpdateSubscriptionsLang updates the translate language of every subscription
// owned by userID.
func (s *SubscriptionStorageImpl) UpdateSubscriptionsLang(
	ctx context.Context, userID int64, lang string,
) (int64, error) {
	result := s.db.WithContext(ctx).Where("user_id = ?", userID).Update("translate_lang", lang)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
