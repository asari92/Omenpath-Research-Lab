import { Link } from "react-router-dom";

const chapters = [
  [
    "The Multiverse",
    "Omenpaths connect the many worlds of Magic: The Gathering. Omenpath Research Lab studies these passages and sends researchers beyond them. Explore all 85 Planes to complete the laboratory’s mission. A world counts as explored only when an Observer brings completed research safely home.",
  ],
  [
    "Laboratory interface",
    "The Dashboard keeps seven fixed Portal Slots visible. Laboratory statistics appear above the navigation. Needs Attention points to a priority Portal without rearranging Slots. Each occupied Slot shows its destination, energy, stability, time, creatures, transit and four quick actions. Details contains Risk, Recommendations and that Portal’s History. Event Log records the whole laboratory; its filters combine. The planar-link indicator reports connection health. During disconnection, the last snapshot remains visible and commands are blocked until the connection recovers.",
  ],
  [
    "Two kinds of energy",
    "Lab Energy is a shared reserve from 0 to 100, regenerating by 1 each second. Portal Energy belongs to each individual Portal and appears as a percentage with one decimal place. Each Portal drains at its own hidden rate. Time Remaining describes its scheduled closure: energy can run out earlier, so the timer is not a guarantee of safety.",
  ],
  [
    "Portal lifecycle",
    "Every opening is a unique Portal instance, even when it leads to a familiar Plane. OPEN Portals occupy a Slot. Stability is a separate property: STABLE or UNSTABLE. STABILIZED is an event, not another lifecycle status. Stabilization makes an UNSTABLE Portal STABLE and adds 15 percentage points of Portal Energy. Scheduled expiry or manual closure produces CLOSED; depleted energy or instability produces COLLAPSED. Both free the Slot. Details retains the final outcome, termination reason and History, with a grayscale Portal. Multiple Portals can lead to the same Plane.",
  ],
  [
    "Risk and instability",
    "LOW (green) means comfortable operating conditions; MEDIUM (yellow) needs attention; HIGH (orange) warns the corridor may soon become unsafe; CRITICAL (red) forbids SEND and RECALL. Risk is guidance derived from current conditions, not a countdown to a hidden collapse. UNSTABLE Portals can collapse unexpectedly. Sending or recalling through one requires confirmation. Terminal Portals have no current Risk or Recommendation.",
  ],
  [
    "Observers and transit",
    "The laboratory has 20 permanent Observers. AVAILABLE means in the Lab; OUTBOUND means travelling to a Plane; EXPLORING means researching; WAITING_RETURN means ready to return; RETURNING means travelling home. LOST is permanent. Each transit takes a separately chosen 5–15 seconds; its identity, direction and remaining time appear on the Slot and in Details. Only one Observer can travel through a Portal at once. Closure strictly before the transit deadline makes that Observer LOST; closure exactly at the deadline allows transit to finish. Observers already in the Plane remain there when a Portal closes. The first SEND locks that Portal to OUTBOUND; the first RECALL locks it to INBOUND. This direction never resets. Multiple Observers may occupy a Plane, and sending to an explored Plane is allowed.",
  ],
  [
    "Research and creatures",
    "Arrival in a Plane begins 20 seconds of research. Completing research changes the Observer to WAITING_RETURN but does not yet mark the Plane EXPLORED. A successful return delivers the research; repeated returns never count the same Plane twice. Creatures pass through a Portal one at a time, every 2 seconds. While any remain, SEND and RECALL are blocked. Closing a corridor containing creatures requires confirmation.",
  ],
  [
    "Portal commands",
    "SEND OBSERVER costs 0 Lab Energy: requires an OPEN, non-CRITICAL Portal, an available Observer, no creatures, no active transit and a direction other than INBOUND. RECALL OBSERVER costs 0: requires an OPEN, non-CRITICAL Portal with no creatures or transit, direction other than OUTBOUND, and a WAITING_RETURN Observer in its destination; it selects the longest-waiting eligible Observer. CLOSE costs 5 Lab Energy and requires an OPEN Portal; creatures or active transit require confirmation, and closing during transit loses that Observer. STABILIZE costs 20 Lab Energy: requires OPEN and UNSTABLE with current Portal Energy at most 85%; it adds 15 percentage points, removes instability, and leaves the drain rate unchanged. Occupied unavailable commands explain their reason without sending a request. Empty Slot controls are disabled.",
  ],
  [
    "Extraction",
    "Open Extraction costs 30 Lab Energy, requires a free Slot and at least one WAITING_RETURN Observer. Select a Plane with a waiting Observer. The new Portal is STABLE, INBOUND, creature-free, and uses one ordinary Slot. After 5 seconds of synchronization, the longest-waiting Observer still eligible in that Plane automatically begins RETURNING. Selection happens at synchronization completion: if the earlier candidate has left, another eligible waiting Observer is selected; if none remain, no automatic return occurs and the Portal otherwise continues normally. Only the first return is automatic; use RECALL for further Observers. Closing during synchronization leaves waiting Observers unchanged in the Plane. Closing strictly before an actual return finishes makes the travelling Observer LOST. The synchronization and transit are separate phases.",
  ],
  [
    "Leyline Override",
    "Every COLLAPSED Portal empties Lab Energy and activates Leyline Override for 20 seconds. The laboratory’s atmosphere changes while Override is active. CLOSE and STABILIZE cost 0 during Override, but all their other restrictions remain. Extraction still costs 30. Energy continues regenerating; another collapse empties it again and restarts the 20-second Override.",
  ],
  [
    "Recommendations",
    "Details prioritizes safe Observer travel, preventing collapse and Lab Energy loss, researching unexplored worlds, then avoiding needless actions in explored worlds. It may suggest LEAVE OPEN, WAIT FOR CORRIDOR, STABILIZE, CLOSE, SEND OBSERVER or RECALL OBSERVER. A waiting Observer’s return takes priority over a new expedition. Advice considers known transit deadlines, research, creatures and a conservative travel allowance, without knowing the hidden instability time. Stabilization is suggested only when available and helpful for the required safety margin. During transit, advice will not recommend closing and guaranteeing a loss. Recommendations never perform commands or replace their authoritative availability rules; LEAVE OPEN is not a promise of safety.",
  ],
  [
    "Tutorial recap",
    "Begin with the laboratory introduction, inspect the training Portal’s Details, wait for the corridor, stabilize when directed, and learn why a CRITICAL Portal rejects travel. Follow the current objective to send an Observer, wait for research, recall and verify a successful return. Inspect the Event Log, then start Live exploration. Training advances when its expected action or event occurs. Tutorial reset starts the training laboratory over; it is not a way to preserve Live progress.",
  ],
  [
    "Glossary",
    "Plane: a persistent world and its research progress. Omenpath / Portal: one temporary connection to that world. Slot: one of seven fixed spaces for OPEN Portals. Corridor: the passage used by creatures and an Observer. Flow: the permanent travel direction of a Portal. Transit: the timed journey between Lab and Plane. Research: work performed after arrival. Extraction: a controlled return Portal. Collapse: an emergency terminal outcome that drains Lab Energy. Snapshot: the latest authoritative laboratory state. One browser profile shares one laboratory across tabs; a different profile or device receives a separate laboratory. An active anonymous session lasts 30 days since activity; an expired or cleared cookie starts a new game.",
  ],
] as const;

export function HelpPage() {
  return (
    <article
      style={{ maxWidth: "65rem", padding: ".5rem 1rem", lineHeight: 1.65 }}
    >
      <h1>Help</h1>
      <p>The Omenpath Research Lab field guide</p>
      <nav aria-label="Field guide chapters">
        {chapters.map(([title], index) => (
          <a
            key={title}
            href={`#chapter-${index}`}
            style={{ display: "inline-block", marginRight: "1rem" }}
          >
            {title}
          </a>
        ))}
      </nav>
      {chapters.map(([title, copy], index) => (
        <section id={`chapter-${index}`} key={title}>
          <h2>{title}</h2>
          <p>{copy}</p>
        </section>
      ))}
      <Link to="/">Return to the laboratory</Link>
    </article>
  );
}
