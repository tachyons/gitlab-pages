package main

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiStringFlagAppendsOnSet(t *testing.T) {
	var concrete MultiStringFlag
	var iface flag.Value

	iface = &concrete

	require.NoError(t, iface.Set("foo"))
	require.NoError(t, iface.Set("bar"))

	require.EqualError(t, iface.Set(""), "value cannot be empty")

	require.Equal(t, MultiStringFlag{"foo", "bar"}, concrete)
}

func TestMultiStringFlag_Split(t *testing.T) {
	tests := []struct {
		name       string
		s          *MultiStringFlag
		wantResult []string
	}{
		{
			name:       "empty_string",
			s:          &MultiStringFlag{}, // -flag ""
			wantResult: []string{},
		},
		{
			name:       "one_value",
			s:          &MultiStringFlag{"value1"}, // -flag "value1"
			wantResult: []string{"value1"},
		},
		{
			name:       "multiple_values",
			s:          &MultiStringFlag{"value1", "", "value3"}, // -flag "value1,,value3"
			wantResult: []string{"value1", "", "value3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := tt.s.Split()
			require.ElementsMatch(t, tt.wantResult, gotResult)
		})
	}
}
