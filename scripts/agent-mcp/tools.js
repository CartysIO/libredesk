import { z } from "zod"

import { STATUSES } from "./client.js"
import {
  presentConversation,
  presentConversationList,
  presentMessageHit,
  presentMessages,
  presentNote,
  presentSearchHit,
  presentSentMessage
} from "./present.js"

const statusSchema = z.enum(STATUSES).describe("Conversation status. One of: " + STATUSES.join(", "))

// buildTools returns the tools exposed to external agents (Grok, Cursor, and so on).
// Customer-visible replies need messages:write on the API key's role.
export function buildTools(client) {
  return [
    {
      name: "list_unassigned",
      description:
        "List conversations with no agent assignee, newest activity first. A good inbox triage queue for an autonomous agent.",
      inputSchema: {
        page: z.number().int().min(1).optional(),
        page_size: z.number().int().min(1).max(50).optional()
      },
      async run({ page, page_size }) {
        return presentConversationList(await client.listUnassigned({ page, pageSize: page_size }))
      }
    },
    {
      name: "list_assigned_to_me",
      description:
        "List conversations assigned to the API key's agent, newest activity first. Use this to work through the bot's own queue.",
      inputSchema: {
        page: z.number().int().min(1).optional(),
        page_size: z.number().int().min(1).max(50).optional()
      },
      async run({ page, page_size }) {
        return presentConversationList(await client.listAssignedToMe({ page, pageSize: page_size }))
      }
    },
    {
      name: "list_conversations",
      description:
        "List conversations, newest activity first. Pass a status to limit the list. Use the uuid from a result with get_conversation or get_messages.",
      inputSchema: {
        status: statusSchema.optional(),
        page: z.number().int().min(1).optional().describe("Page number, starting at 1"),
        page_size: z.number().int().min(1).max(50).optional()
      },
      async run({ status, page, page_size }) {
        const pageData = await client.listConversations({ status, page, pageSize: page_size })
        return presentConversationList(pageData)
      }
    },
    {
      name: "get_conversation",
      description: "Get one conversation by its uuid: subject, status, contact, organization, and tags. Does not include the message transcript; call get_messages for that.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid")
      },
      async run({ uuid }) {
        return presentConversation(await client.getConversation(uuid))
      }
    },
    {
      name: "get_messages",
      description:
        "Get messages for a conversation, oldest first within the page. Page 1 is the latest page. Private notes are included and marked private.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        page: z.number().int().min(1).optional(),
        page_size: z.number().int().min(1).max(50).optional()
      },
      async run({ uuid, page, page_size }) {
        return presentMessages(await client.getMessages(uuid, { page, pageSize: page_size }))
      }
    },
    {
      name: "search",
      description:
        "Search conversations and their messages. The query must be at least 3 characters. A reference number, subject, or a phrase from a message all work.",
      inputSchema: {
        query: z.string().min(3).describe("Search text, at least 3 characters"),
        what: z.enum(["conversations", "messages", "both"]).optional().describe("What to search. Defaults to both")
      },
      async run({ query, what = "both" }) {
        const out = {}
        if (what === "conversations" || what === "both") {
          const page = await client.searchConversations(query)
          out.conversations = (page.results || []).map(presentSearchHit)
          out.conversations_has_more = Boolean(page.has_more)
        }
        if (what === "messages" || what === "both") {
          const page = await client.searchMessages(query)
          out.messages = (page.results || []).map(presentMessageHit)
          out.messages_has_more = Boolean(page.has_more)
        }
        return out
      }
    },
    {
      name: "search_contacts",
      description: "Search contacts by name or email. The query must be at least 3 characters.",
      inputSchema: {
        query: z.string().min(3)
      },
      async run({ query }) {
        const rows = await client.searchContacts(query)
        const list = Array.isArray(rows) ? rows : []
        return list.map((row) => ({
          id: row.id,
          name: [row.first_name, row.last_name].filter(Boolean).join(" "),
          email: row.email ?? "",
          phone_number: row.phone_number ?? "",
          external_user_id: row.external_user_id ?? ""
        }))
      }
    },
    {
      name: "send_reply",
      description:
        "Send a customer-visible reply on a conversation. Email and chat channels deliver it to the contact. Requires messages:write. By default the conversation status may move to On customer unless keep_status is true.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        message: z.string().min(1).describe("Reply text (plain text; converted to HTML)"),
        keep_status: z
          .boolean()
          .optional()
          .describe("When true, do not change status to On customer when sending")
      },
      async run({ uuid, message, keep_status }) {
        return presentSentMessage(await client.sendReply(uuid, message, { keepStatus: Boolean(keep_status) }))
      }
    },
    {
      name: "add_note",
      description:
        "Add a private internal note. The customer does not see it and it is not emailed. Requires messages:write_private.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        message: z.string().min(1).describe("Note text")
      },
      async run({ uuid, message }) {
        return presentNote(await client.addNote(uuid, message))
      }
    },
    {
      name: "set_status",
      description:
        "Change a conversation's status. Snoozed also needs snooze_for, a Go duration such as 4h or 30m, counted from now.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        status: statusSchema,
        snooze_for: z
          .string()
          .regex(/^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/)
          .optional()
          .describe("Required when status is Snoozed. A duration from now, such as 30m, 4h, or 1h30m")
      },
      async run({ uuid, status, snooze_for }) {
        if (status === "Snoozed" && !snooze_for) throw new Error("snooze_for is required when status is Snoozed")
        await client.setStatus(uuid, status, snooze_for)
        return { uuid, status, snooze_for: snooze_for || "" }
      }
    },
    {
      name: "assign",
      description: "Assign a conversation to an agent, by agent id or by the agent's email.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        agent_id: z.number().int().positive().optional(),
        email: z.string().email().optional()
      },
      async run({ uuid, agent_id, email }) {
        const assigneeID = await resolveAssignee(client, agent_id, email)
        await client.assign(uuid, assigneeID)
        return { uuid, agent_id: assigneeID }
      }
    },
    {
      name: "unassign",
      description: "Remove the user assignee from a conversation, leaving it in the unassigned queue.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid")
      },
      async run({ uuid }) {
        await client.unassign(uuid)
        return { uuid, assignee_id: null }
      }
    },
    {
      name: "set_priority",
      description:
        "Set conversation priority by name (for example Low, Medium, High — use the same labels as in Admin → Priorities).",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        priority: z.string().min(1).describe("Priority label")
      },
      async run({ uuid, priority }) {
        await client.setPriority(uuid, priority)
        return { uuid, priority }
      }
    },
    {
      name: "update_tags",
      description: "Add, remove, or replace conversation tags by name.",
      inputSchema: {
        uuid: z.string().min(1).describe("Conversation uuid"),
        tags: z.array(z.string().min(1)).min(1).describe("Tag names"),
        action: z
          .enum(["add_tags", "remove_tags", "set_tags"])
          .optional()
          .describe("Defaults to add_tags")
      },
      async run({ uuid, tags, action = "add_tags" }) {
        await client.updateTags(uuid, tags, action)
        return { uuid, tags, action }
      }
    }
  ]
}

async function resolveAssignee(client, agentID, email) {
  if (agentID) return agentID
  if (!email) throw new Error("agent_id or email is required")
  const rows = await client.listAgents(email)
  const list = Array.isArray(rows) ? rows : []
  const match = list.find((agent) => String(agent.email || "").toLowerCase() === email.toLowerCase())
  if (!match) throw new Error(`no enabled agent with email ${email}`)
  return match.id
}
