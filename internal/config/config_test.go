package config

import (
	"reflect"
	"testing"
)

func TestSplitAndTrim(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "a,b,c", []string{"a", "b", "c"}},
		{"spaces", " a , b ,c ", []string{"a", "b", "c"}},
		{"at_prefix", "@channel_one, channel_two, @channel_three", []string{"channel_one", "channel_two", "channel_three"}},
		{"empty_items", "a,,b, ,c", []string{"a", "b", "c"}},
		{"single", "channel", []string{"channel"}},
		{"empty_string", "", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitAndTrim(tc.in)
			if tc.want == nil && len(got) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitAndTrim(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
