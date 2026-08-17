# Raisin MCP Server

An MCP stdio server that exposes the Raisin email API as tools.

## Installation

From the monorepo root:

```sh
pnpm install
pnpm --filter @raisin-run/mcp-server build
```

## Configuration

Set these environment variables in your MCP client configuration:

- `RAISIN_API_KEY` (required): your Raisin API key
- `RAISIN_BASE_URL` (optional): API origin; defaults to `https://api.raisin.run`

Example configuration after building:

```json
{
  "mcpServers": {
    "raisin": {
      "command": "node",
      "args": ["/absolute/path/to/raisin/packages/mcp-server/dist/index.js"],
      "env": {
        "RAISIN_API_KEY": "your-api-key"
      }
    }
  }
}
```

For local development, set `RAISIN_BASE_URL` to `http://localhost:18080`.

## Tools

- `send_email`
- `list_emails`
- `list_domains`
- `create_domain`
- `list_templates`
- `list_automations`
- `list_ip_pools`
