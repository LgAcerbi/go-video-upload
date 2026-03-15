package util

// NullIfEmpty returns nil if s is empty, otherwise returns s.
// Useful for SQL parameters where empty string should be stored as NULL.
func NullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
