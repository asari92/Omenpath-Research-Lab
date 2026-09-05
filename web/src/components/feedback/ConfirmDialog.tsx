import { useEffect, useRef } from "react";

import styles from "./ConfirmDialog.module.css";

export interface ConfirmDialogProps {
  open: boolean;
  action: string;
  portalName: string;
  message: string;
  opener: HTMLElement | null;
  onConfirm(): void;
  onCancel(): void;
}

export function ConfirmDialog({
  open,
  action,
  portalName,
  message,
  opener,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const confirmRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (open) confirmRef.current?.focus();
    else opener?.focus();
  }, [open, opener]);
  if (!open) return null;
  return (
    <div className={styles.backdrop}>
      <div
        aria-label={`Confirm ${action}`}
        aria-modal="true"
        className={styles.dialog}
        role="dialog"
      >
        <h2>
          {action} {portalName}?
        </h2>
        <p>{message}</p>
        <div>
          <button onClick={onCancel} type="button">
            Cancel
          </button>
          <button onClick={onConfirm} ref={confirmRef} type="button">
            Confirm {action}
          </button>
        </div>
      </div>
    </div>
  );
}
