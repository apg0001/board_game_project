import { games } from "../domain/gameCatalog";
import type {
  AuthSession,
  ChatMessage,
  GameActionType,
  GameSession,
  GuestSession,
  LeaderboardRow,
  Presence,
  Room,
  RoomSettings,
  TutorialGuide
} from "../domain/types";
import { listOf } from "./collections";
import { normalizeChatMessage, normalizeRoom } from "./normalizers";
import {
  authStorageKey,
  normalizeAuthSession,
  readAuthSession,
  readPlayerSession,
  saveAuthSession,
  saveGuestSession
} from "./sessionStorage";

export async function createGuest(apiURL: string): Promise<GuestSession> {
  const response = await fetch(`${apiURL}/api/guests`, { method: "POST" });
  if (!response.ok) throw new Error("failed to create guest");

  const session = (await response.json()) as GuestSession;
  saveGuestSession(session);
  return session;
}

export async function register(username: string, password: string, nickname: string): Promise<AuthSession> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password, nickname })
  });
  if (!response.ok) throw new Error("register failed");
  const session = normalizeAuthSession((await response.json()) as AuthSession);
  saveAuthSession(session);
  return session;
}

export async function login(username: string, password: string): Promise<AuthSession> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password })
  });
  if (!response.ok) throw new Error("login failed");
  const session = normalizeAuthSession((await response.json()) as AuthSession);
  saveAuthSession(session);
  return session;
}

export function submitAuth(mode: "login" | "register", username: string, password: string, nickname: string) {
  if (mode === "register") return register(username, password, nickname);
  return login(username, password);
}

export async function createRoom(
  gameId: string,
  visibility: "PRIVATE" | "PUBLIC",
  maxPlayers: number,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>("/api/rooms", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ gameId, maxPlayers, visibility })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(`${data.room.code} 코드를 공유하세요.`);
    markPresence(data.room.id, undefined, "ONLINE").catch(() => undefined);
  } catch {
    onMessage("방 생성에 실패했습니다.");
  }
}

export async function fetchPublicRooms(gameId = ""): Promise<Room[]> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const params = new URLSearchParams({ visibility: "PUBLIC" });
  if (gameId) params.set("gameId", gameId);
  const response = await fetch(`${apiURL}/api/rooms?${params.toString()}`);
  if (!response.ok) throw new Error("failed to fetch public rooms");
  const data = (await response.json()) as { rooms: Room[] };
  return (data.rooms ?? []).map(normalizeRoom);
}

export async function quickMatch(
  gameId: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const game = games.find((item) => item.id === gameId);
    const data = await authorizedJSON<{ room: Room; matched: boolean }>("/api/match/quick", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ gameId, maxPlayers: game?.maxPlayers ?? 4 })
    });
    onRoom(normalizeRoom(data.room));
    markPresence(data.room.id, data.room.activeSessionId, "ONLINE").catch(() => undefined);
    onMessage(data.matched ? "대기 중인 방에 매칭되었습니다." : "퀵매치 방을 만들고 친구를 기다립니다.");
  } catch {
    onMessage("퀵매치에 실패했습니다.");
  }
}

export async function joinRoom(
  code: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>("/api/rooms/join", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ code })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(
      data.room.status === "PLAYING" ? `${data.room.code} 방에 관전자로 입장했습니다.` : `${data.room.code} 방에 입장했습니다.`
    );
    markPresence(data.room.id, data.room.activeSessionId, "ONLINE").catch(() => undefined);
  } catch {
    onMessage("방 코드를 확인해주세요.");
  }
}

export async function joinPublicRoom(
  roomID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/join`, session.sessionToken, {
      method: "POST"
    });
    onRoom(normalizeRoom(data.room));
    markPresence(data.room.id, data.room.activeSessionId, "ONLINE").catch(() => undefined);
    onMessage(data.room.status === "PLAYING" ? "진행 중인 방에 관전자로 입장했습니다." : "공개 방에 입장했습니다.");
  } catch {
    onMessage("공개 방 입장에 실패했습니다.");
  }
}

export async function spectateRoom(
  roomID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/spectate`, session.sessionToken, {
      method: "POST"
    });
    onRoom(normalizeRoom(data.room));
    onMessage("관전자로 입장했습니다.");
  } catch {
    onMessage("관전 입장에 실패했습니다.");
  }
}

export async function updateRoomOptions(
  roomID: string,
  options: RoomSettings,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/options`, session.sessionToken, {
      method: "PATCH",
      body: JSON.stringify(options)
    });
    onRoom(normalizeRoom(data.room));
    onMessage(
      `방 옵션을 적용했습니다. 턴 ${data.room.options.turnSeconds}초, 대기 ${data.room.options.maxWaitSeconds}초`
    );
  } catch {
    onMessage("방장만 옵션을 바꿀 수 있습니다.");
  }
}

export async function voteRoomRules(
  roomID: string,
  choices: Record<string, string>,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/rules/vote`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ choices })
    });
    onRoom(normalizeRoom(data.room));
    onMessage("원카드 룰 투표를 반영했습니다. Ready가 해제됩니다.");
  } catch {
    onMessage("로비 참가자만 원카드 룰에 투표할 수 있습니다.");
  }
}

export async function kickPlayer(
  roomID: string,
  userID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) return;
  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/kick`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ userId: userID })
    });
    onRoom(normalizeRoom(data.room));
    onMessage("선택한 사용자를 방에서 내보냈습니다.");
  } catch {
    onMessage("방장만 강퇴할 수 있습니다.");
  }
}

export async function transferHost(
  roomID: string,
  userID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) return;
  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/transfer-host`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ userId: userID })
    });
    onRoom(normalizeRoom(data.room));
    onMessage("방장을 위임했습니다.");
  } catch {
    onMessage("방장만 권한을 위임할 수 있습니다.");
  }
}

export async function sendChat(roomID: string, text: string, kind: "chat" | "emoji"): Promise<ChatMessage> {
  const guest = readPlayerSession();
  if (!guest) throw new Error("missing guest");
  const data = await authorizedJSON<{ message: ChatMessage }>(`/api/rooms/${roomID}/chat`, guest.sessionToken, {
    method: "POST",
    body: JSON.stringify({ text, kind })
  });
  return data.message;
}

export async function fetchRoomChat(roomID: string): Promise<ChatMessage[]> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/rooms/${roomID}/chat`);
  if (!response.ok) throw new Error("failed to fetch chat");
  const data = (await response.json()) as { messages: ChatMessage[] };
  return listOf(data.messages).map(normalizeChatMessage);
}

export async function fetchRoom(roomID: string): Promise<Room> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/rooms/${roomID}`);
  if (!response.ok) throw new Error("failed to fetch room");
  const data = (await response.json()) as { room: Room };
  return normalizeRoom(data.room);
}

export async function markPresence(
  roomID: string,
  sessionID: string | undefined,
  status: "ONLINE" | "DISCONNECTED"
) {
  const guest = readPlayerSession();
  if (!guest) return;
  await authorizedJSON<{ presence: Presence }>(`/api/rooms/${roomID}/presence`, guest.sessionToken, {
    method: "POST",
    body: JSON.stringify({ sessionId: sessionID, status })
  });
}

export async function fetchLeaderboard(gameId: string): Promise<LeaderboardRow[]> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/leaderboard?gameId=${gameId}&limit=5`);
  if (!response.ok) throw new Error("failed to fetch leaderboard");
  const data = (await response.json()) as { rows: LeaderboardRow[] };
  return listOf(data.rows);
}

export async function fetchTutorial(gameId: string): Promise<TutorialGuide> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/tutorials/${gameId}`);
  if (!response.ok) throw new Error("failed to fetch tutorial");
  const data = (await response.json()) as { guide: TutorialGuide };
  return data.guide;
}

export async function setReady(
  roomID: string,
  ready: boolean,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/ready`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ ready })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(ready ? "준비 완료했습니다." : "준비를 취소했습니다.");
  } catch {
    onMessage("Ready 변경에 실패했습니다.");
  }
}

export async function startGame(
  roomID: string,
  playerIds: string[],
  onRoom: (room: Room) => void,
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room; session: GameSession }>(
      `/api/rooms/${roomID}/start`,
      session.sessionToken,
      {
        method: "POST",
        body: JSON.stringify({ playerIds })
      }
    );
    onRoom(normalizeRoom(data.room));
    onSession(data.session);
    onMessage("선택된 플레이어로 게임을 시작했습니다.");
  } catch {
    onMessage("선택 인원과 Ready 상태를 확인해주세요.");
  }
}

export async function sendGameAction(
  sessionID: string,
  roomID: string | undefined,
  type: GameActionType,
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void,
  payload?: Record<string, unknown>
) {
  const guest = readPlayerSession();
  if (!guest || !roomID) return;

  try {
    const data = await authorizedJSON<{ session: GameSession }>(
      `/api/sessions/${sessionID}/actions`,
      guest.sessionToken,
      {
        method: "POST",
        body: JSON.stringify({
          roomId: roomID,
          type,
          payload,
          clientRequestId: crypto.randomUUID()
        })
      }
    );
    onSession(data.session);
    onMessage("액션이 반영되었습니다.");
  } catch {
    onMessage("지금은 해당 액션을 할 수 없습니다.");
  }
}

export async function sendGuessAction(
  sessionID: string,
  roomID: string | undefined,
  payload: {
    targetPlayerId: string;
    tileIndex: number;
    color: "black" | "white";
    value: number;
    joker: boolean;
    insertIndex: number;
  },
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void
) {
  const guest = readPlayerSession();
  if (!guest || !roomID || !payload.targetPlayerId) return;

  try {
    const data = await authorizedJSON<{ session: GameSession }>(
      `/api/sessions/${sessionID}/actions`,
      guest.sessionToken,
      {
        method: "POST",
        body: JSON.stringify({
          roomId: roomID,
          type: "davinci.guess",
          payload,
          clientRequestId: crypto.randomUUID()
        })
      }
    );
    onSession(data.session);
    onMessage("추측 결과가 반영되었습니다.");
  } catch {
    onMessage("추측할 수 없는 대상이거나 내 차례가 아닙니다.");
  }
}

export async function roomPostAction(
  roomID: string | undefined,
  action: "rematch" | "return-lobby" | "leave",
  onRoom: (room: Room | null) => void,
  afterAction: () => void,
  onMessage: (message: string) => void
) {
  const guest = readPlayerSession();
  if (!guest || !roomID) return;

  const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/${action}`, guest.sessionToken, {
    method: "POST"
  });

  if (action === "leave" || data.room.status === "CLOSED") {
    onRoom(null);
    onMessage("방에서 나왔습니다.");
  } else {
    onRoom(normalizeRoom(data.room));
    onMessage(action === "rematch" ? "다시 하기 준비 로비로 돌아왔습니다." : "로비로 돌아왔습니다.");
  }
  afterAction();
}

export async function fetchSession(sessionID: string, token: string): Promise<GameSession> {
  const data = await authorizedJSON<{ session: GameSession }>(`/api/sessions/${sessionID}`, token, {
    method: "GET"
  });
  return data.session;
}

export async function authorizedJSON<T>(path: string, token: string, init: RequestInit): Promise<T> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...init.headers
    }
  });

  if (!response.ok) {
    if (response.status === 401 && readAuthSession()?.sessionToken === token) {
      localStorage.removeItem(authStorageKey);
    }
    throw new Error(`request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}
