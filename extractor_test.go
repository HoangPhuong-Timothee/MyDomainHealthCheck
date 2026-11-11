package main

import (
	"testing"
)

func TestExtractTarget(t *testing.T) {
	tests := []struct {
		raw      string
		expected string
	}{
		{"http://ttytthanhhoa.vn", "http://ttytthanhhoa.vn"},
		{"ttytduchoa.com ", "ttytduchoa.com"},
		{"thptthienhoduong.edu.vn /", "thptthienhoduong.edu.vn"},
		{
			"URL: https://14.225.53.247:8443/\nUser: truongsavina.com\nPass: Ku#wRxhTv^u1m12h\n",
			"truongsavina.com",
		},
		{
			"thcsthuthua_edu_vn\n_P9M!5lbKs2jnilw",
			"thcsthuthua.edu.vn",
		},
		{"thptcanduoc", "thptcanduoc"},
		{"", ""},
	}

	for _, tt := range tests {
		target, _, _ := ExtractTarget(tt.raw)
		if target != tt.expected {
			t.Errorf("ExtractTarget(%q) = %q; want %q", tt.raw, target, tt.expected)
		}
	}
}
