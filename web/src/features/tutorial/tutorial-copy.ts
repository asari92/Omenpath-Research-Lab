import type { AppDTO } from "../../api/types";

export function tutorialSystem(app: AppDTO): {
  action: string;
  completion: string;
} {
  const steps = [
    [
      "The Laboratory waits with seven empty Slots. Beginning practice opens a prepared safe Portal; Natural arrivals remain paused.",
      "You begin practice.",
    ],
    [
      "The prepared Portal stays stable while creatures clear its corridor. Opening the target Details records your visit.",
      "You open the highlighted Portal's Details.",
    ],
    [
      "One creature leaves every 2 seconds. If the prepared Portal ends, the Laboratory recreates this exercise.",
      "The corridor contains no creatures.",
    ],
    [
      "An available Observer begins the ordinary outbound journey. The Laboratory prepares a separate unstable Portal for stabilization.",
      "The Observer is OUTBOUND.",
    ],
    [
      "The Laboratory spends 20 Energy, adds 15 Portal Energy and changes UNSTABLE to STABLE, lowering HIGH/CRITICAL risk to MEDIUM/LOW. Then it prepares a CRITICAL exercise.",
      "The target is STABLE with MEDIUM or LOW risk.",
    ],
    [
      "The safety system rejects SEND and records the event. This Portal is prepared to close naturally before its Energy runs out.",
      "The CRITICAL-risk SEND rejection is recorded.",
    ],
    [
      "After 20 seconds of research, a safe Portal opens to the same Plane for the ordinary return journey.",
      "The Observer is RETURNING.",
    ],
    [
      "A successful return makes the Observer AVAILABLE and the Plane EXPLORED. A loss repeats the return exercise with another available Observer.",
      "The Observer is AVAILABLE and its Plane is EXPLORED.",
    ],
    [
      "Opening Event Log records your visit. Reading events alone does not change the exercise.",
      "You open Event Log.",
    ],
    [
      "The Laboratory closes remaining training Portals for free and starts Natural arrivals. Energy, research, Observers and events carry into Live.",
      "You start Live mode.",
    ],
  ];
  if (app.tutorial_step === 6 && app.tutorial_phase === "SEND_REPLACEMENT")
    return {
      action:
        "The Laboratory prepares a safe outbound Portal. Send another available Observer; it must travel and research normally before recall.",
      completion:
        "The replacement Observer is RETURNING after ordinary travel, research and recall.",
    };
  const [action, completion] = steps[app.tutorial_step] ?? ["", ""];
  return { action, completion };
}

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
          "Omenpaths connect distant Planes across the Multiverse. Our Laboratory charts these shifting paths and learns what lies beyond them.",
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
        instruction:
          "Select the highlighted Portal on the Dashboard to inspect it.",
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
