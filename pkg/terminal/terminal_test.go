package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatResult(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   string
	}{
		{
			name: "with backup",
			result: Result{
				CommitMessage:    "TASK-17: add guard\n\n- protect | input",
				BackupRef:        "refs/git-commit-gen/backup/1",
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				Requests:         3,
			},
			want: "┌────────────────┬──────────────────────────────────────┐\n" +
				"│ Field          │ Value                                │\n" +
				"├────────────────┼──────────────────────────────────────┤\n" +
				"│ Commit message │ TASK-17: add guard                   │\n" +
				"│                │                                      │\n" +
				"│                │ - protect | input                    │\n" +
				"│ Backup         │ refs/git-commit-gen/backup/1         │\n" +
				"│ Tokens         │ prompt: 10, completion: 5, total: 15 │\n" +
				"│ API requests   │ 3                                    │\n" +
				"└────────────────┴──────────────────────────────────────┘\n",
		},
		{
			name: "without backup",
			result: Result{
				CommitMessage: "TASK-17: add guard",
				Requests:      1,
			},
			want: "┌────────────────┬────────────────────────────────────┐\n" +
				"│ Field          │ Value                              │\n" +
				"├────────────────┼────────────────────────────────────┤\n" +
				"│ Commit message │ TASK-17: add guard                 │\n" +
				"│ Tokens         │ prompt: 0, completion: 0, total: 0 │\n" +
				"│ API requests   │ 1                                  │\n" +
				"└────────────────┴────────────────────────────────────┘\n",
		},
		{
			name: "ansi stripped",
			result: Result{
				CommitMessage:    "TASK-17: \x1b[31madd guard\x1b[0m",
				BackupRef:        "refs/git-commit-gen/backup/1",
				PromptTokens:     20,
				CompletionTokens: 10,
				TotalTokens:      30,
				Requests:         4,
			},
			want: "┌────────────────┬───────────────────────────────────────┐\n" +
				"│ Field          │ Value                                 │\n" +
				"├────────────────┼───────────────────────────────────────┤\n" +
				"│ Commit message │ TASK-17: add guard                    │\n" +
				"│ Backup         │ refs/git-commit-gen/backup/1          │\n" +
				"│ Tokens         │ prompt: 20, completion: 10, total: 30 │\n" +
				"│ API requests   │ 4                                     │\n" +
				"└────────────────┴───────────────────────────────────────┘\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatResult(tt.result))
		})
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"plain", "TASK-17: add guard", "TASK-17: add guard"},
		{"unicode", "TASK: добавить защиту", "TASK: добавить защиту"},
		{"ansi color", "TASK: \x1b[31mred\x1b[0m", "TASK: red"},
		{"ansi cursor", "TASK: \x1b[2J\x1b[1;1Hclear", "TASK: clear"},
		{"multiple ansi", "\x1b[1mbold\x1b[0m and \x1b[4munderline\x1b[0m", "bold and underline"},
		{"nul", "TASK: a\x00b", "TASK: ab"},
		{"bell", "TASK: a\x07b", "TASK: ab"},
		{"carriage return", "TASK: a\rb", "TASK: ab"},
		{"delete", "TASK: a\x7fb", "TASK: ab"},
		{"c1 control", "TASK: a\u009bb", "TASK: ab"},
		{"newline and tab", "TASK: a\n\tb", "TASK: a\n\tb"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Sanitize(tt.value))
		})
	}
}
