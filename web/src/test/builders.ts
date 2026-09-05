import type { PortalDetails, StateSnapshot } from "../api/types";

export function snapshotAt(
  generatedAt = "2026-09-05T10:00:00Z",
  energy = 100,
): StateSnapshot {
  return {
    generated_at: generatedAt,
    app: {
      mode: "TUTORIAL",
      tutorial_step: 0,
      tutorial_phase: "",
      tutorial_portal_id: null,
      tutorial_plane_id: null,
      tutorial_observer_id: null,
      expected_action: "COMPLETE_INTRO",
    },
    lab: {
      current_energy: energy,
      maximum_energy: 100,
      leyline_override_active: false,
      leyline_override_until: null,
    },
    exploration: { explored: 0, total: 85 },
    observers: {
      available: 5,
      outbound: 0,
      exploring: 0,
      waiting_return: 0,
      returning: 0,
      lost: 0,
      in_lab: 5,
      in_worlds: 0,
      in_transit: 0,
    },
    portals: { active: 0, maximum: 7, critical: 0, closed: 0, collapsed: 0 },
    needs_attention_portal_id: null,
    slots: Array.from({ length: 7 }, (_, index) => ({
      slot_index: index + 1,
      portal: null,
    })),
    planes: [],
  };
}

export function portalDetails(id = 42): PortalDetails {
  return {
    generated_at: "2026-09-05T10:00:00Z",
    portal: {
      id,
      name: `Portal ${id}`,
      slot_index: 1,
      kind: "NATURAL",
      status: "OPEN",
      termination_reason: "",
      energy: 75,
      stability: "STABLE",
      time_remaining_seconds: 90,
      creatures_inside: 0,
      observer_flow: "NONE",
      opened_at: "2026-09-05T09:59:00Z",
      closed_at: null,
      quick_actions: {
        can_stabilize: false,
        stabilize_unavailable_reason: 'PORTAL_ALREADY_STABLE',
        can_close: true,
        close_unavailable_reason: null,
        can_send_observer: true,
        send_observer_unavailable_reason: null,
        can_recall_observer: false,
        recall_observer_unavailable_reason: 'NO_WAITING_OBSERVER',
      },
    },
    destination: {
      plane_id: 1,
      name: "Agyrem",
      aliases: [],
      catalog_tier: "A",
      explored: false,
      observers_exploring: 0,
      observers_waiting_return: 0,
      previous_connection_count: 0,
    },
    risk_level: "LOW",
    recommendation: "SEND OBSERVER",
    history: [],
  };
}
