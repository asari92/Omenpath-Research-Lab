import { Link } from "react-router-dom";

import type { StateSnapshot } from "../../api/types";
import styles from "./NeedsAttention.module.css";

export function NeedsAttention({ snapshot }: { snapshot: StateSnapshot }) {
  const id = snapshot.needs_attention_portal_id;
  const portal = snapshot.slots.find((slot) => slot.portal?.id === id)?.portal;
  if (!portal)
    return <p className={styles.nominal}>All open Portals are nominal.</p>;
  return (
    <Link
      aria-label={`Needs Attention: ${portal.name}`}
      className={styles.alert}
      to={`/portals/${portal.id}`}
    >
      <span>Needs Attention</span>
      <strong>{portal.name}</strong>
      <span>{portal.destination_plane_name}</span>
    </Link>
  );
}
