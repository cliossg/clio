package ssg

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/cliossg/clio/pkg/cl/logger"
)

type contextKey string

const siteContextKey contextKey = "site"

// SiteContextMiddleware creates a middleware that loads the site from query param or form and puts it in context.
func SiteContextMiddleware(service Service, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			siteIDStr := r.URL.Query().Get("site_id")
			if siteIDStr == "" {
				contentType := r.Header.Get("Content-Type")
				if strings.HasPrefix(contentType, "multipart/form-data") {
					_ = r.ParseMultipartForm(32 << 20)
				} else {
					_ = r.ParseForm()
				}
				siteIDStr = r.FormValue("site_id")
			}
			if siteIDStr == "" {
				log.Errorf("site_id query param required: %s %s", r.Method, r.URL.Path)
				http.Error(w, "site_id query param required", http.StatusBadRequest)
				return
			}

			siteID, err := uuid.Parse(siteIDStr)
			if err != nil {
				log.Errorf("Invalid site_id '%s': %v", siteIDStr, err)
				http.Error(w, "Invalid site_id", http.StatusBadRequest)
				return
			}

			site, err := service.GetSite(r.Context(), siteID)
			if err != nil {
				log.Errorf("Site not found for id=%s: %v", siteID, err)
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
