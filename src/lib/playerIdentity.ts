export function safeNickname(value: unknown, fallback: string) {
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : fallback;
}

export function nicknameInitial(value: unknown, fallback: string) {
  return safeNickname(value, fallback).slice(0, 2);
}
