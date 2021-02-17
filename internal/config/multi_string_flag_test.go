package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiStringFlagAppendsOnSet(t *testing.T) {
	var concrete MultiStringFlag
	iface := &concrete

	require.NoError(t, iface.Set("foo"))
	require.NoError(t, iface.Set("bar"))

	require.EqualError(t, iface.Set(""), "value cannot be empty")

	require.Equal(t, MultiStringFlag{value: []string{"foo", "bar"}}, concrete)
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
			s:          &MultiStringFlag{value: []string{"value1"}}, // -flag "value1"
			wantResult: []string{"value1"},
		},
		{
			name:       "multiple_values",
			s:          &MultiStringFlag{value: []string{"value1", "", "value3"}}, // -flag "value1,,value3"
			wantResult: []string{"value1", "", "value3"},
		},
		{
			name:       "multiple_values_in_one_string",
			s:          &MultiStringFlag{value: []string{"value1,value2"}}, // -flag "value1,value2"
			wantResult: []string{"value1", "value2"},
		},
		{
			name:       "different_separator",
			s:          &MultiStringFlag{value: []string{"value1", "value2"}, separator: ";"}, // -flag "value1;value2"
			wantResult: []string{"value1", "value2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := tt.s.Split()
			require.ElementsMatch(t, tt.wantResult, gotResult)
			require.Equal(t, strings.Join(gotResult, tt.s.separator), strings.Join(tt.wantResult, tt.s.separator))
		})
	}
}
