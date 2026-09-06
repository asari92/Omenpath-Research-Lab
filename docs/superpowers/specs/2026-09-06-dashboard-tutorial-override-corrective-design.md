# Dashboard, Tutorial and Override Corrective Design

## Scope

This corrective pass changes only the existing frontend presentation and Tutorial navigation. It does not change Observer lifecycle, Portal mechanics, Tutorial progression on the server, or Leyline Override gameplay semantics.

## Observer Life summary

- Keep exactly 20 Observer pips.
- The numeric value is surviving Observers (`20 - lost`) out of 20.
- Green pips represent `AVAILABLE` Observers.
- Red pips represent terminal `LOST` Observers.
- Dark neutral pips represent surviving Observers currently in worlds or in transit.
- Remove the separate compact `Observers: Available / In worlds / In transit / Lost` row.
- Use the recovered space to increase the compact summary typography.
- Preserve accessible text that exposes the available, deployed/transit and lost counts even though the duplicate visible row is removed.

## Tutorial navigation from Portal Details

- Opening the highlighted Tutorial Portal sends `PORTAL_DETAILS_OPENED` and accepts the authoritative next snapshot without automatic navigation.
- The user remains on Portal Details after the step is accepted.
- Reuse the existing `Forward` button; do not add another `Next` control.
- When older Tutorial cards are being reviewed, `Forward` continues to advance through the local card history.
- When the latest available card is shown on Portal Details after the details-opening objective has completed, `Forward` navigates to Dashboard.
- Directly loading Portal Details without recorded navigation intent still sends no Tutorial signal.

## Leyline Override presentation

- Remove the Dashboard's floating `Leyline Override active until ...` banner because the left summary remains authoritative and visible.
- Keep the existing full-shell palette change.
- Add a pointer-transparent, screen-wide animated energy layer below interactive content. Moving violet-white diagonal streaks and glow pulses make Override unmistakable without animating text, cards or controls.
- Respect `prefers-reduced-motion`: retain a strong static Override glow but stop movement.
- No video, Canvas runtime, dependency or remote asset is introduced.

## Verification

- Component tests cover Observer pip classification and removal of the visible duplicate row.
- Portal Details and Tutorial tests cover staying on Details after the signal and reusing `Forward` for Dashboard navigation.
- Dashboard tests cover removal of the floating Override banner while the summary state remains visible.
- App shell tests cover the Override energy layer and reduced-motion-safe CSS structure.
- Run frontend format, lint, typecheck, unit tests and production build, followed by the repository Go quality commands required by `AGENTS.md`.
