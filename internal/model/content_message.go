package model

// ContentMessage stores the telegram message ID sent to a user/chat for a content
type ContentMessage struct {
	HashID    string `gorm:"primary_key;column:hash_id"`
	UserID    int64  `gorm:"primary_key;column:user_id"`
	MessageID int    `gorm:"not null;column:message_id"`
	EditTime
}
