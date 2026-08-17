#!/usr/bin/env node

import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";

const USER_AGENT = "raisin-mcp-server/0.1.0";
const DEFAULT_BASE_URL = "https://api.raisin.run";

type ToolArguments = Record<string, unknown>;

const tools = [
  {
    name: "send_email",
    description: "Send an email through Raisin.",
    inputSchema: {
      type: "object",
      properties: {
        from: { type: "string", description: "Sender email address." },
        to: { type: "string", description: "Recipient email address." },
        subject: { type: "string", description: "Email subject." },
        html: { type: "string", description: "HTML email body." },
      },
      required: ["from", "to", "subject", "html"],
      additionalProperties: false,
    },
  },
  {
    name: "list_emails",
    description: "List sent emails.",
    inputSchema: emptyInputSchema(),
  },
  {
    name: "list_domains",
    description: "List sending domains.",
    inputSchema: emptyInputSchema(),
  },
  {
    name: "create_domain",
    description: "Create a sending domain.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Domain name, such as example.com." },
        region: { type: "string", description: "Sending region. Defaults to us-east-1." },
      },
      required: ["name"],
      additionalProperties: false,
    },
  },
  {
    name: "list_templates",
    description: "List email templates.",
    inputSchema: emptyInputSchema(),
  },
  {
    name: "list_automations",
    description: "List email automations.",
    inputSchema: emptyInputSchema(),
  },
  {
    name: "list_ip_pools",
    description: "List dedicated IP pools.",
    inputSchema: emptyInputSchema(),
  },
];

const server = new Server(
  { name: "@raisin-run/mcp-server", version: "0.1.0" },
  { capabilities: { tools: {} } },
);

server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools }));

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const args = (request.params.arguments ?? {}) as ToolArguments;

  try {
    let result: unknown;
    switch (request.params.name) {
      case "send_email":
        result = await apiRequest("POST", "/emails", {
          from: requiredString(args, "from"),
          to: requiredString(args, "to"),
          subject: requiredString(args, "subject"),
          html: requiredString(args, "html"),
        });
        break;
      case "list_emails":
        result = await apiRequest("GET", "/emails");
        break;
      case "list_domains":
        result = await apiRequest("GET", "/domains");
        break;
      case "create_domain":
        result = await apiRequest("POST", "/domains", {
          name: requiredString(args, "name"),
          region: optionalString(args, "region") ?? "us-east-1",
        });
        break;
      case "list_templates":
        result = await apiRequest("GET", "/templates");
        break;
      case "list_automations":
        result = await apiRequest("GET", "/automations");
        break;
      case "list_ip_pools":
        result = await apiRequest("GET", "/ip-pools");
        break;
      default:
        throw new Error(`Unknown tool: ${request.params.name}`);
    }

    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  } catch (error) {
    return {
      isError: true,
      content: [
        {
          type: "text",
          text: error instanceof Error ? error.message : String(error),
        },
      ],
    };
  }
});

async function apiRequest(
  method: "GET" | "POST",
  path: string,
  body?: Record<string, string>,
): Promise<unknown> {
  const apiKey = process.env.RAISIN_API_KEY;
  if (!apiKey) {
    throw new Error("RAISIN_API_KEY is required");
  }

  const baseUrl = (process.env.RAISIN_BASE_URL || DEFAULT_BASE_URL).replace(/\/+$/, "");
  const response = await fetch(`${baseUrl}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "User-Agent": USER_AGENT,
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  const text = await response.text();
  let result: unknown = null;
  if (text) {
    try {
      result = JSON.parse(text);
    } catch {
      result = text;
    }
  }

  if (!response.ok) {
    const detail = typeof result === "string" ? result : JSON.stringify(result);
    throw new Error(`Raisin API returned ${response.status}: ${detail}`);
  }
  return result;
}

function requiredString(args: ToolArguments, name: string): string {
  const value = args[name];
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(`${name} must be a non-empty string`);
  }
  return value;
}

function optionalString(args: ToolArguments, name: string): string | undefined {
  const value = args[name];
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(`${name} must be a non-empty string`);
  }
  return value;
}

function emptyInputSchema() {
  return {
    type: "object",
    properties: {},
    additionalProperties: false,
  };
}

async function main() {
  await server.connect(new StdioServerTransport());
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
