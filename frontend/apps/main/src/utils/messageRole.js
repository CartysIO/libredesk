// Who a thread message is from, read from the message's own fields (never guessed from names):
// 'note' (private), 'ai' (AI assistant reply), 'customer' (contact / visitor), 'agent' (human, ours).
export function messageRole(message) {
  if (!message) return 'agent'
  if (message.private) return 'note'
  if (message.author?.type === 'ai_assistant') return 'ai'
  if (
    message.type === 'incoming' ||
    message.sender_type === 'contact' ||
    message.author?.type === 'contact' ||
    message.author?.type === 'visitor'
  ) {
    return 'customer'
  }
  return 'agent'
}
