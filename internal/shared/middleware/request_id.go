package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		c.Set(string(RequestIDKey), id)
		c.Writer.Header().Set(RequestIDHeader, id)

		ctx := context.WithValue(c.Request.Context(), RequestIDKey, id)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
