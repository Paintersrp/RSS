package httpx

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"courier/internal/logx"
	"courier/internal/store"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// HTTPErrorHandler returns an Echo error handler that emits uniform JSON responses
// and logs errors with request context.
func HTTPErrorHandler(service string) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if err == nil {
			return
		}

		req := c.Request()
		status := http.StatusInternalServerError
		message := http.StatusText(status)
		originalErr := err

		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.Code > 0 {
				status = httpErr.Code
			}
			if httpErr.Message != nil {
				message = httpErrorMessage(httpErr.Message)
			}
			if httpErr.Internal != nil {
				originalErr = httpErr.Internal
			}
		}

		switch {
		case errors.Is(originalErr, store.ErrFeedExists):
			status = http.StatusConflict
			message = store.ErrFeedExists.Error()
		case errors.Is(originalErr, sql.ErrNoRows):
			status = http.StatusNotFound
			message = "resource not found"
		case errors.Is(err, echo.ErrNotFound) || errors.Is(originalErr, echo.ErrNotFound):
			status = http.StatusNotFound
			message = "route not found"
		}

		if status == http.StatusInternalServerError {
			message = http.StatusText(status)
		}

		payload := errorEnvelope{Error: errorBody{Message: message, Code: statusCodeToErrorCode(status)}}

		logx.Error(service, "http request failed", originalErr, map[string]any{
			"method": req.Method,
			"path":   req.URL.Path,
			"status": status,
		})

		if c.Response().Committed {
			return
		}

		res := c.Response()
		res.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		res.WriteHeader(status)

		if encodeErr := json.NewEncoder(res).Encode(payload); encodeErr != nil {
			logx.Error(service, "write error response", encodeErr, map[string]any{
				"method": req.Method,
				"path":   req.URL.Path,
				"status": status,
			})
		}
	}
}

func httpErrorMessage(msg any) string {
	switch v := msg.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return fmt.Sprint(v)
	}
}

func statusCodeToErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusInternalServerError:
		return "internal_error"
	default:
		text := http.StatusText(status)
		text = strings.ToLower(strings.ReplaceAll(text, " ", "_"))
		if text == "" {
			return "error"
		}
		return text
	}
}
