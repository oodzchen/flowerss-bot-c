package core

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/model"
	"github.com/indes/flowerss-bot/internal/storage"
	"github.com/indes/flowerss-bot/internal/storage/mock"
)

type mockStorage struct {
	User         *mock.MockUser
	Content      *mock.MockContent
	Source       *mock.MockSource
	Subscription *mock.MockSubscription
	Ctrl         *gomock.Controller
}

func getTestCore(t *testing.T) (*Core, *mockStorage) {
	ctrl := gomock.NewController(t)

	s := &mockStorage{
		Subscription: mock.NewMockSubscription(ctrl),
		User:         mock.NewMockUser(ctrl),
		Content:      mock.NewMockContent(ctrl),
		Source:       mock.NewMockSource(ctrl),
		Ctrl:         ctrl,
	}
	c := NewCore(s.User, s.Content, s.Source, s.Subscription, nil, nil)
	return c, s
}

func TestCore_AddSubscription(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()

	userID := int64(1)
	sourceID := uint(101)
	t.Run(
		"exist error", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(false, errors.New("err")).Times(1)
			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"exist subscription", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(true, nil).Times(1)
			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Equal(t, ErrSubscriptionExist, err)
		},
	)

	t.Run(
		"subscribe fail", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(false, nil).Times(1)
			s.Subscription.EXPECT().AddSubscription(ctx, gomock.Any()).Return(errors.New("err")).Times(1)

			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"subscribe ok", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(false, nil).Times(1)
			s.Subscription.EXPECT().AddSubscription(ctx, gomock.Any()).Return(nil).Times(1)

			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_GetUserSubscribedSources(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()

	userID := int64(1)
	sourceID1 := uint(101)
	sourceID2 := uint(102)
	subscriptionsResult := &storage.GetSubscriptionsResult{
		Subscriptions: []*model.Subscribe{
			&model.Subscribe{SourceID: sourceID1},
			&model.Subscribe{SourceID: sourceID2},
		},
	}
	t.Run(
		"subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscriptionsByUserID(ctx, userID, gomock.Any()).Return(
				nil, errors.New("err"),
			)

			sources, err := c.GetUserSubscribedSources(ctx, userID)
			assert.Error(t, err)
			assert.Nil(t, sources)
		},
	)

	t.Run(
		"source err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscriptionsByUserID(ctx, userID, gomock.Any()).Return(
				subscriptionsResult, nil,
			)

			s.Source.EXPECT().GetSource(ctx, sourceID1).Return(
				nil, errors.New("err"),
			).Times(1)
			s.Source.EXPECT().GetSource(ctx, gomock.Any()).Return(
				&model.Source{}, nil,
			)

			sources, err := c.GetUserSubscribedSources(ctx, userID)
			assert.Nil(t, err)
			assert.Equal(t, len(subscriptionsResult.Subscriptions)-1, len(sources))
		},
	)

	t.Run(
		"source success", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscriptionsByUserID(ctx, userID, gomock.Any()).Return(
				subscriptionsResult, nil,
			)

			s.Source.EXPECT().GetSource(ctx, gomock.Any()).Return(
				&model.Source{}, nil,
			).Times(len(subscriptionsResult.Subscriptions))

			sources, err := c.GetUserSubscribedSources(ctx, userID)
			assert.Nil(t, err)
			assert.Equal(t, len(subscriptionsResult.Subscriptions), len(sources))
		},
	)
}

func TestCore_Unsubscribe(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()

	userID := int64(1)
	sourceID1 := uint(101)

	t.Run(
		"SubscriptionExist err", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID1).Return(
				false, errors.New("err"),
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)
		},
	)

	t.Run(
		"subscription not exist", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID1).Return(
				false, nil,
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Equal(t, ErrSubscriptionNotExist, err)
		},
	)

	s.Subscription.EXPECT().SubscriptionExist(ctx, gomock.Any(), gomock.Any()).Return(
		true, nil,
	).AnyTimes()

	t.Run(
		"unsubscribe failed", func(t *testing.T) {
			s.Subscription.EXPECT().DeleteSubscription(ctx, userID, sourceID1).Return(
				int64(1), errors.New("err"),
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)
		},
	)

	s.Subscription.EXPECT().DeleteSubscription(ctx, gomock.Any(), gomock.Any()).Return(
		int64(1), nil,
	).AnyTimes()

	t.Run(
		"count subs", func(t *testing.T) {
			s.Subscription.EXPECT().CountSourceSubscriptions(ctx, sourceID1).Return(
				int64(1), errors.New("err"),
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)

			s.Subscription.EXPECT().CountSourceSubscriptions(ctx, sourceID1).Return(
				int64(1), nil,
			).Times(1)
			err = c.Unsubscribe(ctx, userID, sourceID1)
			assert.Nil(t, err)
		},
	)

	s.Subscription.EXPECT().CountSourceSubscriptions(ctx, gomock.Any()).Return(
		int64(0), nil,
	).AnyTimes()

	t.Run(
		"remove source", func(t *testing.T) {
			s.Source.EXPECT().Delete(ctx, sourceID1).Return(
				errors.New("err"),
			).Times(1)

			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)

			s.Source.EXPECT().Delete(ctx, sourceID1).Return(nil).AnyTimes()

			s.Content.EXPECT().DeleteSourceContents(ctx, sourceID1).Return(int64(0), errors.New("err")).Times(1)
			err = c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)

			s.Content.EXPECT().DeleteSourceContents(ctx, sourceID1).Return(int64(1), nil).Times(1)
			err = c.Unsubscribe(ctx, userID, sourceID1)
			assert.Nil(t, err)
		},
	)
}

func TestCore_GetSourceByURL(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	sourceURL := "http://google.com"

	t.Run(
		"source err", func(t *testing.T) {
			s.Source.EXPECT().GetSourceByURL(ctx, sourceURL).Return(
				nil, errors.New("err"),
			).Times(1)
			got, err := c.GetSourceByURL(ctx, sourceURL)
			assert.Error(t, err)
			assert.Nil(t, got)
		},
	)

	t.Run(
		"source not exist", func(t *testing.T) {
			s.Source.EXPECT().GetSourceByURL(ctx, sourceURL).Return(
				nil, storage.ErrRecordNotFound,
			).Times(1)
			got, err := c.GetSourceByURL(ctx, sourceURL)
			assert.Equal(t, ErrSourceNotExist, err)
			assert.Nil(t, got)
		},
	)

	t.Run(
		"ok", func(t *testing.T) {
			s.Source.EXPECT().GetSourceByURL(ctx, sourceURL).Return(
				&model.Source{}, nil,
			).Times(1)
			got, err := c.GetSourceByURL(ctx, sourceURL)
			assert.Nil(t, err)
			assert.NotNil(t, got)
		},
	)
}

func TestCore_GetSource(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	sourceID := uint(1)

	t.Run(
		"source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			got, err := c.GetSource(ctx, sourceID)
			assert.Error(t, err)
			assert.Nil(t, got)
		},
	)

	t.Run(
		"source not exist", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				nil, storage.ErrRecordNotFound,
			).Times(1)
			got, err := c.GetSource(ctx, sourceID)
			assert.Equal(t, ErrSourceNotExist, err)
			assert.Nil(t, got)
		},
	)

	t.Run(
		"ok", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)
			got, err := c.GetSource(ctx, sourceID)
			assert.Nil(t, err)
			assert.NotNil(t, got)
		},
	)
}

func TestCore_GetSubscription(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(101)
	sourceID := uint(1)

	t.Run(
		"subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			got, err := c.GetSubscription(ctx, userID, sourceID)
			assert.Error(t, err)
			assert.Nil(t, got)
		},
	)

	t.Run(
		"subscription not exist", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				nil, storage.ErrRecordNotFound,
			).Times(1)
			got, err := c.GetSubscription(ctx, userID, sourceID)
			assert.Equal(t, ErrSubscriptionNotExist, err)
			assert.Nil(t, got)
		},
	)

	t.Run(
		"ok", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				&model.Subscribe{}, nil,
			).Times(1)
			got, err := c.GetSubscription(ctx, userID, sourceID)
			assert.Nil(t, err)
			assert.NotNil(t, got)
		},
	)
}

func TestCore_SetSubscriptionTitle(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(101)
	sourceID := uint(1)

	t.Run("get subscription error", func(t *testing.T) {
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(nil, errors.New("err"))
		err := c.SetSubscriptionTitle(ctx, userID, sourceID, "custom title")
		assert.Error(t, err)
	})

	t.Run("set custom title", func(t *testing.T) {
		subscription := &model.Subscribe{ID: 1, UserID: userID, SourceID: sourceID}
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(subscription, nil)
		s.Subscription.EXPECT().UpsertSubscription(ctx, userID, sourceID, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ int64, _ uint, got *model.Subscribe) error {
				assert.Equal(t, "custom title", got.Title)
				return nil
			},
		)

		err := c.SetSubscriptionTitle(ctx, userID, sourceID, "  custom title  ")
		assert.NoError(t, err)
	})

	t.Run("restore source title", func(t *testing.T) {
		subscription := &model.Subscribe{ID: 1, UserID: userID, SourceID: sourceID, Title: "custom title"}
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(subscription, nil)
		s.Subscription.EXPECT().UpsertSubscription(ctx, userID, sourceID, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ int64, _ uint, got *model.Subscribe) error {
				assert.Empty(t, got.Title)
				return nil
			},
		)

		err := c.SetSubscriptionTitle(ctx, userID, sourceID, "")
		assert.NoError(t, err)
	})
}

func TestCore_SetSubscriptionLang(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(101)
	sourceID := uint(1)

	t.Run("subscription not exist", func(t *testing.T) {
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(nil, storage.ErrRecordNotFound)
		err := c.SetSubscriptionLang(ctx, userID, sourceID, "zh")
		assert.Equal(t, ErrSubscriptionNotExist, err)
	})

	t.Run("update lang", func(t *testing.T) {
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
			&model.Subscribe{UserID: userID, SourceID: sourceID}, nil,
		)
		s.Subscription.EXPECT().UpdateSubscriptionLang(ctx, userID, sourceID, "zh").Return(int64(1), nil)
		err := c.SetSubscriptionLang(ctx, userID, sourceID, "zh")
		assert.NoError(t, err)
	})

	t.Run("update lang error", func(t *testing.T) {
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
			&model.Subscribe{UserID: userID, SourceID: sourceID}, nil,
		)
		s.Subscription.EXPECT().UpdateSubscriptionLang(ctx, userID, sourceID, "zh").Return(
			int64(0), errors.New("db err"),
		)
		err := c.SetSubscriptionLang(ctx, userID, sourceID, "zh")
		assert.Error(t, err)
	})
}

func TestCore_SetSubscriptionLangForAll(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(101)

	t.Run("update all langs", func(t *testing.T) {
		s.Subscription.EXPECT().UpdateSubscriptionsLang(ctx, userID, "ja").Return(int64(3), nil)
		count, err := c.SetSubscriptionLangForAll(ctx, userID, "ja")
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("update all langs error", func(t *testing.T) {
		s.Subscription.EXPECT().UpdateSubscriptionsLang(ctx, userID, "").Return(
			int64(0), errors.New("db err"),
		)
		count, err := c.SetSubscriptionLangForAll(ctx, userID, "")
		assert.Error(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestCore_SetSubscriptionTimezone(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(101)
	sourceID := uint(1)

	t.Run("success", func(t *testing.T) {
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(&model.Subscribe{}, nil)
		s.Subscription.EXPECT().UpdateSubscriptionTimezone(ctx, userID, sourceID, "Asia/Shanghai").Return(int64(1), nil)
		err := c.SetSubscriptionTimezone(ctx, userID, sourceID, "Asia/Shanghai")
		assert.NoError(t, err)
	})

	t.Run("subscription not exist", func(t *testing.T) {
		s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(nil, storage.ErrRecordNotFound)
		err := c.SetSubscriptionTimezone(ctx, userID, sourceID, "Asia/Shanghai")
		assert.Equal(t, ErrSubscriptionNotExist, err)
	})
}

func TestCore_SetSubscriptionTimezoneForAll(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(101)

	t.Run("update all timezones", func(t *testing.T) {
		s.Subscription.EXPECT().UpdateSubscriptionsTimezone(ctx, userID, "+08:00").Return(int64(2), nil)
		count, err := c.SetSubscriptionTimezoneForAll(ctx, userID, "+08:00")
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestCore_DisableSourceUpdate(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	sourceID := uint(1)

	t.Run(
		"get source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			err := c.DisableSourceUpdate(ctx, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)

			s.Source.EXPECT().UpsertSource(ctx, sourceID, gomock.Any()).Return(
				errors.New("err"),
			).Times(1)
			err := c.DisableSourceUpdate(ctx, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)

			s.Source.EXPECT().UpsertSource(ctx, sourceID, gomock.Any()).Return(
				nil,
			).Times(1)
			err := c.DisableSourceUpdate(ctx, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_ClearSourceErrorCount(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	sourceID := uint(1)

	t.Run(
		"get source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			err := c.ClearSourceErrorCount(ctx, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)

			s.Source.EXPECT().UpsertSource(ctx, sourceID, gomock.Any()).Return(
				errors.New("err"),
			).Times(1)
			err := c.ClearSourceErrorCount(ctx, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)

			s.Source.EXPECT().UpsertSource(ctx, sourceID, gomock.Any()).Return(
				nil,
			).Times(1)
			err := c.ClearSourceErrorCount(ctx, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_ToggleSubscriptionNotice(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(123)
	sourceID := uint(1)

	t.Run(
		"get subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			err := c.ToggleSubscriptionNotice(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				&model.Subscribe{}, nil,
			).Times(1)

			s.Subscription.EXPECT().UpsertSubscription(ctx, userID, sourceID, gomock.Any()).Return(
				errors.New("err"),
			).Times(1)

			err := c.ToggleSubscriptionNotice(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"ok", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				&model.Subscribe{}, nil,
			).Times(1)

			s.Subscription.EXPECT().UpsertSubscription(ctx, userID, sourceID, gomock.Any()).Return(
				nil,
			).Times(1)

			err := c.ToggleSubscriptionNotice(ctx, userID, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_ToggleSourceUpdateStatus(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	sourceID := uint(1)

	t.Run(
		"get source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			err := c.ToggleSourceUpdateStatus(ctx, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update source err", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)

			s.Source.EXPECT().UpsertSource(ctx, sourceID, gomock.Any()).Return(
				errors.New("err"),
			).Times(1)
			err := c.ToggleSourceUpdateStatus(ctx, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"ok", func(t *testing.T) {
			s.Source.EXPECT().GetSource(ctx, sourceID).Return(
				&model.Source{}, nil,
			).Times(1)

			s.Source.EXPECT().UpsertSource(ctx, sourceID, gomock.Any()).Return(
				nil,
			).Times(1)
			err := c.ToggleSourceUpdateStatus(ctx, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_ToggleSubscriptionTelegraph(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	userID := int64(123)
	sourceID := uint(1)

	t.Run(
		"get subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				nil, errors.New("err"),
			).Times(1)
			err := c.ToggleSubscriptionTelegraph(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"update subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				&model.Subscribe{}, nil,
			).Times(1)

			s.Subscription.EXPECT().UpsertSubscription(ctx, userID, sourceID, gomock.Any()).Return(
				errors.New("err"),
			).Times(1)

			err := c.ToggleSubscriptionTelegraph(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"ok", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscription(ctx, userID, sourceID).Return(
				&model.Subscribe{}, nil,
			).Times(1)

			s.Subscription.EXPECT().UpsertSubscription(ctx, userID, sourceID, gomock.Any()).Return(
				nil,
			).Times(1)

			err := c.ToggleSubscriptionTelegraph(ctx, userID, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_ContentAndMessages(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()
	hashID := "hash123"
	userID := int64(1001)
	msgID := 888

	t.Run("get content success", func(t *testing.T) {
		expected := &model.Content{HashID: hashID, Title: "Title"}
		s.Content.EXPECT().GetContent(ctx, hashID).Return(expected, nil)
		got, err := c.GetContent(ctx, hashID)
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("get content not found", func(t *testing.T) {
		s.Content.EXPECT().GetContent(ctx, hashID).Return(nil, storage.ErrRecordNotFound)
		got, err := c.GetContent(ctx, hashID)
		assert.Equal(t, ErrContentNotExist, err)
		assert.Nil(t, got)
	})

	t.Run("update content", func(t *testing.T) {
		content := &model.Content{HashID: hashID, Title: "New Title"}
		s.Content.EXPECT().UpdateContent(ctx, hashID, content).Return(nil)
		err := c.UpdateContent(ctx, hashID, content)
		assert.NoError(t, err)
	})

	t.Run("save and get content message", func(t *testing.T) {
		s.Content.EXPECT().SaveContentMessage(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, msg *model.ContentMessage) error {
				assert.Equal(t, hashID, msg.HashID)
				assert.Equal(t, userID, msg.UserID)
				assert.Equal(t, msgID, msg.MessageID)
				return nil
			},
		)
		err := c.SaveContentMessage(ctx, hashID, userID, msgID)
		assert.NoError(t, err)

		s.Content.EXPECT().GetContentMessage(ctx, hashID, userID).Return(&model.ContentMessage{
			HashID:    hashID,
			UserID:    userID,
			MessageID: msgID,
		}, nil)
		msg, err := c.GetContentMessage(ctx, hashID, userID)
		assert.NoError(t, err)
		assert.Equal(t, msgID, msg.MessageID)
	})

	t.Run("get content messages", func(t *testing.T) {
		expectedMsgs := []*model.ContentMessage{
			{HashID: hashID, UserID: 1001, MessageID: 1},
			{HashID: hashID, UserID: 1002, MessageID: 2},
		}
		s.Content.EXPECT().GetContentMessages(ctx, hashID).Return(expectedMsgs, nil)
		msgs, err := c.GetContentMessages(ctx, hashID)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(msgs))
	})
}

