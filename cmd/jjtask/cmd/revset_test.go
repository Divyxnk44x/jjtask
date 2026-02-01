package cmd

import "testing"

func TestLooksLikeRevset(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// @ patterns
		{"@", true},
		{"@-", true},
		{"@-2", true},
		{"@--", true},

		// Operators
		{"::xyz", true},
		{"xyz::", true},
		{"x::y", true},
		{"x..y", true},
		{"~x", true},
		{"x & y", true},
		{"x | y", true},
		{"x+", true},
		{"x-", true},

		// Function calls
		{"root()", true},
		{"trunk()", true},
		{"mine()", true},
		{"ancestors(@)", true},
		{"descendants(xyz)", true},
		{"heads(all())", true},

		// Change IDs (short alphanumeric)
		{"abc", true},
		{"xyz123", true},
		{"k", true},
		{"abcdefghijkl", true}, // 12 chars max

		// NOT revsets (task titles)
		{"Fix the bug", false},
		{"Add feature", false},
		{"Update README", false},
		{"", false},
		{"abcdefghijklm", false}, // 13 chars - too long
		{"ABC", false},           // uppercase
		{"abc_def", false},       // underscore
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeRevset(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeRevset(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
