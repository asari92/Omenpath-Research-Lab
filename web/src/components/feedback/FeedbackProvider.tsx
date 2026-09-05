import {
  createContext,
  type ReactNode,
  useContext,
  useRef,
  useState,
} from "react";

import { ConfirmDialog } from "./ConfirmDialog";
import { ToastRegion, type ToastMessage } from "./ToastRegion";

interface PendingConfirmation {
  action: string;
  portalName: string;
  message: string;
  opener: HTMLElement | null;
  resolve(value: boolean): void;
}

interface FeedbackValue {
  confirmAction(
    action: string,
    portalName: string,
    message: string,
  ): Promise<boolean>;
  notify(message: string): void;
}

const fallback: FeedbackValue = {
  confirmAction: async () => false,
  notify: () => undefined,
};
const FeedbackContext = createContext<FeedbackValue>(fallback);

export function FeedbackProvider({ children }: { children: ReactNode }) {
  const [confirmation, setConfirmation] = useState<PendingConfirmation | null>(
    null,
  );
  const [messages, setMessages] = useState<readonly ToastMessage[]>([]);
  const nextToastID = useRef(1);

  const finish = (answer: boolean) => {
    if (!confirmation) return;
    const { resolve, opener } = confirmation;
    setConfirmation(null);
    resolve(answer);
    queueMicrotask(() => opener?.focus());
  };

  const value: FeedbackValue = {
    confirmAction(action, portalName, message) {
      return new Promise<boolean>((resolve) => {
        setConfirmation((current) => {
          current?.resolve(false);
          return {
            action,
            portalName,
            message,
            opener:
              document.activeElement instanceof HTMLElement
                ? document.activeElement
                : null,
            resolve,
          };
        });
      });
    },
    notify(message) {
      setMessages((current) => [
        ...current,
        { id: nextToastID.current++, message },
      ]);
    },
  };

  return (
    <FeedbackContext.Provider value={value}>
      {children}
      <ToastRegion messages={messages} />
      {confirmation && (
        <ConfirmDialog
          action={confirmation.action}
          message={confirmation.message}
          onCancel={() => finish(false)}
          onConfirm={() => finish(true)}
          open
          opener={confirmation.opener}
          portalName={confirmation.portalName}
        />
      )}
    </FeedbackContext.Provider>
  );
}

export function useFeedback(): FeedbackValue {
  return useContext(FeedbackContext);
}
