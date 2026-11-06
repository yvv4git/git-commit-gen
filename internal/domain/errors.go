package domain

import "errors"

var (
	ErrEmptyRules              = errors.New("got empty rules")
	ErrRulesNotFound           = errors.New("rules file not found") // LLM02: prevents reading config.toml with secrets as rules.
	ErrUnsupportedProxyType    = errors.New("unsupported proxy type")
	ErrInvalidProxyAddr        = errors.New("invalid proxy address")
	ErrInvalidAPIURL           = errors.New("invalid API URL")
	ErrInsecureAPIURL          = errors.New("insecure API URL: http is allowed only for localhost or with allowInsecureHttp") // LLM02, LLM03: prevents plaintext egress to an untrusted endpoint.
	ErrNoResponseChoices       = errors.New("no response choices")
	ErrNoCommitsToUpdate       = errors.New("no commits to update")
	ErrOnBaseBranch            = errors.New("cannot update commit on base branch") // LLM06, ASI02: blocks destructive history rewrite on the base branch.
	ErrCommitNotUpdated        = errors.New("agent finished without updating the commit")
	ErrCommitAlreadyUpdated    = errors.New("commit already updated")                                    // LLM06, ASI02: blocks repeated destructive tool calls.
	ErrFirstCommitMismatch     = errors.New("first commit message does not match the generated message") // LLM09: verifies applied output instead of trusting the agent.
	ErrUnknownTool             = errors.New("unknown tool")                                              // LLM06, ASI02: rejects tool calls outside the allowed set.
	ErrEmptyCommitMessage      = errors.New("got empty commit message")                                  // LLM05: validates generated output before use.
	ErrToolCallWithoutFunction = errors.New("tool call without function")                                // LLM05, ASI02: rejects malformed tool calls.
	ErrTooManyToolCalls        = errors.New("too many tool calls in one turn")                           // LLM10, ASI08: limits unbounded resource consumption.
	ErrCommitMessageTooLong    = errors.New("commit message too long")                                   // LLM05, LLM10: validates output and limits context growth.
	ErrCommitMessageControl    = errors.New("commit message contains control characters")                // LLM05: prevents terminal control injection.
	ErrCommitMessageNUL        = errors.New("commit message contains NUL")                               // LLM05: prevents malformed output.
	ErrCommitMessageANSI       = errors.New("commit message contains ANSI escape sequences")             // LLM05: prevents terminal escape injection.
	ErrForbiddenCommitTrailer  = errors.New("commit message contains forbidden trailer")                 // LLM05: rejects output that can change CI or authorship.
	ErrDiffTooLarge            = errors.New("diff too large")                                            // LLM10, ASI08: limits context size and request cost.
	ErrRulesTooLarge           = errors.New("rules too large")                                           // LLM10: limits system prompt size and request cost.
	ErrContextBudgetExceeded   = errors.New("context budget exceeded")                                   // LLM10, ASI08: stops unbounded context growth and spending.
	ErrAgentTurnsExceeded      = errors.New("agent turns exceeded")                                      // LLM10, ASI08: stops endless tool loops.
	ErrToolArguments           = errors.New("invalid tool arguments")
)
