package session

import (
	"omenpath-lab/internal/persistence"
	"reflect"
	"testing"
)

func TestSessionBoundary_HasTransactionalPersistence(t *testing.T) {
	for _, name := range []string{"CreateSession", "ResolveSession", "ExpiredLabs", "DeleteExpiredLab"} {
		if _, ok := reflect.TypeOf((*persistence.Store)(nil)).MethodByName(name); !ok {
			t.Errorf("session boundary requires %s", name)
		}
	}
}
