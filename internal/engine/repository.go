package engine

import (
	"context"
	"errors"

	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
)

// Repository is the persistence boundary owned by LabManager. Implementations
// must commit the snapshot and event drafts in one transaction.
type Repository interface {
	Load(context.Context) (persistence.Snapshot, error)
	Commit(context.Context, persistence.Snapshot, []domain.EventDraft) ([]domain.Event, error)
	ListEvents(context.Context, *int64) ([]domain.Event, error)
	ResetTutorial(context.Context, persistence.Snapshot) error
}

// Stable engine errors intentionally carry no HTTP semantics. Stage 12 maps
// them at the transport boundary.
var (
	ErrPortalNotFound        = errors.New("portal not found")
	ErrPlaneNotFound         = errors.New("plane not found")
	ErrTutorialNotReady      = errors.New("tutorial not ready")
	ErrInvalidTutorialSignal = errors.New("invalid tutorial signal")
)
