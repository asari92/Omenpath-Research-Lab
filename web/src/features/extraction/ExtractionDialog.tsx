import { useEffect, useMemo, useRef, useState } from "react";

import { planeArt } from "../../assets/plane-art";
import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";
import { filterPlanes, type PlaneFilter } from "./plane-filter";
import styles from "./ExtractionDialog.module.css";
import { commandsEnabled } from "../../state/command-health";

const filters: readonly { value: PlaneFilter; label: string }[] = [
  { value: "ALL", label: "All" },
  { value: "UNEXPLORED", label: "Unexplored" },
  { value: "EXPLORED", label: "Explored" },
  { value: "OBSERVER_PRESENT", label: "Observer Present" },
];

export function ExtractionDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose(): void;
}) {
  const { api, store } = useSnapshotContext();
  const state = useSnapshotState();
  const { snapshot } = state;
  const root = useRef<HTMLDivElement>(null);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<PlaneFilter>("ALL");
  const [selected, setSelected] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const visible = useMemo(
    () => filterPlanes(snapshot?.planes ?? [], query, filter),
    [filter, query, snapshot?.planes],
  );

  useEffect(() => {
    if (open)
      root.current?.querySelector<HTMLElement>("button, input")?.focus();
  }, [open]);
  if (!open || !snapshot) return null;

  return (
    <div className={styles.backdrop}>
      <div
        aria-label="Open Extraction Portal"
        aria-modal="true"
        className={styles.dialog}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            onClose();
            return;
          }
          if (event.key !== "Tab") return;
          const focusable = [
            ...(root.current?.querySelectorAll<HTMLElement>(
              "button:not(:disabled), input:not(:disabled)",
            ) ?? []),
          ];
          if (focusable.length === 0) return;
          const first = focusable[0];
          const last = focusable[focusable.length - 1];
          if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
          } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
          }
        }}
        ref={root}
        role="dialog"
      >
        <header>
          <div>
            <h2>Open Extraction Portal</h2>
            <p>
              Lab Energy {snapshot.lab.current_energy} /{" "}
              {snapshot.lab.maximum_energy}. Backend validation remains
              authoritative.
            </p>
          </div>
          <button
            aria-label="Close Extraction chooser"
            onClick={onClose}
            type="button"
          >
            ×
          </button>
        </header>
        <input
          aria-label="Search Planes"
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search name or alias"
          type="search"
          value={query}
        />
        <fieldset className={styles.filters}>
          <legend>Plane filter</legend>
          {filters.map((item) => (
            <label key={item.value}>
              <input
                checked={filter === item.value}
                name="plane-filter"
                onChange={() => setFilter(item.value)}
                type="radio"
              />
              {item.label}
            </label>
          ))}
        </fieldset>
        <div className={styles.planes}>
          {visible.map((plane) => (
            <button
              aria-label={plane.name}
              aria-pressed={selected === plane.id}
              className={styles.card}
              data-testid="plane-card"
              key={plane.id}
              onClick={() => {
                setSelected(plane.id);
                setError(null);
              }}
              type="button"
            >
              <img
                alt={`${plane.name} Plane`}
                src={planeArt(plane.id).local_path}
              />
              <strong>{plane.name}</strong>
              <span>{plane.explored ? "EXPLORED" : "UNEXPLORED"}</span>
              <span>In Plane {plane.observers_in_plane}</span>
              <span>Waiting {plane.observers_waiting_return}</span>
            </button>
          ))}
        </div>
        <footer>
          <p data-testid="selection-summary">
            {selected === null
              ? "Select a Plane"
              : `Selected Plane ${selected} — cost 30 Lab Energy`}
          </p>
          {error && <p role="alert">{error}</p>}
          <button
            disabled={selected === null || busy || !commandsEnabled(state)}
            onClick={() => {
              if (
                selected === null ||
                busy ||
                !commandsEnabled(store.getState())
              )
                return;
              setBusy(true);
              setError(null);
              void api
                .openExtraction(selected)
                .then((next) => {
                  store.acceptSnapshot(next);
                  onClose();
                })
                .catch((reason: unknown) =>
                  setError(
                    reason instanceof Error
                      ? reason.message
                      : "Unable to open Extraction Portal",
                  ),
                )
                .finally(() => setBusy(false));
            }}
            type="button"
          >
            {busy ? "Opening…" : "Open Extraction"}
          </button>
        </footer>
      </div>
    </div>
  );
}
