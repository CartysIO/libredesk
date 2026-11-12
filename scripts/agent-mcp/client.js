// HTTP client for the Libredesk agent API. API-key auth skips CSRF.
// The key belongs to one agent, and that agent's role is what the desk enforces.

const DEFAULT_URL = "https://libredesk-staging-6deb.up.railway.app"

export const STATUSES = ["Open", "On you", "On customer", "Snoozed", "Resolved", "Closed"]

export function createClient({ baseURL, apiKey, apiSecret, fetchImpl = globalThis.fetch, timeoutMs = 30000 } = {}) {
  const root = (baseURL || DEFAULT_URL).replace(/\/$/, "")
  if (!apiKey || !apiSecret) {
    throw new Error("LIBREDESK_AGENT_API_KEY and LIBREDESK_AGENT_API_SECRET are required")
  }

  async function request(method, path, { query, body } = {}) {
    const url = new URL(path, root + "/")
    if (query) {
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined && value !== null && value !== "") url.searchParams.set(key, String(value))
      }
    }
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), timeoutMs)
    let response
    try {
      response = await fetchImpl(url, {
        method,
        signal: controller.signal,
        headers: {
          Authorization: `token ${apiKey}:${apiSecret}`,
          Accept: "application/json",
          ...(body === undefined ? {} : { "Content-Type": "application/json" })
        },
        body: body === undefined ? undefined : JSON.stringify(body)
      })
    } catch (err) {
      if (err?.name === "AbortError") throw new Error(`${method} ${path} timed out`)
      throw new Error(`${method} ${path} failed: ${err.message}`)
    } finally {
      clearTimeout(timer)
    }

    const text = await response.text()
    let payload
    try {
      payload = text ? JSON.parse(text) : {}
    } catch {
      throw new Error(`${method} ${path} returned ${response.status} with a non-JSON body`)
    }
    if (!response.ok || payload.status === "error") {
      const message = payload.message || response.statusText || "request failed"
      throw new Error(`${response.status} ${message}`)
    }
    return payload.data
  }

  function conversationListQuery(page, pageSize, extra = {}) {
    return { page, page_size: pageSize, order: "desc", order_by: "last_message_at", ...extra }
  }

  return {
    listConversations({ status, page = 1, pageSize = 20 } = {}) {
      const query = conversationListQuery(page, pageSize)
      if (status) query.filters = statusFilter(status)
      return request("GET", "/api/v1/conversations/all", { query })
    },
    listUnassigned({ page = 1, pageSize = 20 } = {}) {
      return request("GET", "/api/v1/conversations/unassigned", {
        query: conversationListQuery(page, pageSize)
      })
    },
    listAssignedToMe({ page = 1, pageSize = 20 } = {}) {
      return request("GET", "/api/v1/conversations/assigned", {
        query: conversationListQuery(page, pageSize)
      })
    },
    getConversation(uuid) {
      return request("GET", `/api/v1/conversations/${encodeURIComponent(uuid)}`)
    },
    getMessages(uuid, { page = 1, pageSize = 20 } = {}) {
      return request("GET", `/api/v1/conversations/${encodeURIComponent(uuid)}/messages`, {
        query: { page, page_size: pageSize }
      })
    },
    searchConversations(query, { pageSize = 15 } = {}) {
      return request("GET", "/api/v1/search/conversations", { query: { query, page_size: pageSize } })
    },
    searchMessages(query, { pageSize = 15 } = {}) {
      return request("GET", "/api/v1/search/messages", { query: { query, page_size: pageSize } })
    },
    searchContacts(query) {
      return request("GET", "/api/v1/contacts/search", { query: { query } })
    },
    sendAgentMessage(uuid, message, { private: isPrivate = false, keepStatus = false } = {}) {
      return request("POST", `/api/v1/conversations/${encodeURIComponent(uuid)}/messages`, {
        body: {
          message: plainTextToHTML(message),
          private: isPrivate,
          sender_type: "agent",
          ...(keepStatus ? { keep_status: true } : {})
        }
      })
    },
    addNote(uuid, message) {
      return this.sendAgentMessage(uuid, message, { private: true })
    },
    sendReply(uuid, message, { keepStatus = false } = {}) {
      return this.sendAgentMessage(uuid, message, { private: false, keepStatus })
    },
    setStatus(uuid, status, snoozedUntil) {
      return request("PUT", `/api/v1/conversations/${encodeURIComponent(uuid)}/status`, {
        body: { status, ...(snoozedUntil ? { snoozed_until: snoozedUntil } : {}) }
      })
    },
    assign(uuid, assigneeID) {
      return request("PUT", `/api/v1/conversations/${encodeURIComponent(uuid)}/assignee/user`, {
        body: { assignee_id: assigneeID }
      })
    },
    listAgents(query) {
      return request("GET", "/api/v1/agents/compact", { query: { q: query, enabled: "true", type: "agent" } })
    },
    setPriority(uuid, priority) {
      return request("PUT", `/api/v1/conversations/${encodeURIComponent(uuid)}/priority`, {
        body: { priority }
      })
    },
    updateTags(uuid, tags, action = "add_tags") {
      return request("POST", `/api/v1/conversations/${encodeURIComponent(uuid)}/tags`, {
        body: { tags, action }
      })
    },
    unassign(uuid) {
      return request("PUT", `/api/v1/conversations/${encodeURIComponent(uuid)}/assignee/user/remove`, { body: {} })
    }
  }
}

export function statusFilter(status) {
  if (!STATUSES.includes(status)) {
    throw new Error(`status must be one of: ${STATUSES.join(", ")}`)
  }
  return JSON.stringify([
    { model: "conversation_statuses", field: "name", operator: "equals", value: status }
  ])
}

export function plainTextToHTML(text) {
  const escaped = String(text)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
  return `<p>${escaped.replace(/\n/g, "<br>")}</p>`
}

export function clientFromEnv(env = process.env) {
  return createClient({
    baseURL: env.LIBREDESK_AGENT_URL,
    apiKey: env.LIBREDESK_AGENT_API_KEY,
    apiSecret: env.LIBREDESK_AGENT_API_SECRET
  })
}
