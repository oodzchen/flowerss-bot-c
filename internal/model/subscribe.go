package model

import "strings"

type Subscribe struct {
	ID                 uint `gorm:"primary_key;AUTO_INCREMENT"`
	UserID             int64
	SourceID           uint
	Title              string
	EnableNotification int
	EnableTelegraph    int
	Tag                string
	Interval           int
	WaitTime           int
	EditTime
}

// DisplayTitle returns the subscription-specific title when one is set,
// otherwise it falls back to the title provided by the RSS source.
func (s *Subscribe) DisplayTitle(sourceTitle string) string {
	if title := strings.TrimSpace(s.Title); title != "" {
		return title
	}
	return sourceTitle
}
