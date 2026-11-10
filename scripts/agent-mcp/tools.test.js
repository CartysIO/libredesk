import assert from "node:assert/strict"
import test from "node:test"

import { buildTools } from "./tools.js"

function tool(name, client) {
  const found = buildTools(client).find((item) => item.name === name)
  if (!found) throw new Error(`missing tool ${name}`)
  return found
}

test("assign resolves an agent email to an id", async () => {
  const calls = []
  const client = {
    async listAgents(query) {
      calls.push(["list", query])
      return [
        { id: 7, email: "omer@clyr.io", first_name: "Omer" },
        { id: 8, email: "eran@clyr.io", first_name: "Eran" }
      ]
    },
    async assign(uuid, id) {
      calls.push(["assign", uuid, id])
    }
  }
  const result = await tool("assign", client).run({ uuid: "conv-1", email: "Omer@clyr.io" })
  assert.deepEqual(result, { uuid: "conv-1", agent_id: 7 })
  assert.deepEqual(calls, [
    ["list", "Omer@clyr.io"],
    ["assign", "conv-1", 7]
  ])
})

test("assign refuses when neither id nor email is given", async () => {
  await assert.rejects(tool("assign", {}).run({ uuid: "conv-1" }), /agent_id or email/)
})

test("set_status requires a duration when snoozing", async () => {
  let called = false
  const client = {
    async setStatus() {
      called = true
    }
  }
  await assert.rejects(tool("set_status", client).run({ uuid: "conv-1", status: "Snoozed" }), /snooze_for/)
  assert.equal(called, false)
})

test("set_status passes the duration through", async () => {
  const client = {
    async setStatus(uuid, status, snoozeFor) {
      assert.equal(uuid, "conv-1")
      assert.equal(status, "Snoozed")
      assert.equal(snoozeFor, "4h")
    }
  }
  const result = await tool("set_status", client).run({ uuid: "conv-1", status: "Snoozed", snooze_for: "4h" })
  assert.deepEqual(result, { uuid: "conv-1", status: "Snoozed", snooze_for: "4h" })
})
