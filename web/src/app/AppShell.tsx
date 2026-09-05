import { useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";

import { ArtCreditsDialog } from "../components/art/ArtCreditsDialog";
import { ConnectionState } from "../components/feedback/ConnectionState";
import { FeedbackProvider } from "../components/feedback/FeedbackProvider";
import { ExtractionDialog } from "../features/extraction/ExtractionDialog";
import { recordNavigationIntent } from "../features/tutorial/navigation-signal";
import { TutorialPanel } from "../features/tutorial/TutorialPanel";
import styles from "./AppShell.module.css";
import { LabSummary } from "../components/dashboard/LabSummary";
import { useSnapshotState } from "../state/SnapshotProvider";
import { commandsEnabled } from "../state/command-health";

export function AppShell() {
  const { pathname } = useLocation();
  const longForm = /^\/(ai-worklog|help)\/?$/.test(pathname);
  const state = useSnapshotState();
  const { snapshot } = state;
  const [creditsOpen, setCreditsOpen] = useState(false);
  const [extractionOpen, setExtractionOpen] = useState(false);
  return (
    <FeedbackProvider>
      <div
        className={styles.shell}
        data-testid="app-shell"
        data-override={snapshot?.lab.leyline_override_active || undefined}
      >
        <header className={styles.header}>
          <NavLink className={styles.brand} to="/">
            Omenpath Research Lab
          </NavLink>
          {snapshot && <LabSummary snapshot={snapshot} />}
          <ConnectionState />
        </header>
        <nav aria-label="Primary navigation" className={styles.navigation}>
          <NavLink to="/">Dashboard</NavLink>
          <NavLink
            onClick={() => recordNavigationIntent({ kind: "events" })}
            to="/events"
          >
            Event Log
          </NavLink>
          <NavLink to="/help">Help</NavLink>
          <button
            className={styles.credits}
            onClick={() => setExtractionOpen(true)}
            type="button"
            disabled={!commandsEnabled(state)}
          >
            Open Extraction
          </button>
          <button
            className={styles.credits}
            onClick={() => setCreditsOpen(true)}
            type="button"
          >
            Artwork Credits
          </button>
          <NavLink to="/ai-worklog">AI Worklog</NavLink>
        </nav>
        <main className={`${styles.main} ${longForm ? styles.longForm : ""}`}>
          <div className={styles.tutorial}>
            <TutorialPanel />
          </div>
          <Outlet />
        </main>
        <div aria-live="polite" data-testid="toast-host" />
        <div data-testid="modal-host" />
        <ArtCreditsDialog
          open={creditsOpen}
          onClose={() => setCreditsOpen(false)}
        />
        <ExtractionDialog
          open={extractionOpen}
          onClose={() => setExtractionOpen(false)}
        />
      </div>
    </FeedbackProvider>
  );
}
