export const storageKeys = {
  authSession: "board-table.auth-session",
  guestSession: "board-table.guest-session",
  currentRoomId: "board-table.current-room-id",
  currentSessionId: "board-table.current-session-id"
} as const;

export function clearStoredSessionState() {
  localStorage.removeItem(storageKeys.authSession);
  localStorage.removeItem(storageKeys.guestSession);
  localStorage.removeItem(storageKeys.currentRoomId);
  localStorage.removeItem(storageKeys.currentSessionId);
}
