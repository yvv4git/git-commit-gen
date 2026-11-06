package llm

import "regexp"

const (
	redactedValue        = "[REDACTED]"
	redactedPrivateKey   = "[REDACTED_PRIVATE_KEY]"
	redactedURLCredsTmpl = "${1}" + redactedValue + "@"
)

var (
	privateKeyPattern = regexp.MustCompile(
		`(?is)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	openAIKeyPattern      = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}\b`)
	githubTokenPattern    = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`)
	awsAccessKeyPattern   = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	jwtPattern            = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)
	bearerPattern         = regexp.MustCompile(`(?i)\b(Bearer\s+)[A-Za-z0-9._~+/=-]{8,}`)
	urlCredentialsPattern = regexp.MustCompile(
		`(https?://)[^/\s:@]+:[^/\s@]+@`)
	quotedSecretAssignmentPattern = regexp.MustCompile(
		`(?i)\b(api[_-]?key|secret|token|password|passwd|passw|authorization|private[_-]?key)\b(\s*[:=]\s*)(["'])[^"'\n]*(["'])`)
	secretAssignmentPattern = regexp.MustCompile(
		`(?i)\b(api[_-]?key|secret|token|password|passwd|passw|authorization|private[_-]?key)\b(\s*[:=]\s*)(["']?)([^\s"',]{4,})(["']?)`)
)

func redactSecrets(value string) string {
	value = privateKeyPattern.ReplaceAllString(value, redactedPrivateKey)
	value = openAIKeyPattern.ReplaceAllString(value, redactedValue)
	value = githubTokenPattern.ReplaceAllString(value, redactedValue)
	value = awsAccessKeyPattern.ReplaceAllString(value, redactedValue)
	value = jwtPattern.ReplaceAllString(value, redactedValue)
	value = bearerPattern.ReplaceAllString(value, "${1}"+redactedValue)
	value = urlCredentialsPattern.ReplaceAllString(value, redactedURLCredsTmpl)
	value = quotedSecretAssignmentPattern.ReplaceAllString(value, "${1}${2}${3}"+redactedValue+"${4}")
	value = secretAssignmentPattern.ReplaceAllString(value, "${1}${2}${3}"+redactedValue+"${5}")

	return value
}
