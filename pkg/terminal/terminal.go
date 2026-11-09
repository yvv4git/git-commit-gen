package terminal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

// Sanitize removes ANSI escape sequences and control characters from value
// before printing LLM-generated text to a terminal. Without this, a commit
// message from an untrusted diff could inject escape sequences that alter the
// terminal state, hide output, or spoof other messages.
func Sanitize(value string) string {
	value = ansiSequence.ReplaceAllString(value, "")

	var b strings.Builder
	b.Grow(len(value))

	for _, r := range value {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
		default:
			b.WriteRune(r)
		}
	}

	return b.String()
}

type Result struct {
	CommitMessage    string
	BackupRef        string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Requests         int
}

func FormatResult(result Result) string {
	message := strings.TrimRight(Sanitize(result.CommitMessage), "\n")
	backup := strings.TrimRight(Sanitize(result.BackupRef), "\n")

	tokens := "prompt: " + strconv.Itoa(result.PromptTokens) +
		", completion: " + strconv.Itoa(result.CompletionTokens) +
		", total: " + strconv.Itoa(result.TotalTokens)

	style := table.StyleLight
	style.Format.Header = text.FormatDefault

	writer := table.NewWriter()
	writer.SetStyle(style)
	writer.AppendHeader(table.Row{"Field", "Value"})
	writer.AppendRow(table.Row{"Commit message", message})

	if backup != "" {
		writer.AppendRow(table.Row{"Backup", backup})
	}

	writer.AppendRow(table.Row{"Tokens", tokens})
	writer.AppendRow(table.Row{"API requests", strconv.Itoa(result.Requests)})

	return writer.Render() + "\n"
}

func FormatCheck(provider, reply string) string {
	reply = strings.TrimRight(Sanitize(reply), "\n")

	return fmt.Sprintf("Provider %q is working.\n\nModel reply:\n%s\n", provider, reply)
}
