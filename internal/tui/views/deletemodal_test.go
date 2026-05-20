package views

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeDestroyError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		stderr  string
		wantSub string
	}{
		{
			name:    "tofu binary missing in PATH",
			err:     errors.New(`exec: "tofu": executable file not found in $PATH`),
			stderr:  "",
			wantSub: "OpenTofu binary 'tofu' not found",
		},
		{
			name:    "tofu binary is the TOTP Manager imposter",
			err:     errors.New("exit status 1"),
			stderr:  "\x1b[1;1HTOTP Manager\nCreate master password",
			wantSub: "TOTP Manager",
		},
		{
			name:    "generic tofu failure surfaces first stderr line",
			err:     errors.New("exit status 1"),
			stderr:  "\nError: AWS credentials expired\n  on main.tf line 3\n",
			wantSub: "Error: AWS credentials expired",
		},
		{
			name:    "no stderr falls back to error string",
			err:     errors.New("exit status 2"),
			stderr:  "",
			wantSub: "exit status 2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := summarizeDestroyError(tc.err, tc.stderr)
			assert.Contains(t, got, tc.wantSub)
		})
	}
}

func TestFirstNonEmptyLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"skips blank lines", "\n\n  \nhello\nworld\n", 100, "hello"},
		{"truncates long line", strings.Repeat("a", 50), 10, "aaaaaaaaaa…"},
		{"empty input", "", 100, ""},
		{"only whitespace", "   \n\t\n", 100, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, firstNonEmptyLine(tc.in, tc.max))
		})
	}
}
