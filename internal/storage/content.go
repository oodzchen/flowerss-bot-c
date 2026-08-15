package storage

import (
	"context"

	"gorm.io/gorm"

	"github.com/indes/flowerss-bot/internal/model"
)

type ContentStorageImpl struct {
	db *gorm.DB
}

func NewContentStorageImpl(db *gorm.DB) *ContentStorageImpl {
	return &ContentStorageImpl{db: db}
}

func (s *ContentStorageImpl) Init(ctx context.Context) error {
	return s.db.Migrator().AutoMigrate(&model.Content{}, &model.ContentMessage{})
}

func (s *ContentStorageImpl) DeleteSourceContents(ctx context.Context, sourceID uint) (int64, error) {
	_ = s.db.WithContext(ctx).Where("hash_id IN (SELECT hash_id FROM contents WHERE source_id = ?)", sourceID).Delete(&model.ContentMessage{})
	result := s.db.WithContext(ctx).Where("source_id = ?", sourceID).Delete(&model.Content{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *ContentStorageImpl) AddContent(ctx context.Context, content *model.Content) error {
	result := s.db.WithContext(ctx).Create(content)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *ContentStorageImpl) HashIDExist(ctx context.Context, hashID string) (bool, error) {
	var count int64
	result := s.db.WithContext(ctx).Model(&model.Content{}).Where("hash_id = ?", hashID).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return (count > 0), nil
}

func (s *ContentStorageImpl) GetContent(ctx context.Context, hashID string) (*model.Content, error) {
	var contents []model.Content
	result := s.db.WithContext(ctx).Where("hash_id = ?", hashID).Limit(1).Find(&contents)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(contents) == 0 {
		return nil, ErrRecordNotFound
	}
	return &contents[0], nil
}

func (s *ContentStorageImpl) UpdateContent(ctx context.Context, hashID string, newContent *model.Content) error {
	newContent.HashID = hashID
	result := s.db.WithContext(ctx).Save(newContent)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *ContentStorageImpl) SaveContentMessage(ctx context.Context, msg *model.ContentMessage) error {
	result := s.db.WithContext(ctx).Save(msg)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *ContentStorageImpl) GetContentMessage(ctx context.Context, hashID string, userID int64) (*model.ContentMessage, error) {
	var msgs []model.ContentMessage
	result := s.db.WithContext(ctx).Where("hash_id = ? AND user_id = ?", hashID, userID).Limit(1).Find(&msgs)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(msgs) == 0 {
		return nil, ErrRecordNotFound
	}
	return &msgs[0], nil
}

func (s *ContentStorageImpl) GetContentMessages(ctx context.Context, hashID string) ([]*model.ContentMessage, error) {
	var msgs []*model.ContentMessage
	result := s.db.WithContext(ctx).Where("hash_id = ?", hashID).Find(&msgs)
	if result.Error != nil {
		return nil, result.Error
	}
	return msgs, nil
}

func (s *ContentStorageImpl) DeleteContentMessagesByHashID(ctx context.Context, hashID string) error {
	result := s.db.WithContext(ctx).Where("hash_id = ?", hashID).Delete(&model.ContentMessage{})
	return result.Error
}

func (s *ContentStorageImpl) DeleteContentMessagesBySourceID(ctx context.Context, sourceID uint) error {
	result := s.db.WithContext(ctx).Where("hash_id IN (SELECT hash_id FROM contents WHERE source_id = ?)", sourceID).Delete(&model.ContentMessage{})
	return result.Error
}

