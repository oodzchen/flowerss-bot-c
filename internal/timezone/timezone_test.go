package timezone

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseLocation(t *testing.T) {
	tests := []struct {
		input       string
		expectNil   bool
		expectError bool
		expectName  string
	}{
		{input: "", expectNil: true},
		{input: "off", expectNil: true},
		{input: "reset", expectNil: true},
		{input: "clear", expectNil: true},
		{input: "Asia/Shanghai", expectName: "Asia/Shanghai"},
		{input: "UTC", expectName: "UTC"},
		{input: "+08:00", expectName: "+08:00"},
		{input: "+8", expectName: "+08:00"},
		{input: "-05:00", expectName: "-05:00"},
		{input: "-5", expectName: "-05:00"},
		{input: "UTC+8", expectName: "+08:00"},
		{input: "invalid_tz", expectError: true},
		{input: "+25:00", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			loc, err := ParseLocation(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectNil {
					assert.Nil(t, loc)
				} else {
					assert.NotNil(t, loc)
					assert.Equal(t, tt.expectName, loc.String())
				}
			}
		})
	}
}

func TestNormalizeTimezone(t *testing.T) {
	tests := []struct {
		input  string
		expect string
		err    bool
	}{
		{input: "Asia/Shanghai", expect: "Asia/Shanghai"},
		{input: "+8", expect: "+08:00"},
		{input: "off", expect: ""},
		{input: "bad", err: true},
	}

	for _, tt := range tests {
		res, err := NormalizeTimezone(tt.input)
		if tt.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expect, res)
		}
	}
}

func TestTimezoneFormatTime(t *testing.T) {
	utcTime := time.Date(2026, 8, 14, 6, 30, 0, 0, time.UTC)

	loc, err := ParseLocation("Asia/Shanghai")
	assert.NoError(t, err)
	formatted := utcTime.In(loc).Format("2006-01-02 15:04:05 -07:00")
	assert.Equal(t, "2026-08-14 14:30:00 +08:00", formatted)

	locOffset, err := ParseLocation("+08:00")
	assert.NoError(t, err)
	formattedOffset := utcTime.In(locOffset).Format("2006-01-02 15:04:05 -07:00")
	assert.Equal(t, "2026-08-14 14:30:00 +08:00", formattedOffset)
}
