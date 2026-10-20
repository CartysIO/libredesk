// The ai_classify_attribute action value is [attribute key, instructions, per-value descriptions as JSON].
// Trailing empty parts are dropped; blank descriptions are left out.
export const classifyActionValue = (key, instructions, descriptions) => {
  const filled = Object.fromEntries(
    Object.entries(descriptions || {})
      .map(([value, text]) => [value, (text || '').trim()])
      .filter(([, text]) => text)
  )
  const value = [key || '', (instructions || '').trim()]
  if (Object.keys(filled).length > 0) return [...value, JSON.stringify(filled)]
  return value[1] ? value : [value[0]]
}
