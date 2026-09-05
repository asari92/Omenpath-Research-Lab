import type { AppDTO } from "../../api/types";

export interface TutorialGuidance {
  title: string;
  explanation: readonly string[];
  instruction: string;
  cta: "BEGIN_PRACTICE" | "TRY_SEND" | "START_LIVE" | null;
  waiting: boolean;
}

export function tutorialGuidance(app: AppDTO): TutorialGuidance | null {
  if (app.mode !== "TUTORIAL") return null;
  switch (app.tutorial_step) {
    case 0:
      return {
        title: "Welcome to Omenpath Research Lab",
        explanation: [
          "Your mission is to explore all 85 worlds and reach 85/85.",
          "The Dashboard contains the Lab Summary, Observers and seven Slots. Needs Attention points to urgent work; Details and Event Log explain what happened.",
        ],
        instruction: "Survey the Laboratory interface, then begin practice.",
        cta: "BEGIN_PRACTICE",
        waiting: false,
      };
    case 1:
      return {
        title: "Read a Portal",
        explanation: [
          "Portal Energy differs from Lab Energy and each Portal has individual energy loss. Time Remaining does not guarantee that Energy lasts as long.",
          "Portal Details explains Stability, Risk, Recommendation and History.",
        ],
        instruction: "Open Details for the highlighted Portal.",
        cta: null,
        waiting: false,
      };
    case 2:
      return {
        title: "Wait for the corridor",
        explanation: [
          "Creatures block SEND and one creature leaves every 2 seconds.",
        ],
        instruction:
          "Wait until the authoritative Portal state clears the corridor.",
        cta: null,
        waiting: true,
      };
    case 3:
      return {
        title: "Send an Observer",
        explanation: [
          "SEND costs 0, requires an AVAILABLE Observer and transit takes 5–15 seconds. First movement fixes the Portal direction as OUTBOUND.",
        ],
        instruction: "Use Send Observer on the highlighted Portal.",
        cta: null,
        waiting: false,
      };
    case 4:
      return {
        title: "Stabilize an Omenpath",
        explanation: [
          "STABILIZE costs 20 Lab Energy. Lab Energy ranges from 0–100 and regenerates +1 per second. The Portal must be UNSTABLE and at or below 85%; success adds 15 Portal Energy and produces MEDIUM or LOW Risk.",
        ],
        instruction: "Use Stabilize on the highlighted Portal.",
        cta: null,
        waiting: false,
      };
    case 5:
      return {
        title: "Respect CRITICAL risk",
        explanation: [
          "CRITICAL blocks SEND and RECALL. CLOSE costs 5; CLOSED is controlled, while COLLAPSED drains Lab Energy and begins a Leyline 20-second Override.",
        ],
        instruction:
          "Try SEND on the highlighted CRITICAL Portal to observe the safety rejection.",
        cta: "TRY_SEND",
        waiting: false,
      };
    case 6: {
      if (app.tutorial_phase === "SEND_REPLACEMENT") {
        return {
          title: "Bring the Observer home",
          explanation: [
            "A lost Observer must be replaced before training can continue.",
          ],
          instruction:
            "Send a replacement Observer through the highlighted Portal.",
          cta: null,
          waiting: false,
        };
      }
      if (app.tutorial_phase === "WAIT_RESEARCH") {
        return {
          title: "Bring the Observer home",
          explanation: ["Research takes 20 seconds."],
          instruction: "Wait for authoritative research completion.",
          cta: null,
          waiting: true,
        };
      }
      return {
        title: "Bring the Observer home",
        explanation: [
          "RECALL costs 0 and selects the longest-waiting Observer. A new Portal fixes its direction as INBOUND.",
        ],
        instruction:
          "Recall the waiting Observer through the highlighted Portal.",
        cta: null,
        waiting: false,
      };
    }
    case 7:
      return {
        title: "Survive the return",
        explanation: [
          "A Plane becomes EXPLORED only after a successful return. If a terminal Portal interrupts transit, the Observer becomes LOST.",
        ],
        instruction: "Wait for the authoritative return result.",
        cta: null,
        waiting: true,
      };
    case 8:
      return {
        title: "Inspect the Event Log",
        explanation: ["Global Log and Portal History use the same source."],
        instruction: "Open Event Log from navigation.",
        cta: null,
        waiting: false,
      };
    case 9:
      return {
        title: "Training complete",
        explanation: [
          "Live mode adds Natural Portals. Extraction has a cost 30, a 5-second sync and the first automatic return while preserving Tutorial-to-Live continuity.",
        ],
        instruction: "Start Live mode when ready.",
        cta: "START_LIVE",
        waiting: false,
      };
    default:
      return null;
  }
}
