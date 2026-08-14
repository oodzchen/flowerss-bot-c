package timezone

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidTimezone 无效的时区字符串
	ErrInvalidTimezone = errors.New("invalid timezone")
	offsetRegexp       = regexp.MustCompile(`^(?:UTC|GMT)?([+-])(\d{1,2})(?::?(\d{2}))?$`)
)

// ParseLocation 将时区字符串解析为 *time.Location。
// 若 tz 为空或 "off"/"reset"/"clear"，返回 nil, nil。
func ParseLocation(tz string) (*time.Location, error) {
	tz = strings.TrimSpace(tz)
	if tz == "" || strings.EqualFold(tz, "off") || strings.EqualFold(tz, "reset") || strings.EqualFold(tz, "clear") {
		return nil, nil
	}

	// 尝试 IANA 时区（如 "Asia/Shanghai", "UTC", "America/New_York"）
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc, nil
	}

	// 尝试解析时区偏移（如 "+08:00", "+8", "-5", "UTC+8"）
	upper := strings.ToUpper(tz)
	matches := offsetRegexp.FindStringSubmatch(upper)
	if len(matches) == 4 {
		sign := 1
		if matches[1] == "-" {
			sign = -1
		}
		hours, _ := strconv.Atoi(matches[2])
		minutes := 0
		if matches[3] != "" {
			minutes, _ = strconv.Atoi(matches[3])
		}

		if hours > 14 || minutes > 59 {
			return nil, ErrInvalidTimezone
		}

		totalSeconds := sign * (hours*3600 + minutes*60)
		name := fmt.Sprintf("%s%02d:%02d", matches[1], hours, minutes)
		return time.FixedZone(name, totalSeconds), nil
	}

	return nil, ErrInvalidTimezone
}

// NormalizeTimezone 校验并规范化时区字符串以便持久化存储。
// 空字符串表示恢复默认。
func NormalizeTimezone(tz string) (string, error) {
	tz = strings.TrimSpace(tz)
	if tz == "" || strings.EqualFold(tz, "off") || strings.EqualFold(tz, "reset") || strings.EqualFold(tz, "clear") {
		return "", nil
	}

	loc, err := ParseLocation(tz)
	if err != nil {
		return "", err
	}
	if loc == nil {
		return "", nil
	}
	return loc.String(), nil
}
