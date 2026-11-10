import assert from "node:assert/strict"
import test from "node:test"

import { presentMessages, stripHTML } from "./present.js"

test("presentMessages returns a page oldest first and prefers text content", () => {
  const page = presentMessages({
    page: 1,
    per_page: 2,
    total: 2,
    total_pages: 1,
    results: [
      { uuid: "new", created_at: "2026-11-10T12:00:00Z", sender_type: "agent", private: true, text_content: "note", author: { first_name: "Ada" } },
      { uuid: "old", created_at: "2026-11-10T11:00:00Z", sender_type: "contact", private: false, text_content: "", content: "<p>Hello<br>there</p>", author: { first_name: "Sam", last_name: "Lee" } }
    ]
  })
  assert.deepEqual(page.messages.map((m) => m.uuid), ["old", "new"])
  assert.equal(page.messages[0].text, "Hello there")
  assert.equal(page.messages[0].author, "Sam Lee")
  assert.equal(page.messages[1].private, true)
  assert.equal(page.order, "oldest_first")
})

test("stripHTML drops tags and decodes the entities notes use", () => {
  assert.equal(stripHTML("<p>a &amp; b</p>"), "a & b\n")
})
