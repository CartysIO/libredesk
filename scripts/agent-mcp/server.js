import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js"
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js"

import { clientFromEnv } from "./client.js"
import { buildTools } from "./tools.js"

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

const server = new McpServer({ name: "libredesk", version: "0.1.0" })
for (const tool of buildTools(client)) {
  server.registerTool(tool.name, { description: tool.description, inputSchema: tool.inputSchema }, async (args) => {
    try {
      const data = await tool.run(args)
      return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] }
    } catch (err) {
      return { isError: true, content: [{ type: "text", text: err.message || "request failed" }] }
    }
  })
}

const transport = new StdioServerTransport()
await server.connect(transport)
