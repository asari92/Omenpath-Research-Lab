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

func TestCookieSecure_RequiresExplicitSecureInProduction(t *testing.T) {
	for _, tc := range []struct {
		value                  string
		production, want, fail bool
	}{{"", false, false, false}, {"true", true, true, false}, {"false", false, false, false}, {"false", true, false, true}, {"", true, false, true}, {"wrong", false, false, true}} {
		got, err := CookieSecure(tc.value, tc.production)
		if tc.fail {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		}
	}
}
