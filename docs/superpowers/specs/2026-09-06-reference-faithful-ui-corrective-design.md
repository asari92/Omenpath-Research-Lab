# Reference-Faithful UI Corrective Design

**Date:** 2026-09-06

**Status:** APPROVED BY USER

**Scope:** frontend presentation and route choreography only

## Authority and intent

`00_FINAL_SPEC_v5.md` remains the product/domain source of truth. This document
replaces the visual interpretation of the earlier Block D corrective design
where the two differ. It does not change gameplay, command eligibility,
simulation timing, persistence, session isolation or Tutorial state-machine
order.

The user-provided 1280×720 dashboard image is the required visual target, not a
loose mood board. The implementation must reproduce its composition, visual
hierarchy and atmosphere with a real responsive React interface. It must not
embed the screenshot as the interface.

## Visual target

The application is one continuous dark magical-laboratory surface:

- blackened parchment/stone texture across the entire viewport;
- fine antique-gold frame around the viewport;
- runic geometry, corner filigree and candlelight as restrained background
  decoration;
- no visually separate sci-fi sidebar or generic application panels;
- warm parchment Tutorial overlay centred along the top edge;
- compact serif typography, antique-gold labels and brighter semantic values;
- no page scrolling on Dashboard or Portal Details at supported viewports.

A dedicated local decorative background plate will be generated from the
approved reference. It contains no text, controls, Portal cards or world art.
All content and controls remain semantic HTML over the background. Existing
local Plane artwork remains the live image inside each Portal.

The primary fidelity target is 1280×720. Larger desktop sizes scale without
loosening the composition. Phone layouts preserve all seven Slots and controls
without page scrolling, using the already approved compact grid.

## Global shell

The left area is integrated into the same full-page surface rather than drawn
as a separate sidebar. Its upper section contains:

1. Lab Energy as a large arcane circular gauge;
2. Observer Life as 20 individually visible markers;
3. Leyline Override state;
4. connection state in in-world language.

Navigation remains at the lower left in the existing order. The entire shell,
including Details, Events, Help and AI Worklog, receives the same background,
frame, typography and ornamental language.

The Dashboard title row keeps `Laboratory Overview`. The single bounded
notification region moves into the unused upper-right area opposite that title.
It must not overlay Portal cards. Confirmation dialogs remain modal. Normal
command errors and unavailable reasons remain available through the relocated
notification region or existing inline command outcome; feedback is not
silently discarded.

## Dashboard and Portal cards

Exactly seven equal-size fixed Slots use a centred `4 + 3` desktop composition.
No Dashboard scrollbar is allowed. Empty and occupied Slots retain identical
outer dimensions.

Each occupied card follows the reference hierarchy:

1. compact Slot number, Plane name and status indicator;
2. large circular world image with a coloured magical Portal ring;
3. Observer transit/status when relevant;
4. compact semantic metrics;
5. one row of four command buttons: STABILIZE, SEND, RECALL and CLOSE.

The standalone `Details` button/link is removed. The occupied card itself is a
navigation affordance: pointer click, `Enter` and `Space` open that Portal's
Details route. Command buttons, unavailable-reason controls and other nested
interactive elements stop card navigation and retain their existing behaviour.
Empty Slots are not navigable.

The Tutorial target outline covers the whole Portal card. Step 1 copy is:

> Select the highlighted Portal on the Dashboard to inspect it.

Help uses the same instruction and no longer refers to pressing a Details
button.

## Semantic colour hierarchy

Labels stay subdued antique gold. Values must not collapse into one text colour:

- Portal Energy uses luminous amber/gold;
- Time Remaining uses pale blue/ivory, distinct from Portal Energy;
- Observer/transit information uses cyan/green with outbound/inbound distinction;
- LOW is green, MEDIUM yellow, HIGH orange and CRITICAL red;
- STABLE is green/teal; UNSTABLE is red;
- explored/unexplored and terminal states remain distinguishable;
- CLOSED/COLLAPSED Portal art stays grayscale.

Colour never becomes the only status carrier: visible text and accessible names
remain.

## Tutorial route choreography

The server-side Tutorial steps and completion conditions remain unchanged.
Frontend route choreography removes context traps:

1. selecting the highlighted Portal opens its Details route and emits the
   existing matching `PORTAL_DETAILS_OPENED` signal;
2. after the authoritative response advances beyond the Details objective, the
   frontend automatically navigates to `/`, so the next Dashboard action is
   visible;
3. opening Event Log at its objective still emits `EVENT_LOG_OPENED`;
4. when the user chooses `Start Live`, the frontend waits for the authoritative
   `mode: LIVE` snapshot, then navigates to `/`;
5. failed/aborted/obsolete requests do not navigate or advance locally.

The Tutorial parchment remains an overlay and never changes page layout. Its
Back/Forward history presentation and seven-second skipped-step replay remain
unchanged.

## Portal Details and secondary routes

Portal Details uses the same full-page visual system and fits into one viewport:

- one large live Portal at centre;
- Portal facts and Observer transit on the left;
- destination, Risk and Recommendation on the right;
- four actions directly below the Portal without a surrounding action card;
- compact bounded History at the bottom; only its inner event list may scroll.

Events, Help and AI Worklog reuse the same shell and ornaments. Long-form content
may keep its bounded inner scroll; the shell itself does not scroll.

## Implementation boundaries

- No gameplay/domain/persistence/API contract changes.
- No new npm dependencies.
- No external hotlinked assets.
- Preserve all 85 local Plane artworks and current Portal canvas lifecycle.
- Preserve reduced-motion behaviour.
- Do not modify deployment topology, session isolation or nginx configuration.

## Acceptance evidence

Implementation requires regression coverage for:

- card click plus keyboard activation opens the correct Portal;
- nested command interaction does not trigger navigation;
- no separate Details control remains in occupied cards;
- Step 1 and Help use the Portal-card instruction;
- authoritative Details completion returns to Dashboard;
- authoritative Start Live completion returns to Dashboard;
- failed Tutorial actions do not navigate;
- notification region is in the Dashboard title row rather than viewport bottom;
- seven equal Slots retain centred `4 + 3` layout with no Dashboard page scroll;
- semantic value/risk/stability classes are present and accessible;
- Dashboard and Portal Details match the approved reference at desktop review.

The corrective pass is complete only after focused RED/GREEN cycles, frontend
unit/type/build checks, a production Docker rebuild and visual verification of
the deployed layout.
