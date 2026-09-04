package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"omenpath-lab/internal/domain"
)

func encodeTime(value time.Time, field string) (int64, error) {
	if value.IsZero() || value.Location() != time.UTC {
		return 0, fmt.Errorf("%s must be a non-zero UTC timestamp", field)
	}
	return value.UnixNano(), nil
}

func encodeOptionalTime(value *time.Time, field string) (any, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := encodeTime(*value, field)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func decodeTime(value int64) time.Time {
	return time.Unix(0, value).UTC()
}

func decodeRequiredTime(value int64, field string) (time.Time, error) {
	if value == 0 {
		return time.Time{}, fmt.Errorf("%s must be a non-zero Unix nanosecond timestamp", field)
	}
	return decodeTime(value), nil
}

func decodeOptionalTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	decoded := decodeTime(value.Int64)
	return &decoded
}

func encodeOptionalInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func decodeOptionalInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	decoded := value.Int64
	return &decoded
}

func encodeAliases(aliases []string) (string, error) {
	if aliases == nil {
		aliases = []string{}
	}
	encoded, err := json.Marshal(aliases)
	if err != nil {
		return "", fmt.Errorf("encode plane aliases: %w", err)
	}
	return string(encoded), nil
}

func decodeAliases(encoded string) ([]string, error) {
	var aliases []string
	if err := json.Unmarshal([]byte(encoded), &aliases); err != nil || aliases == nil {
		return nil, fmt.Errorf("invalid plane aliases JSON")
	}
	return aliases, nil
}

func validatePlane(plane domain.Plane) error {
	if plane.ID <= 0 || strings.TrimSpace(plane.Name) == "" || strings.TrimSpace(plane.CatalogTier) == "" ||
		plane.Explored != (plane.ExploredAt != nil) {
		return fmt.Errorf("invalid plane %d", plane.ID)
	}
	if plane.ExploredAt != nil {
		if _, err := encodeTime(*plane.ExploredAt, "plane.explored_at"); err != nil {
			return fmt.Errorf("invalid plane %d: %w", plane.ID, err)
		}
	}
	_, err := encodeAliases(plane.Aliases)
	return err
}

func validPortalKind(value domain.PortalKind) bool {
	return value == domain.PortalKindNatural || value == domain.PortalKindExtraction
}

func validPortalStability(value domain.PortalStability) bool {
	return value == domain.PortalStable || value == domain.PortalUnstable
}

func validPortalFlow(value domain.PortalFlow) bool {
	return value == domain.PortalFlowNone || value == domain.PortalFlowOutbound || value == domain.PortalFlowInbound
}

func validPortalStatus(value domain.PortalStatus) bool {
	return value == domain.PortalStatusOpen || value == domain.PortalStatusClosed || value == domain.PortalStatusCollapsed
}

func validTerminationReason(value domain.TerminationReason) bool {
	switch value {
	case domain.TerminationNone, domain.TerminationNaturalClose, domain.TerminationManualClose,
		domain.TerminationEnergyDepleted, domain.TerminationInstability:
		return true
	default:
		return false
	}
}

func validatePortal(portal domain.Portal) error {
	if portal.ID <= 0 || strings.TrimSpace(portal.Name) == "" || portal.SlotIndex < 1 || portal.SlotIndex > 7 ||
		portal.DestinationPlaneID <= 0 || portal.EnergyBase < 0 || portal.EnergyDecayRate <= 0 ||
		portal.CreaturesInitial < 0 || !validPortalKind(portal.Kind) ||
		!validPortalStability(portal.Stability) || !validPortalFlow(portal.ObserverFlow) ||
		!validPortalStatus(portal.Status) || !validTerminationReason(portal.TerminationReason) {
		return fmt.Errorf("invalid portal %d", portal.ID)
	}
	for field, value := range map[string]time.Time{
		"portal.energy_base_at":     portal.EnergyBaseAt,
		"portal.opened_at":          portal.OpenedAt,
		"portal.scheduled_close_at": portal.ScheduledCloseAt,
		"portal.created_at":         portal.CreatedAt,
		"portal.updated_at":         portal.UpdatedAt,
	} {
		if _, err := encodeTime(value, field); err != nil {
			return fmt.Errorf("invalid portal %d: %w", portal.ID, err)
		}
	}
	for field, value := range map[string]*time.Time{
		"portal.instability_collapse_at":    portal.InstabilityCollapseAt,
		"portal.extraction_synchronized_at": portal.ExtractionSynchronizedAt,
		"portal.closed_at":                  portal.ClosedAt,
	} {
		if _, err := encodeOptionalTime(value, field); err != nil {
			return fmt.Errorf("invalid portal %d: %w", portal.ID, err)
		}
	}
	if portal.Status == domain.PortalStatusOpen {
		if portal.ClosedAt != nil || portal.TerminationReason != domain.TerminationNone {
			return fmt.Errorf("invalid open portal %d", portal.ID)
		}
	} else if portal.ClosedAt == nil || portal.TerminationReason == domain.TerminationNone {
		return fmt.Errorf("invalid terminal portal %d", portal.ID)
	}
	return nil
}

func validObserverStatus(value domain.ObserverStatus) bool {
	switch value {
	case domain.ObserverAvailable, domain.ObserverOutbound, domain.ObserverExploring,
		domain.ObserverWaitingReturn, domain.ObserverReturning, domain.ObserverLost:
		return true
	default:
		return false
	}
}

func validateObserver(observer domain.Observer) error {
	if observer.ID <= 0 || !validObserverStatus(observer.Status) {
		return fmt.Errorf("invalid observer %d", observer.ID)
	}
	if _, err := encodeTime(observer.CreatedAt, "observer.created_at"); err != nil {
		return fmt.Errorf("invalid observer %d: %w", observer.ID, err)
	}
	if _, err := encodeTime(observer.UpdatedAt, "observer.updated_at"); err != nil {
		return fmt.Errorf("invalid observer %d: %w", observer.ID, err)
	}
	if _, err := encodeOptionalTime(observer.PhaseStartedAt, "observer.phase_started_at"); err != nil {
		return fmt.Errorf("invalid observer %d: %w", observer.ID, err)
	}
	if _, err := encodeOptionalTime(observer.PhaseEndsAt, "observer.phase_ends_at"); err != nil {
		return fmt.Errorf("invalid observer %d: %w", observer.ID, err)
	}
	return nil
}

func validAppMode(value domain.AppMode) bool {
	return value == domain.ModeTutorial || value == domain.ModeLive
}
