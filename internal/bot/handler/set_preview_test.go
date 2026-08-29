package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePreviewArg(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantVal   int
		expectErr bool
	}{
		{name: "default keyword", input: "default", wantNil: true},
		{name: "reset keyword", input: "reset", wantNil: true},
		{name: "chinese default", input: "默认", wantNil: true},
		{name: "chinese reset", input: "重置", wantNil: true},
		{name: "off keyword", input: "off", wantVal: 0},
		{name: "none keyword", input: "none", wantVal: 0},
		{name: "chinese close", input: "关闭", wantVal: 0},
		{name: "zero string", input: "0", wantVal: 0},
		{name: "positive integer", input: "400", wantVal: 400},
		{name: "positive integer with plus", input: "+400", wantVal: 400},
		{name: "negative integer", input: "-400", wantVal: -400},
		{name: "invalid text", input: "abc", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := parsePreviewArg(tt.input)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.wantNil {
					assert.Nil(t, val)
				} else {
					require.NotNil(t, val)
					assert.Equal(t, tt.wantVal, *val)
				}
			}
		})
	}
}
