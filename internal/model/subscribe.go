package model

import (
	"strconv"
	"strings"
)

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
	// PreviewLength 推送正文预览字符数限制，nil 为遵循全局配置，0 为关闭预览，正数截取前 N 字符，负数截取后 N 字符
	PreviewLength *int `gorm:"column:preview_length"`
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

// GetPreviewLimit returns the effective preview character limit for this subscription,
// falling back to defaultLimit (e.g. config.PreviewText) if not set.
func (s *Subscribe) GetPreviewLimit(defaultLimit int) int {
	if s != nil && s.PreviewLength != nil {
		return *s.PreviewLength
	}
	return defaultLimit
}

// PreviewLengthDisplay returns a human-readable display string for this subscription's preview setting.
func (s *Subscribe) PreviewLengthDisplay(defaultLimit int) string {
	if s == nil || s.PreviewLength == nil {
		if defaultLimit == 0 {
			return "默认 (关闭)"
		} else if defaultLimit > 0 {
			return "默认 (前" + strings.TrimSpace(strconv.Itoa(defaultLimit)) + "字符)"
		} else {
			return "默认 (后" + strings.TrimSpace(strconv.Itoa(-defaultLimit)) + "字符)"
		}
	}
	val := *s.PreviewLength
	if val == 0 {
		return "关闭"
	} else if val > 0 {
		return "前" + strconv.Itoa(val) + "字符"
	} else {
		return "后" + strconv.Itoa(-val) + "字符"
	}
}
