import type { RealtimeMessage } from "../domain/types";

export function parseRealtimeMessage(value: unknown): RealtimeMessage | null {
  if (typeof value !== "string") return null;
  try {
    const parsed = JSON.parse(value) as Partial<RealtimeMessage>;
    if (typeof parsed.type !== "string" || typeof parsed.payload !== "object" || parsed.payload === null) {
      return null;
    }
    return parsed as RealtimeMessage;
  } catch {
    return null;
  }
}
