package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// NewProductionHandler combines the session-scoped API with the compiled SPA.
// Health deliberately bypasses the API so container probes do not create labs.
func NewProductionHandler(api http.Handler, webRoot string) (http.Handler, error) {
	if api == nil {
		return nil, fmt.Errorf("new production handler: nil API")
	}
	if webRoot == "" {
		return nil, fmt.Errorf("new production handler: empty web root")
	}

	root := os.DirFS(webRoot)
	index, err := fs.ReadFile(root, "index.html")
	if err != nil {
		return nil, fmt.Errorf("new production handler: read index: %w", err)
	}
	files := http.FileServerFS(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws/lab" {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			api.ServeHTTP(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if info, statErr := fs.Stat(root, name); statErr == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
	}), nil
}
