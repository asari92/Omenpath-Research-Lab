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
  const totalObservers = 20;
  const availableObservers = Math.max(
    0,
    Math.min(totalObservers, snapshot.observers.available),
  );
  const lostObservers = Math.max(
    0,
    Math.min(totalObservers - availableObservers, snapshot.observers.lost),
  );
  const deployedObservers =
    totalObservers - availableObservers - lostObservers;
  const survivingObservers = totalObservers - lostObservers;

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
          {survivingObservers} <span>/ {totalObservers}</span>
        </dd>
        <div
          className={styles.pips}
          role="img"
          aria-label={`Observer status: ${availableObservers} available, ${deployedObservers} deployed, ${lostObservers} lost`}
        >
          {Array.from({ length: totalObservers }, (_, index) => (
            <i
              key={index}
              data-testid="observer-pip"
              data-state={
                index < availableObservers
                  ? "available"
                  : index >= totalObservers - lostObservers
                    ? "lost"
                    : "deployed"
              }
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
