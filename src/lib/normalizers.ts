import type { ChatMessage, Room } from "../domain/types";
import { tail } from "./collections";
import { safeNickname } from "./playerIdentity";

export function normalizeRoom(room: Room): Room {
  return {
    ...room,
    visibility: room.visibility ?? "PRIVATE",
    participants: (room.participants ?? []).map((participant, index) => ({
      ...participant,
      user: normalizeUserRef(participant.user, `Player_${index + 1}`)
    })),
    spectators: (room.spectators ?? []).map((spectator, index) => ({
      ...spectator,
      user: normalizeUserRef(spectator.user, `Spectator_${index + 1}`)
    })),
    playingPlayerIds: room.playingPlayerIds ?? [],
    ruleVotes: room.ruleVotes ?? [],
    gameRules: room.gameRules ?? {},
    ruleMessages: room.ruleMessages ?? [],
    options: {
      turnSeconds: room.options?.turnSeconds ?? 60,
      maxWaitSeconds: room.options?.maxWaitSeconds ?? 180,
      autoStart: room.options?.autoStart ?? false,
      allowSpectators: room.options?.allowSpectators ?? true
    }
  };
}

export function normalizeChatMessage(message: ChatMessage | null | undefined): ChatMessage {
  const source = message ?? ({} as ChatMessage);
  return {
    ...source,
    id: typeof source.id === "string" && source.id.length > 0 ? source.id : crypto.randomUUID(),
    text: typeof source.text === "string" ? source.text : "",
    kind: source.kind === "emoji" ? "emoji" : "chat",
    user: normalizeUserRef(source.user, "Player")
  };
}

export function appendChatMessage(
  current: ChatMessage[] | null | undefined,
  message: ChatMessage | null | undefined
) {
  const normalized = normalizeChatMessage(message);
  return [...tail(current, 49).filter((item) => item.id !== normalized.id), normalized];
}

export function normalizeUserRef(user: { id: string; nickname: string } | null | undefined, fallback: string) {
  return {
    id: typeof user?.id === "string" && user.id.length > 0 ? user.id : fallback,
    nickname: safeNickname(user?.nickname, fallback)
  };
}
