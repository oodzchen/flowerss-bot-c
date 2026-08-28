package model

import "time"

// Content fetcher content
type Content struct {
	SourceID     uint
	HashID       string `gorm:"primary_key"`
	RawID        string
	RawLink      string
	Title        string
	Description  string `gorm:"type:text"`
	TelegraphURL string
	PublishedAt  *time.Time
	Language     string
	EditTime
}
