export function listOf<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

export function tail<T>(value: T[] | null | undefined, count: number): T[] {
  return listOf(value).slice(-count);
}

export function head<T>(value: T[] | null | undefined, count: number): T[] {
  return listOf(value).slice(0, count);
}
