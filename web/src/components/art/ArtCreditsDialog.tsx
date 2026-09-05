import { fanContentNotice, planeArtEntries } from "../../assets/plane-art";
import styles from "./ArtCreditsDialog.module.css";

export interface ArtCreditsDialogProps {
  open: boolean;
  onClose(): void;
}

export function ArtCreditsDialog({ open, onClose }: ArtCreditsDialogProps) {
  if (!open) return null;
  return (
    <section
      aria-labelledby="art-credits-title"
      aria-modal="true"
      className={styles.dialog}
      role="dialog"
    >
      <header>
        <h2 id="art-credits-title">Plane Artwork Credits</h2>
        <button onClick={onClose} type="button">
          Close
        </button>
      </header>
      <p>{fanContentNotice}</p>
      <ul>
        {planeArtEntries().map((entry) => (
          <li key={entry.plane_id}>
            <strong>{entry.plane_name}</strong> — {entry.credit}{" "}
            <a href={entry.source_url} rel="noreferrer" target="_blank">
              Source
            </a>{" "}
            <a href={entry.policy_url} rel="noreferrer" target="_blank">
              Policy
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
}
