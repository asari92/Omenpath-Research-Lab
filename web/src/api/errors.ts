export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    confirmable: boolean;
  };
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    readonly confirmable: boolean,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function isApiErrorBody(value: unknown): value is ApiErrorBody {
  if (!value || typeof value !== "object" || !("error" in value)) return false;
  const error = value.error;
  return (
    !!error &&
    typeof error === "object" &&
    "code" in error &&
    typeof error.code === "string" &&
    "message" in error &&
    typeof error.message === "string" &&
    "confirmable" in error &&
    typeof error.confirmable === "boolean"
  );
}
