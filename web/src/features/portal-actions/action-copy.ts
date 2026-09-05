const unavailableCopy: Readonly<Record<string, string>> = {
  PORTAL_NOT_OPEN: 'Portal is no longer open.',
  PORTAL_ALREADY_STABLE: 'Portal is already stable.',
  PORTAL_OVERCHARGE_RISK: 'Portal Energy must be 85% or lower.',
  PORTAL_CRITICAL_RISK: 'CRITICAL risk blocks Observer travel.',
  PORTAL_DIRECTION_CONFLICT: 'Portal direction conflicts with this movement.',
  PORTAL_BUSY: 'An Observer is already moving through this Portal.',
  PORTAL_CREATURES_PRESENT: 'Creatures must clear the corridor first.',
  NO_AVAILABLE_OBSERVER: 'No Observer is available in the Laboratory.',
  NO_WAITING_OBSERVER: 'No Observer is waiting in this Plane.',
  INSUFFICIENT_LAB_ENERGY: 'Laboratory Energy is insufficient.',
  EXTRACTION_SYNCHRONIZING: 'Extraction synchronization is still running.',
};

export function unavailableActionCopy(code: string): string {
  return unavailableCopy[code] ?? `Action unavailable: ${code}`;
}
