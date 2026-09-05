import type { StateSnapshot } from "../../api/types";
import styles from "./LabSummary.module.css";

export function LabSummary({ snapshot }: { snapshot: StateSnapshot }) {
  const items = [
    [
      "Laboratory Energy",
      `${snapshot.lab.current_energy} / ${snapshot.lab.maximum_energy}`,
    ],
    [
      "Exploration",
      `${snapshot.exploration.explored} / ${snapshot.exploration.total}`,
    ],
    ["Observers", `Available ${snapshot.observers.available}`],
    ["Observers", `In worlds ${snapshot.observers.in_worlds}`],
    ["Observers", `In transit ${snapshot.observers.in_transit}`],
    ["Observers", `Lost ${snapshot.observers.lost}`],
    [
      "Portals",
      `Active ${snapshot.portals.active} / ${snapshot.portals.maximum}`,
    ],
    ["Portals", `Critical ${snapshot.portals.critical}`],
    [
      "Portals",
      `Closed ${snapshot.portals.closed} · Collapsed ${snapshot.portals.collapsed}`,
    ],
  ] as const;

  return (
    <dl aria-label="Laboratory summary" className={styles.summary}>
      <div className={styles.roster}>
        <dt>Observers in Lab</dt>
        <dd>{snapshot.observers.in_lab} / 20</dd>
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
      {items.map(([label, value], index) => (
        <div className={styles.item} key={`${label}-${index}`}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}
