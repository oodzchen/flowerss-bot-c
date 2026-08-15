package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/model"
)

func TestContentStorageImpl(t *testing.T) {
	db := GetTestDB(t)
	s := NewContentStorageImpl(db)
	ctx := context.Background()
	s.Init(ctx)

	content := &model.Content{
		SourceID: 1,
		HashID:   "id",
	}
	content2 := &model.Content{
		SourceID: 1,
		HashID:   "id2",
	}

	t.Run(
		"add content", func(t *testing.T) {
			err := s.AddContent(ctx, content)
			assert.Nil(t, err)
			err = s.AddContent(ctx, content2)
			assert.Nil(t, err)
		},
	)

	t.Run(
		"hash id exist", func(t *testing.T) {
			exist, err := s.HashIDExist(ctx, content.HashID)
			assert.Nil(t, err)
			assert.True(t, exist)
		},
	)

	t.Run(
		"del content", func(t *testing.T) {
			got, err := s.DeleteSourceContents(ctx, content.SourceID)
			assert.Nil(t, err)
			assert.Equal(t, int64(2), got)
		},
	)

	t.Run(
		"get and update content", func(t *testing.T) {
			c := &model.Content{
				SourceID:    10,
				HashID:      "id_edit",
				Title:       "Original Title",
				Description: "Original Description",
			}
			err := s.AddContent(ctx, c)
			assert.Nil(t, err)

			got, err := s.GetContent(ctx, "id_edit")
			assert.Nil(t, err)
			assert.Equal(t, "Original Title", got.Title)
			assert.Equal(t, "Original Description", got.Description)

			got.Title = "Updated Title"
			got.Description = "Updated Description"
			err = s.UpdateContent(ctx, "id_edit", got)
			assert.Nil(t, err)

			updated, err := s.GetContent(ctx, "id_edit")
			assert.Nil(t, err)
			assert.Equal(t, "Updated Title", updated.Title)
			assert.Equal(t, "Updated Description", updated.Description)
		},
	)

	t.Run(
		"content messages", func(t *testing.T) {
			msg1 := &model.ContentMessage{
				HashID:    "id_edit",
				UserID:    1001,
				MessageID: 555,
			}
			msg2 := &model.ContentMessage{
				HashID:    "id_edit",
				UserID:    1002,
				MessageID: 556,
			}

			err := s.SaveContentMessage(ctx, msg1)
			assert.Nil(t, err)
			err = s.SaveContentMessage(ctx, msg2)
			assert.Nil(t, err)

			gotMsg1, err := s.GetContentMessage(ctx, "id_edit", 1001)
			assert.Nil(t, err)
			assert.Equal(t, 555, gotMsg1.MessageID)

			// Update message id for user 1001
			msg1.MessageID = 777
			err = s.SaveContentMessage(ctx, msg1)
			assert.Nil(t, err)
			gotMsg1Updated, err := s.GetContentMessage(ctx, "id_edit", 1001)
			assert.Nil(t, err)
			assert.Equal(t, 777, gotMsg1Updated.MessageID)

			allMsgs, err := s.GetContentMessages(ctx, "id_edit")
			assert.Nil(t, err)
			assert.Equal(t, 2, len(allMsgs))

			// Delete by hash ID
			err = s.DeleteContentMessagesByHashID(ctx, "id_edit")
			assert.Nil(t, err)

			_, err = s.GetContentMessage(ctx, "id_edit", 1001)
			assert.Equal(t, ErrRecordNotFound, err)
		},
	)
}

