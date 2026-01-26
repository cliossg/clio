package middleware

import (
	"context"
	"net/http"
)

const (
	// LocaleKey is the context key for the locale.
	LocaleKey = contextKey("locale")

	// CSRFTokenKey is the context key for the CSRF token.
	CSRFTokenKey = contextKey("csrf_token")
)

// Locale injects the locale into the request context.
func Locale(defaultLocale string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := defaultLocale

			// Check Accept-Language header
			if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
				// Simple parsing - take the first language
				if len(acceptLang) >= 2 {
					locale = acceptLang[:2]
				}
			}

			// Check query parameter override
			if q := r.URL.Query().Get("lang"); q != "" {
				locale = q
			}

			ctx := context.WithValue(r.Context(), LocaleKey, locale)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetLocale extracts the locale from the context.
// Returns "en" if no locale is found.
func GetLocale(ctx context.Context) string {
	if ctx == nil {
		return "en"
	}
	if locale, ok := ctx.Value(LocaleKey).(string); ok {
		return locale
	}
	return "en"
}

// GetCSRFToken extracts the CSRF token from the context.
func GetCSRFToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if token, ok := ctx.Value(CSRFTokenKey).(string); ok {
		return token
	}
	return ""
}
