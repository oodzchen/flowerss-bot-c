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
	// TranslateLang 推送前翻译的目标语言（语言代码，如 zh/en/ja），为空表示不翻译
	TranslateLang string
	// Timezone 推送时间显示的时区（如 Asia/Shanghai, +08:00, UTC），为空表示默认
	Timezone string
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
