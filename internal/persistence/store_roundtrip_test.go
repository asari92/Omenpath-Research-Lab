package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
)

func completeSnapshot(now time.Time) Snapshot {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{
			ID:          int64(i + 1),
			Name:        fmt.Sprintf("Plane %02d", i+1),
			Aliases:     []string{},
			CatalogTier: "core",
		}
	}
	exploredAt := now.Add(-time.Minute)
	planes[1].Aliases = []string{"Second World", "World Two"}
	planes[1].CatalogTier = "expanded"
	planes[1].Explored = true
	planes[1].ExploredAt = &exploredAt

	openedAt := now.Add(-30 * time.Second)
	energyAt := now.Add(-10 * time.Second)
	instabilityAt := now.Add(20 * time.Second)
	syncAt := openedAt.Add(5 * time.Second)
	closedAt := now.Add(-5 * time.Second)
	portals := []domain.Portal{
		{
			ID:                    7,
			Name:                  "Aperture 7",
			SlotIndex:             1,
			Kind:                  domain.PortalKindNatural,
			DestinationPlaneID:    1,
			EnergyBase:            73.25,
			EnergyBaseAt:          energyAt,
			EnergyDecayRate:       0.45,
			Stability:             domain.PortalUnstable,
			InstabilityCollapseAt: &instabilityAt,
			OpenedAt:              openedAt,
			ScheduledCloseAt:      now.Add(2 * time.Minute),
			CreaturesInitial:      3,
			ObserverFlow:          domain.PortalFlowOutbound,
			Status:                domain.PortalStatusOpen,
			TerminationReason:     domain.TerminationNone,
			CreatedAt:             openedAt,
			UpdatedAt:             energyAt,
		},
		{
			ID:                       8,
			Name:                     "Extraction 8",
			SlotIndex:                2,
			Kind:                     domain.PortalKindExtraction,
			DestinationPlaneID:       2,
			EnergyBase:               80,
			EnergyBaseAt:             openedAt,
			EnergyDecayRate:          0.2,
			Stability:                domain.PortalStable,
			OpenedAt:                 openedAt,
			ScheduledCloseAt:         now.Add(time.Minute),
			ObserverFlow:             domain.PortalFlowInbound,
			ExtractionSynchronizedAt: &syncAt,
			Status:                   domain.PortalStatusClosed,
			TerminationReason:        domain.TerminationManualClose,
			ClosedAt:                 &closedAt,
			CreatedAt:                openedAt,
			UpdatedAt:                closedAt,
		},
	}

	observers := make([]domain.Observer, 10)
	for i := range observers {
		observers[i] = domain.NewObserver(int64(i+1), openedAt)
		observers[i].UpdatedAt = now
	}
	phaseStart := now.Add(-3 * time.Second)
	phaseEnd := now.Add(12 * time.Second)
	activePortalID := int64(7)
	observers[0].Status = domain.ObserverOutbound
	observers[0].ActivePortalID = &activePortalID
	observers[0].PhaseStartedAt = &phaseStart
	observers[0].PhaseEndsAt = &phaseEnd

	scheduledAt := now.Add(-2 * time.Second)
	dueAt := now.Add(8 * time.Second)
	lastTickAt := now
	overrideUntil := now.Add(17 * time.Second)
	return Snapshot{
		Simulation: domain.SimulationState{
			Lab: domain.LabState{
				EnergyBase:           42,
				EnergyBaseAt:         now,
				LeylineOverrideUntil: &overrideUntil,
			},
			Portals:      portals,
			Planes:       planes,
			Observers:    observers,
			NextPortalID: 9,
			NaturalSpawn: domain.NaturalSpawnState{
				ScheduledAt: &scheduledAt,
				DueAt:       &dueAt,
			},
			LastTickAt: &lastTickAt,
		},
		App: domain.AppState{Mode: domain.ModeLive, TutorialStep: 9},
	}
}

func TestStoreCommitAndLoad_RoundTripsCompleteSnapshot(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	want := completeSnapshot(time.Date(2026, 9, 4, 12, 0, 0, 123, time.UTC))

	_, err := store.Commit(ctx, want, nil)
	require.NoError(t, err)
	got, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestStoreCommit_PreservesExtractionAndOverrideTimestamps(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	want := completeSnapshot(time.Date(2026, 9, 4, 12, 10, 0, 456, time.UTC))

	_, err := store.Commit(ctx, want, nil)
	require.NoError(t, err)
	got, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, want.Simulation.Lab.LeylineOverrideUntil, got.Simulation.Lab.LeylineOverrideUntil)
	require.Equal(t, want.Simulation.Portals[1].ExtractionSynchronizedAt, got.Simulation.Portals[1].ExtractionSynchronizedAt)
}

func TestStoreCommit_PersistsSchedulerAndNextPortalID(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	want := completeSnapshot(time.Date(2026, 9, 4, 12, 20, 0, 789, time.UTC))

	_, err := store.Commit(ctx, want, nil)
	require.NoError(t, err)
	got, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, want.Simulation.NextPortalID, got.Simulation.NextPortalID)
	require.Equal(t, want.Simulation.NaturalSpawn, got.Simulation.NaturalSpawn)
	require.Equal(t, want.Simulation.LastTickAt, got.Simulation.LastTickAt)
}

func TestStoreLoad_RecoversOverdueStateWithoutResolvingIt(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	want := completeSnapshot(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	want.Simulation.Portals[0].ScheduledCloseAt = want.Simulation.LastTickAt.Add(time.Second)
	want.Simulation.Portals[0].Stability = domain.PortalStable
	want.Simulation.Portals[0].InstabilityCollapseAt = nil

	_, err := store.Commit(ctx, want, nil)
	require.NoError(t, err)
	got, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusOpen, got.Simulation.Portals[0].Status)
	require.Equal(t, want.Simulation.Portals[0].ScheduledCloseAt, got.Simulation.Portals[0].ScheduledCloseAt)
}

func TestSchema_HasNoDerivedRealtimeColumns(t *testing.T) {
	store := openMigratedStore(t)

	for table, forbidden := range map[string][]string{
		"portals":   {"current_energy", "creatures_current", "risk", "risk_score", "remaining_time", "energy_lifetime"},
		"lab_state": {"current_energy"},
	} {
		rows, err := store.db.Query(`PRAGMA table_info(` + table + `)`)
		require.NoError(t, err)
		columns := map[string]bool{}
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			require.NoError(t, rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey))
			columns[name] = true
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		for _, name := range forbidden {
			require.Falsef(t, columns[name], "%s must not contain derived column %s", table, name)
		}
	}
}
