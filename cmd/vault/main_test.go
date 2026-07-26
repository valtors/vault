package main

import (
	"testing"
)

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a,,c", []string{"a", "c"}},
		{",a,b,", []string{"a", "b"}},
		{"hello,world", []string{"hello", "world"}},
		{"single", []string{"single"}},
		{",,,", nil},
		{"a,b,c,d,e", []string{"a", "b", "c", "d", "e"}},
	}
	for _, tt := range tests {
		got := splitCSV(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestUsage(t *testing.T) {
	usage()
}
