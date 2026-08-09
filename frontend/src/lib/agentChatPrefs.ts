const AUTO_ACCEPT_KEY = 'agent-chat:auto-accept'

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
  }
}
