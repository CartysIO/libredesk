import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js"

import { buildTools } from "./tools.js"

/** Wire Libredesk tools onto an MCP server instance. */
export function registerLibredeskTools(server, client) {
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
}

export function createLibredeskMcpServer(client) {
  const server = new McpServer({ name: "libredesk", version: "0.2.0" })
  registerLibredeskTools(server, client)
  return server
}
