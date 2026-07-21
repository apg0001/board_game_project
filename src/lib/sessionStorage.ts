import type { AuthSession, GuestSession } from "../domain/types";
import { storageKeys } from "../storageKeys";
import { safeNickname } from "./playerIdentity";

export const guestStorageKey = storageKeys.guestSession;
export const authStorageKey = storageKeys.authSession;
export const roomStorageKey = storageKeys.currentRoomId;
export const sessionStorageKey = storageKeys.currentSessionId;

export function readGuestSession(): GuestSession | null {
  try {
    const raw = localStorage.getItem(guestStorageKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<GuestSession>;
    if (!isStoredPlayerSession(parsed)) {
      localStorage.removeItem(guestStorageKey);
      return null;
    }
    return parsed;
  } catch {
    localStorage.removeItem(guestStorageKey);
    return null;
  }
}

export function saveGuestSession(session: GuestSession) {
  localStorage.setItem(guestStorageKey, JSON.stringify(session));
}

export function readAuthSession(): AuthSession | null {
  try {
    const raw = localStorage.getItem(authStorageKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<AuthSession>;
    if (!isStoredPlayerSession(parsed) || typeof parsed.user?.username !== "string") {
      localStorage.removeItem(authStorageKey);
      return null;
    }
    return {
      sessionToken: parsed.sessionToken,
      user: {
        id: parsed.user.id,
        username: parsed.user.username,
        nickname: safeNickname(parsed.user.nickname, parsed.user.username),
        role: parsed.user.role === "ADMIN" ? "ADMIN" : "USER"
      }
    };
  } catch {
    localStorage.removeItem(authStorageKey);
    return null;
  }
}

export function saveAuthSession(session: AuthSession) {
  localStorage.setItem(authStorageKey, JSON.stringify(normalizeAuthSession(session)));
}

export function readPlayerSession(): GuestSession | null {
  const auth = readAuthSession();
  if (auth) {
    return {
      sessionToken: auth.sessionToken,
      user: { id: auth.user.id, nickname: auth.user.nickname }
    };
  }
  return readGuestSession();
}

export function normalizeAuthSession(session: AuthSession): AuthSession {
  return {
    sessionToken: session.sessionToken,
    user: {
      id: session.user.id,
      username: session.user.username,
      nickname: safeNickname(session.user.nickname, session.user.username),
      role: session.user.role === "ADMIN" ? "ADMIN" : "USER"
    }
  };
}

function isStoredPlayerSession(value: Partial<GuestSession> | Partial<AuthSession>): value is GuestSession {
  return (
    typeof value.sessionToken === "string" &&
    value.sessionToken.length > 0 &&
    typeof value.user?.id === "string" &&
    value.user.id.length > 0 &&
    typeof value.user?.nickname === "string" &&
    value.user.nickname.length > 0
  );
}
