ALTER TABLE app_state ADD COLUMN tutorial_phase TEXT NOT NULL DEFAULT ''
    CHECK (tutorial_phase IN ('', 'SEND_REPLACEMENT', 'WAIT_RESEARCH', 'RECALL_READY'));
ALTER TABLE app_state ADD COLUMN tutorial_portal_id INTEGER CHECK (tutorial_portal_id IS NULL OR tutorial_portal_id > 0);
ALTER TABLE app_state ADD COLUMN tutorial_plane_id INTEGER CHECK (tutorial_plane_id IS NULL OR tutorial_plane_id > 0);
ALTER TABLE app_state ADD COLUMN tutorial_observer_id INTEGER CHECK (tutorial_observer_id IS NULL OR tutorial_observer_id > 0);
