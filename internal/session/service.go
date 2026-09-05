// Package session resolves opaque anonymous laboratory cookies. HTTP emission
// and runtime lifetime management belong to the transport integration boundary.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/persistence"
)

type Resolution struct {
	LabID     persistence.LabID
	ExpiresAt time.Time
	// Token is populated only for creation. On refresh the caller reuses the
	// incoming cookie value transiently. Neither result is a browser DTO.
	Token   string
	Reissue bool
}

type Service struct {
	store     *persistence.Store
	cfg       config.Config
	clock     clock.Clock
	entropy   io.Reader
	entropyMu sync.Mutex
}

// New defaults entropy to crypto/rand.Reader; tests may inject deterministic
// bytes. The mutex permits even non-concurrent injected readers to be used safely.
func New(store *persistence.Store, cfg config.Config, c clock.Clock, entropy io.Reader) (*Service, error) {
	if store == nil || c == nil || cfg.SessionTTL <= 0 || cfg.SessionRefreshInterval <= 0 || cfg.SessionRefreshInterval >= cfg.SessionTTL || cfg.SessionTokenBytes != 32 || cfg.LabIDBytes != 16 {
		return nil, fmt.Errorf("invalid session configuration")
	}
	if entropy == nil {
		entropy = rand.Reader
	}
	return &Service{store: store, cfg: cfg, clock: c, entropy: entropy}, nil
}

func (s *Service) Resolve(ctx context.Context, token string) (Resolution, error) {
	now := s.clock.Now().UTC()
	if len(token) == base64.RawURLEncoding.EncodedLen(s.cfg.SessionTokenBytes) {
		decoded, err := base64.RawURLEncoding.Strict().DecodeString(token)
		if err == nil && len(decoded) == s.cfg.SessionTokenBytes {
			id, expiry, refresh, err := s.store.ResolveSession(ctx, sha256.Sum256([]byte(token)), now, s.cfg)
			if err != nil {
				return Resolution{}, fmt.Errorf("resolve laboratory session failed")
			}
			if id != "" {
				return Resolution{LabID: id, ExpiresAt: expiry, Reissue: refresh}, nil
			}
		}
	}
	// Require each entropy call to fill its complete buffer; silently accepting a
	// short read could reduce token entropy or create partially fixed identifiers.
	tokenBytes := make([]byte, s.cfg.SessionTokenBytes)
	idBytes := make([]byte, s.cfg.LabIDBytes)
	s.entropyMu.Lock()
	n, err := s.entropy.Read(tokenBytes)
	if err == nil && n == len(tokenBytes) {
		n, err = s.entropy.Read(idBytes)
		if n != len(idBytes) {
			err = io.ErrUnexpectedEOF
		}
	} else if err == nil {
		err = io.ErrUnexpectedEOF
	}
	s.entropyMu.Unlock()
	if err != nil {
		return Resolution{}, fmt.Errorf("generate laboratory session entropy failed")
	}
	raw := base64.RawURLEncoding.EncodeToString(tokenBytes)
	id := persistence.LabID(hex.EncodeToString(idBytes))
	if err := s.store.CreateSession(ctx, id, sha256.Sum256([]byte(raw)), now, s.cfg); err != nil {
		return Resolution{}, fmt.Errorf("create laboratory session failed")
	}
	return Resolution{LabID: id, ExpiresAt: now.Add(s.cfg.SessionTTL), Token: raw, Reissue: true}, nil
}
