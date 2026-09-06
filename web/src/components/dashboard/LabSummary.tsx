import type { CSSProperties } from "react";

import type { StateSnapshot } from "../../api/types";
import styles from "./LabSummary.module.css";

export function LabSummary({ snapshot }: { snapshot: StateSnapshot }) {
  const energyPercent = Math.max(
    0,
    Math.min(
      100,
      (snapshot.lab.current_energy / snapshot.lab.maximum_energy) * 100,
    ),
  );
  const gaugeStyle = {
    "--energy-level": `${energyPercent}%`,
  } as CSSProperties;

  return (
    <dl aria-label="Laboratory summary" className={styles.summary}>
      <div
        className={styles.energy}
        data-testid="lab-energy-gauge"
        style={gaugeStyle}
      >
        <dt>Lab Energy</dt>
        <dd>
          <strong>{snapshot.lab.current_energy}</strong>
          <span> / {snapshot.lab.maximum_energy}</span>
        </dd>
      </div>
      <div className={styles.roster}>
        <dt>Observer Life</dt>
        <dd>
          {snapshot.observers.in_lab} <span>/ 20</span>
        </dd>
        <div
          className={styles.pips}
          role="img"
          aria-label={`${snapshot.observers.in_lab} of 20 Observers in Lab`}
        >
          {Array.from({ length: 20 }, (_, index) => (
            <i
              key={index}
              data-testid="observer-pip"
              data-available={index < snapshot.observers.in_lab}
            />
          ))}
        </div>
      </div>
      <div className={styles.override}>
        <dt>Leyline Override</dt>
        <dd data-active={snapshot.lab.leyline_override_active}>
          {snapshot.lab.leyline_override_active ? "ACTIVE" : "INACTIVE"}
        </dd>
      </div>
      <div className={styles.compact}>
        <div>
          <dt>Exploration</dt>
          <dd>
            {snapshot.exploration.explored} / {snapshot.exploration.total}
          </dd>
        </div>
        <div>
          <dt>Portals</dt>
          <dd>
            Active {snapshot.portals.active} / {snapshot.portals.maximum}
          </dd>
        </div>
        <div>
          <dt>Critical </dt>
          <dd>{snapshot.portals.critical}</dd>
        </div>
        <div>
          <dt>Observers</dt>
          <dd>
            Available {snapshot.observers.available} · In worlds{" "}
            {snapshot.observers.in_worlds} · In transit{" "}
            {snapshot.observers.in_transit} · Lost {snapshot.observers.lost}
          </dd>
        </div>
        <div>
          <dt>Terminal Portals</dt>
          <dd>
            Closed {snapshot.portals.closed} · Collapsed{" "}
            {snapshot.portals.collapsed}
          </dd>
        </div>
      </div>
    </dl>
  );
}
