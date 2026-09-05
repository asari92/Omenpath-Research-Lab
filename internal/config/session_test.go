package config

import (
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
)

func TestDefault_MatchesSessionContract(t *testing.T) {
	values := map[string]any{"SessionTTL": 30 * 24 * time.Hour, "SessionRefreshInterval": 12 * time.Hour, "SessionCleanupInterval": time.Hour, "SessionTokenBytes": 32, "LabIDBytes": 16, "SessionCookieName": "omenpath_session"}
	for name, want := range values {
		t.Run(name, func(t *testing.T) {
			field := reflect.ValueOf(Default()).FieldByName(name)
			require.True(t, field.IsValid(), "missing session configuration %s", name)
			require.Equal(t, want, field.Interface())
		})
	}
}
