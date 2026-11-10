// Turn desk payloads into short records. The raw conversation and message
// objects carry HTML, attachment URLs, and fields an agent does not need.

const TEXT_LIMIT = 500

export function presentConversationList(page) {
  const results = Array.isArray(page?.results) ? page.results.map(presentListItem) : []
  return {
    page: page?.page ?? 1,
    per_page: page?.per_page ?? results.length,
    total: page?.total ?? results.length,
    total_pages: page?.total_pages ?? 1,
    conversations: results
  }
}

export function presentListItem(item) {
  return {
    uuid: item.uuid,
    reference_number: item.reference_number,
    subject: item.subject ?? "",
    status: item.status ?? "",
    priority: item.priority ?? "",
    inbox: item.inbox_name ?? "",
    contact: personName(item.contact),
    contact_email: item.contact?.email ?? "",
    organization: item.organization_name ?? "",
    assignee_id: item.assigned_user_id ?? null,
    unread: item.unread_message_count ?? 0,
    last_message_at: item.last_message_at ?? null,
    last_message: clip(item.last_message)
  }
}

export function presentConversation(conv) {
  return {
    uuid: conv.uuid,
    reference_number: conv.reference_number,
    subject: conv.subject ?? "",
    status: conv.status ?? "",
    priority: conv.priority ?? "",
    inbox: conv.inbox_name ?? "",
    channel: conv.inbox_channel ?? "",
    contact_id: conv.contact_id,
    contact: personName(conv.contact),
    contact_email: conv.contact?.email ?? "",
    organization: conv.organization_name ?? conv.contact?.organization_name ?? "",
    assignee_id: conv.assigned_user_id ?? null,
    tags: conv.tags ?? [],
    custom_attributes: conv.custom_attributes ?? {},
    last_message_at: conv.last_message_at ?? null
  }
}

export function presentMessages(page) {
  const results = Array.isArray(page?.results) ? page.results : []
  // The API returns newest first. Chronological order is easier to read.
  const messages = results.slice().reverse().map(presentMessage)
  return {
    page: page?.page ?? 1,
    per_page: page?.per_page ?? messages.length,
    total: page?.total ?? messages.length,
    total_pages: page?.total_pages ?? 1,
    order: "oldest_first",
    messages
  }
}

export function presentMessage(message) {
  const author = message.author || {}
  return {
    uuid: message.uuid,
    created_at: message.created_at,
    sender_type: message.sender_type,
    private: Boolean(message.private),
    author: [author.first_name, author.last_name].filter(Boolean).join(" "),
    text: clip(message.text_content || stripHTML(message.content || ""))
  }
}

export function presentSearchHit(item) {
  return {
    uuid: item.uuid,
    reference_number: item.reference_number,
    subject: item.subject ?? "",
    status: item.status ?? "",
    contact: personName(item.contact),
    contact_email: item.contact?.email ?? "",
    last_message: clip(item.last_message),
    last_message_at: item.last_message_at ?? null
  }
}

export function presentMessageHit(item) {
  return {
    message_uuid: item.uuid,
    conversation_uuid: item.conversation_uuid,
    reference_number: item.conversation_reference_number,
    subject: item.conversation_subject ?? "",
    status: item.conversation_status ?? "",
    private: Boolean(item.private),
    text: clip(item.text_content || item.snippet),
    created_at: item.created_at,
    contact: personName(item.contact)
  }
}

export function presentNote(message) {
  return { uuid: message.uuid, private: Boolean(message.private), created_at: message.created_at }
}

function personName(contact) {
  if (!contact) return ""
  return [contact.first_name, contact.last_name].filter(Boolean).join(" ")
}

export function clip(value) {
  if (value === null || value === undefined) return ""
  const text = String(value).replace(/\s+/g, " ").trim()
  if (text.length <= TEXT_LIMIT) return text
  return text.slice(0, TEXT_LIMIT) + "…"
}

export function stripHTML(html) {
  return String(html)
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/p>/gi, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&nbsp;/g, " ")
}
