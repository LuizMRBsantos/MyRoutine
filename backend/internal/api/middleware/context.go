package middleware

import "context"

// setContextValue is an internal helper to store values in context.
func setContextValue(ctx context.Context, key contextKey, value string) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetUserID extracts the authenticated user ID from the request context.
// Returns empty string if not found (should not happen in protected routes).
func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}
