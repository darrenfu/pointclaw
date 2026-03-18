package types

import "testing"

func TestCabinName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"FIRST", "business"},
		{"BUSINESS", "business"},
		{"MAIN", "economy"},
		{"COACH", "economy"},
		{"SAVER", "economy"},
		{"UNKNOWN", "economy"},
		{"", "economy"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CabinName(tt.input)
			if got != tt.want {
				t.Errorf("CabinName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsSaverCabin(t *testing.T) {
	if !IsSaverCabin("SAVER") {
		t.Error("expected SAVER to be saver cabin")
	}
	if IsSaverCabin("MAIN") {
		t.Error("expected MAIN to not be saver cabin")
	}
	if IsSaverCabin("FIRST") {
		t.Error("expected FIRST to not be saver cabin")
	}
}
