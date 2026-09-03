package domain

import "time"

// Plane is a permanent world (Final Spec §3).
// Exploration status belongs to the Plane, not to a Portal.
type Plane struct {
	ID          int64
	Name        string
	Aliases     []string
	CatalogTier string
	Explored    bool
	ExploredAt  *time.Time
}
