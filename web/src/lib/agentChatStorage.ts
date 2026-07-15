// Minimal stub for the deleted agentChatStorage module.
//
// Originally dev had `agentChatStorage.ts` for persisting NOFXi agent
// chat history in localStorage. dev deleted it during the merge into
// Individual. The AgentChatPage.tsx still imports from this path.
//
// Until a real persistence layer is wired up, these helpers exist as
// thin no-ops / in-memory shims so the page keeps compiling and runs
// without crashing. They DO NOT persist across reloads.

export interface AgentMessage {
  id: string
  role: string
  text: string
  time: string
  [key: string]: unknown
}

export function loadAgentMessages<T = AgentMessage>(
  _storage: Storage,
  _userId?: string
): { messages: T[]; version: number } {
  return { messages: [] as T[], version: 1 }
}

export function persistAgentMessages(
  _storage: Storage,
  _userId?: string | undefined,
  _messages?: AgentMessage[]
): void {
  // no-op: see file comment.
}

export function clearAgentMessages(_storage: Storage, _userId?: string): void {
  // no-op: see file comment.
}

export function migrateAgentMessages(
  _storage: Storage,
  _userId?: string
): void {
  // no-op: see file comment.
}

export function prepareAgentMessagesForPersistence<T extends AgentMessage>(
  messages: T[]
): T[] {
  // strip runtime-only fields, if any. Keep the public surface stable.
  return messages.map((m) => {
    const { id, role, text, time } = m
    return { ...m, id, role, text, time } as T
  })
}

export function getStoredAuthUserId(_storage?: Storage): string | undefined {
  return undefined
}

export function chatStorageKey(userId: string): string {
  return `nofx:agent:messages:${userId}`
}
