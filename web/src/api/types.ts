export type AppMode = "TUTORIAL" | "LIVE";
export type TutorialPhase =
  "" | "SEND_REPLACEMENT" | "WAIT_RESEARCH" | "RECALL_READY";
export type TutorialExpectedAction =
  | "COMPLETE_INTRO"
  | "OPEN_PORTAL_DETAILS"
  | "WAIT_CORRIDOR"
  | "SEND_OBSERVER"
  | "STABILIZE"
  | "ATTEMPT_CRITICAL_SEND"
  | "WAIT_RESEARCH"
  | "RECALL_OBSERVER"
  | "WAIT_RETURN"
  | "OPEN_EVENT_LOG"
  | "START_LIVE";
export type TutorialSignal =
  "TUTORIAL_INTRO_COMPLETED" | "PORTAL_DETAILS_OPENED" | "EVENT_LOG_OPENED";
export type PortalKind = "NATURAL" | "EXTRACTION";
export type PortalStatus = "OPEN" | "CLOSED" | "COLLAPSED";
export type PortalStability = "STABLE" | "UNSTABLE";
export type PortalFlow = "NONE" | "OUTBOUND" | "INBOUND";
export type TerminationReason =
  "" | "NATURAL_CLOSE" | "MANUAL_CLOSE" | "ENERGY_DEPLETED" | "INSTABILITY";
export type RiskLevel = "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";
export type Recommendation =
  | "LEAVE OPEN"
  | "STABILIZE"
  | "RECALL OBSERVER"
  | "WAIT FOR CORRIDOR"
  | "SEND OBSERVER"
  | "CLOSE";
export type EventType =
  | "PORTAL_OPENED"
  | "PORTAL_STABILIZED"
  | "PORTAL_CLOSED"
  | "PORTAL_COLLAPSED"
  | "RISK_LEVEL_CHANGED"
  | "OBSERVER_DISPATCHED"
  | "OBSERVER_ARRIVED"
  | "RESEARCH_STARTED"
  | "RESEARCH_COMPLETED"
  | "OBSERVER_RETURN_STARTED"
  | "OBSERVER_RETURNED"
  | "OBSERVER_LOST"
  | "PLANE_EXPLORED"
  | "EXTRACTION_PORTAL_OPENED"
  | "EXTRACTION_SYNCHRONIZED"
  | "LEYLINE_OVERRIDE_STARTED"
  | "LEYLINE_OVERRIDE_ENDED"
  | "ACTION_REJECTED";

export interface AppDTO {
  mode: AppMode;
  tutorial_step: number;
  tutorial_phase: TutorialPhase;
  tutorial_portal_id: number | null;
  tutorial_plane_id: number | null;
  tutorial_observer_id: number | null;
  expected_action: TutorialExpectedAction | null;
}

export interface LabDTO {
  current_energy: number;
  maximum_energy: number;
  leyline_override_active: boolean;
  leyline_override_until: string | null;
}

export interface ExplorationDTO {
  explored: number;
  total: number;
}

export interface ObserverCountsDTO {
  available: number;
  outbound: number;
  exploring: number;
  waiting_return: number;
  returning: number;
  lost: number;
  in_lab: number;
  in_worlds: number;
  in_transit: number;
}

export interface PortalCountsDTO {
  active: number;
  maximum: number;
  critical: number;
  closed: number;
  collapsed: number;
}

export interface PlaneDTO {
  id: number;
  name: string;
  aliases: string[];
  catalog_tier: string;
  explored: boolean;
  explored_at: string | null;
}

export interface QuickActionsDTO {
  can_stabilize: boolean;
  stabilize_unavailable_reason: string | null;
  can_close: boolean;
  close_unavailable_reason: string | null;
  can_send_observer: boolean;
  send_observer_unavailable_reason: string | null;
  can_recall_observer: boolean;
  recall_observer_unavailable_reason: string | null;
}

export interface SlotPortalDTO {
  id: number;
  name: string;
  destination_plane_id: number;
  destination_plane_name: string;
  destination_explored: boolean;
  energy: number;
  stability: PortalStability;
  time_remaining_seconds: number;
  creatures_inside: number;
  status: PortalStatus;
  quick_actions: QuickActionsDTO;
}

export interface SlotDTO {
  slot_index: number;
  portal: SlotPortalDTO | null;
}

export interface StateSnapshot {
  generated_at: string;
  app: AppDTO;
  lab: LabDTO;
  exploration: ExplorationDTO;
  observers: ObserverCountsDTO;
  portals: PortalCountsDTO;
  needs_attention_portal_id: number | null;
  slots: SlotDTO[];
  planes: PlaneDTO[];
}

export interface PortalViewDTO extends Omit<
  SlotPortalDTO,
  "destination_plane_id" | "destination_plane_name" | "destination_explored"
> {
  slot_index: number;
  kind: PortalKind;
  termination_reason: TerminationReason;
  observer_flow: PortalFlow;
  opened_at: string;
  closed_at: string | null;
}

export interface DestinationDTO {
  plane_id: number;
  name: string;
  aliases: string[];
  catalog_tier: string;
  explored: boolean;
  observers_exploring: number;
  observers_waiting_return: number;
  previous_connection_count: number;
}

export interface EventDTO {
  id: number;
  event_type: EventType;
  portal_id: number | null;
  observer_id: number | null;
  plane_id: number | null;
  message: string;
  payload_json: unknown;
  created_at: string;
}

export interface PortalDetails {
  generated_at: string;
  portal: PortalViewDTO;
  destination: DestinationDTO;
  risk_level: RiskLevel | null;
  recommendation: Recommendation | null;
  history: EventDTO[];
}

export type TutorialSignalRequest =
  | { signal: "TUTORIAL_INTRO_COMPLETED" }
  | { signal: "PORTAL_DETAILS_OPENED"; portal_id: number }
  | { signal: "EVENT_LOG_OPENED" };
