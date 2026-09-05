import { NavLink, Outlet } from "react-router-dom";

import styles from "./AppShell.module.css";

export function AppShell() {
  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <NavLink className={styles.brand} to="/">
          Omenpath Research Lab
        </NavLink>
        <nav aria-label="Primary navigation" className={styles.navigation}>
          <NavLink to="/">Dashboard</NavLink>
          <NavLink to="/events">Event Log</NavLink>
          <NavLink to="/ai-worklog">AI Worklog</NavLink>
        </nav>
      </header>
      <main className={styles.main}>
        <Outlet />
      </main>
      <div aria-live="polite" data-testid="toast-host" />
      <div data-testid="modal-host" />
    </div>
  );
}
