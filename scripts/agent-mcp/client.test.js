import assert from "node:assert/strict"
import test from "node:test"

import { createClient, plainTextToHTML, statusFilter } from "./client.js"

function fakeFetch(handler) {
  const calls = []
  const fetchImpl = async (url, init) => {
    calls.push({ url: String(url), init })
    const result = await handler(url, init)
    return {
      ok: result.status >= 200 && result.status < 300,
      status: result.status,
      statusText: result.statusText || "",
      text: async () => JSON.stringify(result.body)
    }
  }
  return { fetchImpl, calls }
}

test("sends the API key as a token and reads the success envelope", async () => {
  const { fetchImpl, calls } = fakeFetch(async () => ({
    status: 200,
    body: { status: "success", data: { uuid: "abc", subject: "Hi" } }
  }))
  const client = createClient({
    baseURL: "https://desk.example/",
    apiKey: "key",
    apiSecret: "secret",
    fetchImpl
  })
  const got = await client.getConversation("abc")
  assert.deepEqual(got, { uuid: "abc", subject: "Hi" })
  assert.equal(calls[0].url, "https://desk.example/api/v1/conversations/abc")
  assert.equal(calls[0].init.headers.Authorization, "token key:secret")
  assert.equal(calls[0].init.body, undefined)
})

test("surfaces the desk error message", async () => {
  const { fetchImpl } = fakeFetch(async () => ({
    status: 403,
    body: { status: "error", message: "Permission denied" }
  }))
  const client = createClient({ apiKey: "key", apiSecret: "secret", fetchImpl })
  await assert.rejects(client.getConversation("abc"), /403 Permission denied/)
})

test("listConversations filters by a known status", async () => {
  const { fetchImpl, calls } = fakeFetch(async () => ({
    status: 200,
    body: { status: "success", data: { results: [], total: 0, page: 1, per_page: 20, total_pages: 0 } }
  }))
  const client = createClient({ baseURL: "https://desk.example", apiKey: "k", apiSecret: "s", fetchImpl })
  await client.listConversations({ status: "On you", page: 2, pageSize: 5 })
  const url = new URL(calls[0].url)
  assert.equal(url.searchParams.get("page"), "2")
  assert.equal(url.searchParams.get("page_size"), "5")
  assert.deepEqual(JSON.parse(url.searchParams.get("filters")), [
    { model: "conversation_statuses", field: "name", operator: "equals", value: "On you" }
  ])
})

test("addNote posts a private agent note as escaped HTML", async () => {
  const { fetchImpl, calls } = fakeFetch(async () => ({
    status: 200,
    body: { status: "success", data: { uuid: "m1", private: true } }
  }))
  const client = createClient({ baseURL: "https://desk.example", apiKey: "k", apiSecret: "s", fetchImpl })
  await client.addNote("conv", "Hello <b>\nthere")
  assert.equal(calls[0].init.method, "POST")
  assert.deepEqual(JSON.parse(calls[0].init.body), {
    message: "<p>Hello &lt;b&gt;<br>there</p>",
    private: true,
    sender_type: "agent"
  })
})

test("statusFilter rejects an unknown status", () => {
  assert.throws(() => statusFilter("pending"), /status must be one of/)
})

test("plainTextToHTML escapes markup", () => {
  assert.equal(plainTextToHTML("a & b"), "<p>a &amp; b</p>")
})

test("createClient refuses to start without a secret", () => {
  assert.throws(() => createClient({ apiKey: "k" }), /API_SECRET/)
})
