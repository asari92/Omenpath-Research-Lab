import { useState } from "react";
import { NavLink, Outlet } from "react-router-dom";

import { ArtCreditsDialog } from "../components/art/ArtCreditsDialog";
import { ConnectionState } from "../components/feedback/ConnectionState";
import { FeedbackProvider } from "../components/feedback/FeedbackProvider";
import { ExtractionDialog } from "../features/extraction/ExtractionDialog";
import styles from "./AppShell.module.css";

export function AppShell() {
  const [creditsOpen, setCreditsOpen] = useState(false);
  const [extractionOpen, setExtractionOpen] = useState(false);
  return (
    <FeedbackProvider>
      <div className={styles.shell}>
        <header className={styles.header}>
          <NavLink className={styles.brand} to="/">
            Omenpath Research Lab
          </NavLink>
          <nav aria-label="Primary navigation" className={styles.navigation}>
            <NavLink to="/">Dashboard</NavLink>
            <NavLink to="/events">Event Log</NavLink>
            <NavLink to="/ai-worklog">AI Worklog</NavLink>
            <button
              className={styles.credits}
              onClick={() => setExtractionOpen(true)}
              type="button"
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
          </nav>
        </header>
        <main className={styles.main}>
          <ConnectionState />
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
