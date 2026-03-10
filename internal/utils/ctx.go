package utils

import "context"

type contextKey string

const RequestIDKey contextKey = "request_id"

func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(RequestIDKey).(string)
	return requestID
}
