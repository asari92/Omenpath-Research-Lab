package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/persistence"
	"sync"
	"testing"
	"time"
)

type fixedClock struct{ at time.Time }

func (c *fixedClock) Now() time.Time { return c.at }
func serviceStore(t *testing.T) *persistence.Store {
	t.Helper()
	s, err := persistence.Open(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	require.NoError(t, s.Migrate(context.Background()))
	return s
}

func TestService_TokenFormatReuseRefreshAndExpiry(t *testing.T) {
	s := serviceStore(t)
	c := &fixedClock{time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)}
	entropy := append(bytes.Repeat([]byte{0x21}, 48), bytes.Repeat([]byte{0x42}, 48)...)
	svc, err := New(s, config.Default(), c, bytes.NewReader(entropy))
	require.NoError(t, err)
	ctx := context.Background()
	first, err := svc.Resolve(ctx, "")
	require.NoError(t, err)
	require.True(t, first.Reissue)
	require.Len(t, first.Token, 43)
	decoded, err := base64.RawURLEncoding.DecodeString(first.Token)
	require.NoError(t, err)
	require.Equal(t, bytes.Repeat([]byte{0x21}, 32), decoded)
	require.Equal(t, persistence.LabID("21212121212121212121212121212121"), first.LabID)
	storedID, _, _, err := s.ResolveSession(ctx, sha256.Sum256([]byte(first.Token)), c.at, config.Default())
	require.NoError(t, err)
	require.Equal(t, first.LabID, storedID, "database lookup uses SHA256 of the encoded raw cookie")
	c.at = c.at.Add(12*time.Hour - time.Nanosecond)
	reuse, err := svc.Resolve(ctx, first.Token)
	require.NoError(t, err)
	require.Equal(t, first.LabID, reuse.LabID)
	require.Empty(t, reuse.Token)
	require.False(t, reuse.Reissue)
	require.Equal(t, first.ExpiresAt, reuse.ExpiresAt)
	c.at = c.at.Add(time.Nanosecond)
	renewed, err := svc.Resolve(ctx, first.Token)
	require.NoError(t, err)
	require.Equal(t, first.LabID, renewed.LabID)
	require.Empty(t, renewed.Token)
	require.True(t, renewed.Reissue)
	require.Equal(t, c.at.Add(30*24*time.Hour), renewed.ExpiresAt)
	c.at = renewed.ExpiresAt
	fresh, err := svc.Resolve(ctx, first.Token)
	require.NoError(t, err)
	require.NotEqual(t, first.LabID, fresh.LabID)
	require.NotEqual(t, first.Token, fresh.Token)
}

type brokenEntropy struct {
	n   int
	err error
}

func (r brokenEntropy) Read(p []byte) (int, error) { return r.n, r.err }
func TestService_RejectsShortEntropyWithoutLeakingErrorsOrPartialLabs(t *testing.T) {
	for _, reader := range []io.Reader{bytes.NewReader(make([]byte, 31)), bytes.NewReader(make([]byte, 47)), brokenEntropy{0, errors.New("raw-cookie-secret")}, brokenEntropy{1, nil}} {
		s := serviceStore(t)
		c := &fixedClock{time.Now().UTC()}
		svc, err := New(s, config.Default(), c, reader)
		require.NoError(t, err)
		result, err := svc.Resolve(context.Background(), "raw-cookie-secret")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "raw-cookie-secret")
		require.Empty(t, result.Token)
		labs, err := s.ExpiredLabs(context.Background(), c.at.Add(31*24*time.Hour))
		require.NoError(t, err)
		require.Empty(t, labs)
	}
}

func TestService_InvalidAndUnknownTokensCreateFreshLabs(t *testing.T) {
	for _, token := range []string{"garbage", "", base64.URLEncoding.EncodeToString(make([]byte, 32)), base64.RawURLEncoding.EncodeToString(make([]byte, 32))} {
		t.Run(token, func(t *testing.T) {
			svc, err := New(serviceStore(t), config.Default(), &fixedClock{time.Now().UTC()}, rand.Reader)
			require.NoError(t, err)
			result, err := svc.Resolve(context.Background(), token)
			require.NoError(t, err)
			require.True(t, result.Reissue)
			require.NotEmpty(t, result.LabID)
			require.NotEqual(t, token, result.Token)
		})
	}
}

func TestService_ConcurrentResolutionsCreateCompleteIsolatedLabs(t *testing.T) {
	s := serviceStore(t)
	svc, err := New(s, config.Default(), &fixedClock{time.Now().UTC()}, rand.Reader)
	require.NoError(t, err)
	var wg sync.WaitGroup
	type outcome struct {
		result Resolution
		err    error
	}
	results := make(chan outcome, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); r, err := svc.Resolve(context.Background(), ""); results <- outcome{r, err} }()
	}
	wg.Wait()
	close(results)
	ids := map[persistence.LabID]bool{}
	tokens := map[string]bool{}
	for o := range results {
		require.NoError(t, o.err)
		require.False(t, ids[o.result.LabID])
		require.False(t, tokens[o.result.Token])
		ids[o.result.LabID] = true
		tokens[o.result.Token] = true
		repo, err := s.ForLab(o.result.LabID)
		require.NoError(t, err)
		snap, err := repo.Load(context.Background())
		require.NoError(t, err)
		require.Len(t, snap.Simulation.Planes, 85)
		require.Len(t, snap.Simulation.Observers, 20)
	}
}
