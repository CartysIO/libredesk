// Avatar initials: first letter of the first name and of the last name ("Dana Levi" -> "DL").
// One name only gives one letter; nothing gives the fallback.
export const initials = (firstName, lastName, fallback = '?') => {
  const first = (firstName || '').trim()
  const last = (lastName || '').trim()
  const letters = `${first.charAt(0)}${last.charAt(0)}`.toUpperCase()
  return letters || fallback
}
