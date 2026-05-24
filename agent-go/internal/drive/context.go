package drive

import "context"

type ctxKey string

const userKey ctxKey = "td.user_id"

func WithUser(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return context.WithValue(ctx, userKey, userID)
}

func UserFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userKey).(string)
	return value
}
