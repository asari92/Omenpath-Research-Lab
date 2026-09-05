package httpapi

import (
	"context"
	"net/http"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/labruntime"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/realtime"
	"omenpath-lab/internal/session"
)

type SessionResolver interface {
	Resolve(context.Context, string) (session.Resolution, error)
}
type scopeKey struct{}
type requestScope struct {
	labID   persistence.LabID
	lease   *labruntime.Lease
	runtime *labruntime.Runtime
	manager Manager
	hub     *realtime.Hub
}

func requestManager(r *http.Request) Manager {
	return r.Context().Value(scopeKey{}).(*requestScope).manager
}

func sessionMiddleware(resolver SessionResolver, registry *labruntime.Registry, cfg config.Config, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if cookie, err := r.Cookie(cfg.SessionCookieName); err == nil {
				token = cookie.Value
			}
			resolved, err := resolver.Resolve(r.Context(), token)
			if err != nil {
				writeInternal(w)
				return
			}
			lease, err := registry.Acquire(r.Context(), resolved.LabID)
			if err != nil {
				writeInternal(w)
				return
			}
			defer lease.Release()
			if resolved.Reissue {
				if resolved.Token != "" {
					token = resolved.Token
				}
				http.SetCookie(w, &http.Cookie{Name: cfg.SessionCookieName, Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: int(cfg.SessionTTL.Seconds()), Expires: resolved.ExpiresAt})
			}
			scope := &requestScope{labID: resolved.LabID, lease: lease, runtime: lease.Runtime, manager: lease.Runtime.Manager, hub: lease.Runtime.Hub}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, scope)))
		})
	}
}
