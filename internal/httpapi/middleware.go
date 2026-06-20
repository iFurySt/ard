package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "request_id"

func requestIDMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		requestID := strings.TrimSpace(context.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		context.Set(requestIDKey, requestID)
		context.Header("X-Request-ID", requestID)
		context.Next()
	}
}

func jsonAccessLogMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		startedAt := time.Now()
		context.Next()
		event := map[string]any{
			"ts":        time.Now().UTC().Format(time.RFC3339Nano),
			"level":     "info",
			"event":     "http_request",
			"requestId": requestIDFromContext(context),
			"method":    context.Request.Method,
			"path":      context.Request.URL.Path,
			"status":    context.Writer.Status(),
			"latencyMs": time.Since(startedAt).Milliseconds(),
			"clientIp":  context.ClientIP(),
		}
		if len(context.Errors) > 0 {
			event["level"] = "error"
			event["errors"] = context.Errors.String()
		}
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintln(gin.DefaultWriter, string(data))
	}
}

func requestIDFromContext(context *gin.Context) string {
	value, ok := context.Get(requestIDKey)
	if !ok {
		return ""
	}
	requestID, _ := value.(string)
	return requestID
}
