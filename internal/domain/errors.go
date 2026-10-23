package domain

import "errors"

var (
	ErrEmptyRules              = errors.New("got empty rules")
	ErrUnsupportedProxyType    = errors.New("unsupported proxy type")
	ErrNoResponseChoices       = errors.New("no response choices")
	ErrNoCommitsToUpdate       = errors.New("no commits to update")
	ErrOnBaseBranch            = errors.New("cannot update commit on base branch")
	ErrCommitNotUpdated        = errors.New("agent finished without updating the commit")
	ErrCommitAlreadyUpdated    = errors.New("commit already updated")
	ErrUnknownTool             = errors.New("unknown tool")
	ErrEmptyCommitMessage      = errors.New("got empty commit message")
	ErrToolCallWithoutFunction = errors.New("tool call without function")
)
