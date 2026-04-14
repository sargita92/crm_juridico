package application

import "testing"

func TestIsResetCommand(t *testing.T) {
	cases := []struct {
		name     string
		messages []string
		want     bool
	}{
		{"single exact", []string{"/reset"}, true},
		{"single trimmed", []string{"  /reset  "}, true},
		{"reset inside batch", []string{"oi", "/reset"}, true},
		{"empty batch", []string{}, false},
		{"not a command", []string{"resetando minha conta"}, false},
		{"case mismatch", []string{"/RESET"}, false},
		{"slash reset with text", []string{"/reset please"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsResetCommand(tc.messages); got != tc.want {
				t.Errorf("IsResetCommand(%v) = %v, want %v", tc.messages, got, tc.want)
			}
		})
	}
}
