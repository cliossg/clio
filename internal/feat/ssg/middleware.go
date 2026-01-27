package ssg

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const siteContextKey contextKey = "site"

// SiteContextMiddleware creates a middleware that loads the site from query param and puts it in context.
func SiteContextMiddleware(service Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			siteIDStr := r.URL.Query().Get("site_id")
			if siteIDStr == "" {
				http.Error(w, "site_id query param required", http.StatusBadRequest)
				return
			}

			siteID, err := uuid.Parse(siteIDStr)
			if err != nil {
				http.Error(w, "Invalid site_id", http.StatusBadRequest)
				return
			}

			site, err := service.GetSite(r.Context(), siteID)
			if err != nil {
				http.Error(w, "Site not found", http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), siteContextKey, site)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// getSiteFromContext retrieves the site from context.
func getSiteFromContext(ctx context.Context) *Site {
	site, _ := ctx.Value(siteContextKey).(*Site)
	return site
}

// GetSiteFromContext retrieves the site from context (exported version).
func GetSiteFromContext(ctx context.Context) *Site {
	return getSiteFromContext(ctx)
}
