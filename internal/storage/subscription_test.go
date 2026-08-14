package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/model"
)

func TestSubscriptionStorageImpl(t *testing.T) {
	db := GetTestDB(t)
	s := NewSubscriptionStorageImpl(db)
	ctx := context.Background()
	s.Init(ctx)

	subscriptions := []*model.Subscribe{
		&model.Subscribe{
			SourceID:           1,
			UserID:             100,
			EnableNotification: 1,
		},
		&model.Subscribe{
			SourceID:           1,
			UserID:             101,
			EnableNotification: 1,
		},
		&model.Subscribe{
			SourceID:           2,
			UserID:             100,
			EnableNotification: 1,
		},
		&model.Subscribe{
			SourceID:           2,
			UserID:             101,
			EnableNotification: 1,
		},
		&model.Subscribe{
			SourceID:           3,
			UserID:             101,
			EnableNotification: 1,
		},
	}

	t.Run(
		"add subscription", func(t *testing.T) {
			for _, subscription := range subscriptions {
				err := s.AddSubscription(ctx, subscription)
				assert.Nil(t, err)
			}
			got, err := s.CountSubscriptions(ctx)
			assert.Nil(t, err)
			assert.Equal(t, int64(5), got)

			exist, err := s.SubscriptionExist(ctx, 101, 1)
			assert.Nil(t, err)
			assert.True(t, exist)

			subscription, err := s.GetSubscription(ctx, 101, 1)
			assert.Nil(t, err)
			assert.NotNil(t, subscription)

			opt := &GetSubscriptionsOptions{
				Count: 2,
			}
			result, err := s.GetSubscriptionsByUserID(ctx, 101, opt)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(result.Subscriptions))
			assert.True(t, result.HasMore)

			opt = &GetSubscriptionsOptions{
				Count:  1,
				Offset: 2,
			}
			result, err = s.GetSubscriptionsByUserID(ctx, 101, opt)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(result.Subscriptions))
			assert.False(t, result.HasMore)

			opt = &GetSubscriptionsOptions{
				Count: 2,
			}
			result, err = s.GetSubscriptionsBySourceID(ctx, 1, opt)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(result.Subscriptions))
			assert.False(t, result.HasMore)

			opt = &GetSubscriptionsOptions{
				Count:  1,
				Offset: 2,
			}
			result, err = s.GetSubscriptionsByUserID(ctx, 1, opt)
			assert.Nil(t, err)
			assert.Equal(t, 0, len(result.Subscriptions))
			assert.False(t, result.HasMore)

			got, err = s.DeleteSubscription(ctx, 101, 1)
			assert.Nil(t, err)
			assert.Equal(t, int64(1), got)

			exist, err = s.SubscriptionExist(ctx, 101, 1)
			assert.Nil(t, err)
			assert.False(t, exist)

			subscription, err = s.GetSubscription(ctx, 101, 1)
			assert.Error(t, err)
			assert.Nil(t, subscription)

			got, err = s.CountSubscriptions(ctx)
			assert.Nil(t, err)
			assert.Equal(t, int64(4), got)

			got, err = s.CountSourceSubscriptions(ctx, 2)
			assert.Nil(t, err)
			assert.Equal(t, int64(2), got)
		},
	)

	t.Run(
		"update subscription", func(t *testing.T) {
			sub := &model.Subscribe{
				ID:                 10001,
				SourceID:           1000,
				UserID:             1002,
				EnableNotification: 1,
			}
			err := s.UpdateSubscription(ctx, sub.UserID, sub.SourceID, sub)
			assert.Nil(t, err)

			err = s.AddSubscription(ctx, sub)
			assert.Nil(t, err)

			sub.Tag = "tag"
			err = s.UpdateSubscription(ctx, sub.UserID, sub.SourceID, sub)
			assert.Nil(t, err)

			subscription, err := s.GetSubscription(ctx, sub.UserID, sub.SourceID)
			assert.Nil(t, err)
			assert.Equal(t, sub.Tag, subscription.Tag)
		},
	)

	t.Run(
		"upsert subscription", func(t *testing.T) {
			sub := &model.Subscribe{
				ID:                 10001,
				SourceID:           1000,
				UserID:             1002,
				EnableNotification: 1,
			}
			err := s.UpsertSubscription(ctx, sub.UserID, sub.SourceID, sub)
			assert.Nil(t, err)

			err = s.AddSubscription(ctx, sub)
			assert.Error(t, err)

			sub.Tag = "tag"
			err = s.UpsertSubscription(ctx, sub.UserID, sub.SourceID, sub)
			assert.Nil(t, err)

			subscription, err := s.GetSubscription(ctx, sub.UserID, sub.SourceID)
			assert.Nil(t, err)
			assert.Equal(t, sub.Tag, subscription.Tag)
		},
	)

	t.Run(
		"upsert loaded subscription must not degrade to insert", func(t *testing.T) {
			// 回归：GORM 的 Save 在不同版本对已加载（带主键）结构体行为不一致，
			// 部分版本会退化为 INSERT 并触发主键冲突。UpsertSubscription 必须
			// 对已加载行执行更新而不是插入。
			sub := &model.Subscribe{
				SourceID:           2000,
				UserID:             2001,
				EnableNotification: 1,
				EnableTelegraph:    1,
				Interval:           10,
				WaitTime:           10,
			}
			err := s.AddSubscription(ctx, sub)
			assert.Nil(t, err)
			assert.NotZero(t, sub.ID)

			loaded, err := s.GetSubscription(ctx, sub.UserID, sub.SourceID)
			assert.Nil(t, err)
			loaded.TranslateLang = "zh"
			err = s.UpsertSubscription(ctx, loaded.UserID, loaded.SourceID, loaded)
			assert.Nil(t, err)

			check, err := s.GetSubscription(ctx, sub.UserID, sub.SourceID)
			assert.Nil(t, err)
			assert.Equal(t, "zh", check.TranslateLang)
		},
	)

	t.Run(
		"upsert persists empty string", func(t *testing.T) {
			// 回归：恢复空值（如清空标题）必须能写回，零值不能像 Updates(struct) 那样被跳过
			sub := &model.Subscribe{
				SourceID:           3000,
				UserID:             3001,
				EnableNotification: 1,
				Title:              "custom",
			}
			err := s.AddSubscription(ctx, sub)
			assert.Nil(t, err)

			loaded, err := s.GetSubscription(ctx, sub.UserID, sub.SourceID)
			assert.Nil(t, err)
			loaded.Title = ""
			err = s.UpsertSubscription(ctx, loaded.UserID, loaded.SourceID, loaded)
			assert.Nil(t, err)

			check, err := s.GetSubscription(ctx, sub.UserID, sub.SourceID)
			assert.Nil(t, err)
			assert.Empty(t, check.Title)
		},
	)
}

func TestSubscriptionLangUpdate(t *testing.T) {
	db := GetTestDB(t)
	s := NewSubscriptionStorageImpl(db)
	ctx := context.Background()
	s.Init(ctx)

	add := func(userID int64, sourceID uint) *model.Subscribe {
		sub := &model.Subscribe{
			UserID:             userID,
			SourceID:           sourceID,
			EnableNotification: 1,
			EnableTelegraph:    1,
			Interval:           10,
			WaitTime:           10,
		}
		assert.NoError(t, s.AddSubscription(ctx, sub))
		return sub
	}
	add(4001, 1)
	add(4001, 2)
	add(4002, 1)

	t.Run("update single subscription lang", func(t *testing.T) {
		affected, err := s.UpdateSubscriptionLang(ctx, 4001, 1, "zh")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		sub, err := s.GetSubscription(ctx, 4001, 1)
		assert.NoError(t, err)
		assert.Equal(t, "zh", sub.TranslateLang)

		// 未订阅的 (userID, sourceID) 应影响 0 行且不报错
		affected, err = s.UpdateSubscriptionLang(ctx, 4001, 999, "zh")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})

	t.Run("update all subscriptions lang", func(t *testing.T) {
		affected, err := s.UpdateSubscriptionsLang(ctx, 4001, "ja")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), affected)

		for _, sourceID := range []uint{1, 2} {
			sub, err := s.GetSubscription(ctx, 4001, sourceID)
			assert.NoError(t, err)
			assert.Equal(t, "ja", sub.TranslateLang)
		}

		// 其他用户不受影响
		sub, err := s.GetSubscription(ctx, 4002, 1)
		assert.NoError(t, err)
		assert.Empty(t, sub.TranslateLang)
	})

	t.Run("empty lang disables translation and is persisted", func(t *testing.T) {
		affected, err := s.UpdateSubscriptionsLang(ctx, 4001, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), affected)

		sub, err := s.GetSubscription(ctx, 4001, 1)
		assert.NoError(t, err)
		assert.Empty(t, sub.TranslateLang)
	})
}
