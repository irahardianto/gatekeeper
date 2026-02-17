package parser

// hintDatabase maps rule IDs to concise, actionable fix hints.
// Keys are rule IDs as they appear in StructuredError.Rule.
var hintDatabase = map[string]string{
	// --- Go: gosec ---
	"G101": "Use environment variables or a secret manager instead of hardcoded credentials.",
	"G102": "Bind to a specific IP address instead of 0.0.0.0 to limit network exposure.",
	"G103": "Avoid unsafe.Pointer unless absolutely necessary; prefer safe alternatives.",
	"G104": "Always check returned errors — unhandled errors hide failures.",
	"G107": "Validate or sanitize URLs before making HTTP requests to prevent SSRF.",
	"G110": "Limit the size of decompressed data to prevent zip bomb attacks.",
	"G201": "Use parameterized queries to prevent SQL injection.",
	"G202": "Use parameterized queries instead of string concatenation for SQL.",
	"G204": "Validate and sanitize arguments before passing to exec.Command.",
	"G301": "Use restrictive directory permissions (0750 or less).",
	"G302": "Use restrictive file permissions (0600 or 0644).",
	"G303": "Use os.CreateTemp instead of predictable temp file names.",
	"G304": "Validate file paths against a known-safe base directory before opening.",
	"G306": "Use restrictive permissions when writing files (0600 or 0644).",
	"G401": "Use SHA-256 or SHA-3 instead of weak hash algorithms (MD5/SHA1).",
	"G501": "Import crypto/sha256 or crypto/sha3 instead of weak hash packages.",

	// --- Go: staticcheck ---
	"S1000":  "Use a plain channel send/receive instead of a single-case select.",
	"S1001":  "Replace the loop with copy().",
	"S1003":  "Use strings.Contains instead of strings.Index to check for substrings.",
	"S1005":  "Drop the blank identifier from the range; it is unnecessary.",
	"S1023":  "Omit redundant return/break at the end of a function/case block.",
	"S1025":  "Use the value directly instead of fmt.Sprintf(\"%s\", x).",
	"S1028":  "Use fmt.Errorf instead of errors.New(fmt.Sprintf(...)).",
	"SA1019": "This API is deprecated — check the documentation for the replacement.",
	"SA4006": "This value is assigned but never used.",
	"SA5001": "Defer the Close call to ensure the resource is always released.",
	"ST1003": "Use MixedCaps (Go naming convention) instead of underscores.",

	// --- Go: common linters ---
	"errcheck":    "Always handle returned errors with 'if err != nil'.",
	"ineffassign": "Remove the assignment — the variable is reassigned before it is read.",
	"govet":       "Fix the issue reported by go vet — it usually indicates a real bug.",

	// --- JavaScript/TypeScript: ESLint ---
	"no-unused-vars":            "Remove the unused variable, or prefix with _ if intentionally unused.",
	"no-undef":                  "Declare the variable or import it before use.",
	"no-console":                "Remove console.log statements or use a proper logger.",
	"eqeqeq":                    "Use === and !== instead of == and != for strict equality.",
	"no-var":                    "Use let or const instead of var.",
	"prefer-const":              "Use const for variables that are never reassigned.",
	"no-async-promise-executor": "Remove async from the Promise executor — throw will silently fail.",

	// --- Python: ruff / flake8 ---
	"E501": "Break long lines to improve readability (default limit: 88 or 120 chars).",
	"F401": "Remove the unused import.",
	"F811": "Remove the redefined variable — it shadows an earlier definition.",
	"F841": "Remove the unused variable assignment.",
	"E712": "Use 'is' / 'is not' for comparisons to True/False/None.",
	"W291": "Remove trailing whitespace.",

	// --- Python: bandit ---
	"B101": "Avoid assert in production code — it is stripped with python -O.",
	"B105": "Do not hardcode passwords — use environment variables or a secret manager.",
	"B108": "Avoid hardcoded /tmp paths — use tempfile.mkdtemp() instead.",
	"B301": "Avoid pickle — it can execute arbitrary code during deserialization.",
	"B608": "Use parameterized queries to prevent SQL injection.",
}

// EnrichHints populates the Hint field of each StructuredError from
// the static hint database. Pre-existing hints are preserved.
func EnrichHints(errors []StructuredError) []StructuredError {
	for i := range errors {
		if errors[i].Hint != "" {
			continue
		}
		if hint, ok := hintDatabase[errors[i].Rule]; ok {
			errors[i].Hint = hint
		}
	}
	return errors
}
