CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS planes (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    aliases_json TEXT NOT NULL,
    catalog_tier TEXT NOT NULL,
    explored INTEGER NOT NULL CHECK (explored IN (0, 1)),
    explored_at INTEGER
);

CREATE TABLE IF NOT EXISTS portals (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    slot_index INTEGER NOT NULL CHECK (slot_index BETWEEN 1 AND 7),
    kind TEXT NOT NULL CHECK (kind IN ('NATURAL', 'EXTRACTION')),
    destination_plane_id INTEGER NOT NULL REFERENCES planes(id),
    energy_base REAL NOT NULL CHECK (energy_base >= 0),
    energy_base_at INTEGER NOT NULL,
    energy_decay_rate REAL NOT NULL CHECK (energy_decay_rate > 0),
    stability TEXT NOT NULL CHECK (stability IN ('STABLE', 'UNSTABLE')),
    opened_at INTEGER NOT NULL,
    scheduled_close_at INTEGER NOT NULL,
    instability_collapse_at INTEGER,
    creatures_initial INTEGER NOT NULL CHECK (creatures_initial >= 0),
    observer_flow TEXT NOT NULL CHECK (observer_flow IN ('NONE', 'OUTBOUND', 'INBOUND')),
    extraction_synchronized_at INTEGER,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'CLOSED', 'COLLAPSED')),
    termination_reason TEXT NOT NULL CHECK (termination_reason IN ('', 'NATURAL_CLOSE', 'MANUAL_CLOSE', 'ENERGY_DEPLETED', 'INSTABILITY')),
    closed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_portals_one_open_per_slot
    ON portals(slot_index) WHERE status = 'OPEN';

CREATE TABLE IF NOT EXISTS observers (
    id INTEGER PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('AVAILABLE', 'OUTBOUND', 'EXPLORING', 'WAITING_RETURN', 'RETURNING', 'LOST')),
    current_plane_id INTEGER REFERENCES planes(id),
    active_portal_id INTEGER REFERENCES portals(id),
    phase_started_at INTEGER,
    phase_ends_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'PORTAL_OPENED', 'PORTAL_STABILIZED', 'PORTAL_CLOSED', 'PORTAL_COLLAPSED',
        'RISK_LEVEL_CHANGED', 'OBSERVER_DISPATCHED', 'OBSERVER_ARRIVED',
        'RESEARCH_STARTED', 'RESEARCH_COMPLETED', 'OBSERVER_RETURN_STARTED',
        'OBSERVER_RETURNED', 'OBSERVER_LOST', 'PLANE_EXPLORED',
        'EXTRACTION_PORTAL_OPENED', 'EXTRACTION_SYNCHRONIZED',
        'LEYLINE_OVERRIDE_STARTED', 'LEYLINE_OVERRIDE_ENDED', 'ACTION_REJECTED'
    )),
    portal_id INTEGER,
    observer_id INTEGER,
    plane_id INTEGER,
    message TEXT NOT NULL CHECK (length(trim(message)) > 0),
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json) AND json_type(payload_json) = 'object'),
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_created_at_id
    ON events(created_at, id);
CREATE INDEX IF NOT EXISTS idx_events_portal_created_at_id
    ON events(portal_id, created_at, id);

CREATE TABLE IF NOT EXISTS lab_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    energy_base INTEGER NOT NULL CHECK (energy_base BETWEEN 0 AND 100),
    energy_base_at INTEGER NOT NULL,
    override_until INTEGER
);

CREATE TABLE IF NOT EXISTS app_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    mode TEXT NOT NULL CHECK (mode IN ('TUTORIAL', 'LIVE')),
    tutorial_step INTEGER NOT NULL CHECK (tutorial_step >= 0),
    next_portal_id INTEGER NOT NULL CHECK (next_portal_id > 0),
    spawn_scheduled_at INTEGER,
    spawn_due_at INTEGER,
    spawn_paused INTEGER NOT NULL CHECK (spawn_paused IN (0, 1)),
    last_tick_at INTEGER
);
