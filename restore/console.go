package restore

// Console interface for output (injected from CLI).
type Console interface {
	Printf(format string, args ...any)
	Error(format string, args ...any)
	Warning(format string, args ...any)
}
