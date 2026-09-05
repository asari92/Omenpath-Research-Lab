export interface ToastMessage {
  id: number;
  message: string;
}

export function ToastRegion({
  messages,
}: {
  messages: readonly ToastMessage[];
}) {
  return (
    <div aria-label="Notifications" aria-live="polite">
      {messages.map((message) => (
        <p key={message.id} role="status">
          {message.message}
        </p>
      ))}
    </div>
  );
}
