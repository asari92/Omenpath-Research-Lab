export interface ToastMessage {
  id: number;
  message: string;
}

export function ToastRegion({
  messages,
  onDismiss,
}: {
  messages: readonly ToastMessage[];
  onDismiss(): void;
}) {
  return (
    <div
      aria-label="Notifications"
      aria-live="polite"
      aria-atomic="true"
      data-placement="page-header"
    >
      {messages.map((message) => (
        <p key={message.id} role="status">
          {message.message}
        </p>
      ))}
      {messages.length > 0 && (
        <button
          aria-label="Dismiss notification"
          onClick={onDismiss}
          type="button"
        >
          ×
        </button>
      )}
    </div>
  );
}
