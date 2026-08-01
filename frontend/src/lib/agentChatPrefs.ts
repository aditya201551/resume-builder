const AUTO_ACCEPT_KEY = 'agent-chat:auto-accept'

// Global, not per-resume: it's a preference about how the user wants to
// review the assistant's proposals, not something tied to one resume's data
// (unlike resumeDraftStorage's per-resume drafts).
export function loadAutoAccept(): boolean {
  try {
    return localStorage.getItem(AUTO_ACCEPT_KEY) === 'true'
  } catch {
    return false
  }
}

export function saveAutoAccept(value: boolean) {
  try {
    localStorage.setItem(AUTO_ACCEPT_KEY, String(value))
  } catch {
    // localStorage can throw (quota, private mode) — this is a nice-to-have
    // preference, not worth crashing the chat panel over.
  }
}
