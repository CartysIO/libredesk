import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js"

import { clientFromEnv } from "./client.js"
import { createLibredeskMcpServer } from "./register.js"

// Stdout is the MCP protocol. Logs go to stderr.
function fail(message) {
  console.error(message)
  process.exit(1)
}

let client
try {
  client = clientFromEnv()
} catch (err) {
  fail(err.message)
}

const server = createLibredeskMcpServer(client)

const transport = new StdioServerTransport()
await server.connect(transport)
