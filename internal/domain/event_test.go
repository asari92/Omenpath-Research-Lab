package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestEventDraftValidate_AcceptsCanonicalEvent(t *testing.T) {
	portalID := int64(7)
	draft := domain.EventDraft{
		EventType:   domain.EventPortalOpened,
		PortalID:    &portalID,
		Message:     "Portal opened",
		PayloadJSON: `{"kind":"NATURAL"}`,
		CreatedAt:   testutil.BaseTime,
	}

	require.NoError(t, draft.Validate())
	assert.True(t, domain.IsEventType(domain.EventPortalOpened))
}

func TestEventDraftValidate_RejectsUnknownType(t *testing.T) {
	draft := canonicalEventDraft()
	draft.EventType = domain.EventType("UNKNOWN")

	require.Error(t, draft.Validate())
	assert.False(t, domain.IsEventType(draft.EventType))
}

func TestEventDraftValidate_RequiresJSONObjectPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "missing", payload: ""},
		{name: "malformed", payload: "{"},
		{name: "array", payload: `[]`},
		{name: "scalar", payload: `"value"`},
		{name: "null", payload: `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draft := canonicalEventDraft()
			draft.PayloadJSON = tt.payload
			require.Error(t, draft.Validate())
		})
	}
}

func TestEventDraftValidate_RequiresCanonicalFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.EventDraft)
	}{
		{name: "empty message", mutate: func(d *domain.EventDraft) { d.Message = "" }},
		{name: "zero time", mutate: func(d *domain.EventDraft) { d.CreatedAt = time.Time{} }},
		{name: "non UTC time", mutate: func(d *domain.EventDraft) {
			d.CreatedAt = d.CreatedAt.In(time.FixedZone("offset", 2*60*60))
		}},
		{name: "non positive portal", mutate: func(d *domain.EventDraft) { value := int64(0); d.PortalID = &value }},
		{name: "non positive observer", mutate: func(d *domain.EventDraft) { value := int64(-1); d.ObserverID = &value }},
		{name: "non positive plane", mutate: func(d *domain.EventDraft) { value := int64(0); d.PlaneID = &value }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draft := canonicalEventDraft()
			tt.mutate(&draft)
			require.Error(t, draft.Validate())
		})
	}
}

func TestEventValidatePersisted_RequiresPositiveID(t *testing.T) {
	draft := canonicalEventDraft()
	event := domain.Event{
		ID:          1,
		EventType:   draft.EventType,
		PortalID:    draft.PortalID,
		ObserverID:  draft.ObserverID,
		PlaneID:     draft.PlaneID,
		Message:     draft.Message,
		PayloadJSON: draft.PayloadJSON,
		CreatedAt:   draft.CreatedAt,
	}
	require.NoError(t, event.ValidatePersisted())

	event.ID = 0
	require.Error(t, event.ValidatePersisted())
}

func TestPortalHistory_FiltersAndKeepsChronology(t *testing.T) {
	portalOne := int64(1)
	portalTwo := int64(2)
	observerID := int64(4)
	later := testutil.BaseTime.Add(time.Second)
	events := []domain.Event{
		{ID: 3, EventType: domain.EventPortalClosed, PortalID: &portalOne, ObserverID: &observerID, Message: "third", PayloadJSON: `{}`, CreatedAt: later},
		{ID: 2, EventType: domain.EventRiskLevelChanged, PortalID: &portalOne, Message: "second", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime},
		{ID: 4, EventType: domain.EventPortalOpened, PortalID: &portalTwo, Message: "other", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime},
		{ID: 1, EventType: domain.EventPortalOpened, PortalID: &portalOne, Message: "first", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime},
	}

	history := domain.PortalHistory(events, portalOne)
	require.Len(t, history, 3)
	assert.Equal(t, []int64{1, 2, 3}, []int64{history[0].ID, history[1].ID, history[2].ID})

	*history[0].PortalID = 99
	*history[2].ObserverID = 99
	assert.Equal(t, int64(1), *events[3].PortalID)
	assert.Equal(t, int64(4), *events[0].ObserverID)
}

func TestNewActionRejectedEvent_EncodesDomainCause(t *testing.T) {
	portalID := int64(9)
	cause := errors.New("confirmation required")

	draft, err := domain.NewActionRejectedEvent(
		testutil.BaseTime,
		"close_portal",
		&portalID,
		nil,
		nil,
		cause,
	)
	require.NoError(t, err)
	require.NoError(t, draft.Validate())
	assert.Equal(t, domain.EventActionRejected, draft.EventType)
	assert.Equal(t, &portalID, draft.PortalID)
	assert.JSONEq(t, `{"action":"close_portal","cause":"confirmation required"}`, draft.PayloadJSON)
}

func canonicalEventDraft() domain.EventDraft {
	portalID := int64(1)
	return domain.EventDraft{
		EventType:   domain.EventPortalOpened,
		PortalID:    &portalID,
		Message:     "Portal opened",
		PayloadJSON: `{}`,
		CreatedAt:   testutil.BaseTime,
	}
}
