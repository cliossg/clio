package ssg

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type contextKey string

const siteContextKey contextKey = "site"

// SiteContextMiddleware creates a middleware that loads the site from the URL and puts it in context.
func SiteContextMiddleware(service Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			siteIDStr := chi.URLParam(r, "siteID")
			if siteIDStr == "" {
				http.Error(w, "Site ID required", http.StatusBadRequest)
				return
			}

			siteID, err := uuid.Parse(siteIDStr)
			if err != nil {
				http.Error(w, "Invalid site ID", http.StatusBadRequest)
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
