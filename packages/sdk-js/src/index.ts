export type SendEmailParams = {
  from: string;
  to: string | string[];
  cc?: string | string[];
  bcc?: string | string[];
  replyTo?: string | string[];
  subject?: string;
  html?: string;
  text?: string;
  headers?: Record<string, string>;
  tags?: Record<string, string>;
  scheduledAt?: string;
  templateId?: string;
  attachments?: AttachmentIn[];
  /** React Email component — rendered to HTML in Node before send */
  react?: unknown;
};

export type AttachmentIn = {
  filename: string;
  content: string; // base64
  content_type?: string;
  content_id?: string;
};

export type AttachmentMeta = {
  id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  content_id?: string;
};

export type Email = {
  id: string;
  from: string;
  to: string[];
  subject: string;
  status: string;
  created_at: string;
};

export type RaisinOptions = {
  apiKey: string;
  baseUrl?: string;
};

function asArray(v?: string | string[]): string[] | undefined {
  if (v == null) return undefined;
  return Array.isArray(v) ? v : [v];
}

export class Raisin {
  readonly apiKey: string;
  readonly baseUrl: string;
  readonly emails: Emails;
  readonly domains: Domains;
  readonly apiKeys: ApiKeys;
  readonly webhooks: Webhooks;
  readonly contacts: Contacts;
  readonly templates: Templates;
  readonly broadcasts: Broadcasts;

  constructor(apiKeyOrOpts: string | RaisinOptions) {
    if (typeof apiKeyOrOpts === "string") {
      this.apiKey = apiKeyOrOpts;
      this.baseUrl = "https://api.raisin.run";
    } else {
      this.apiKey = apiKeyOrOpts.apiKey;
      this.baseUrl = apiKeyOrOpts.baseUrl ?? "https://api.raisin.run";
    }
    this.emails = new Emails(this);
    this.domains = new Domains(this);
    this.apiKeys = new ApiKeys(this);
    this.webhooks = new Webhooks(this);
    this.contacts = new Contacts(this);
    this.templates = new Templates(this);
    this.broadcasts = new Broadcasts(this);
  }

  async request<T>(
    method: string,
    path: string,
    body?: unknown,
    headers?: Record<string, string>
  ): Promise<{ data: T | null; error: { name: string; message: string; statusCode: number } | null }> {
    try {
      const res = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers: {
          Authorization: `Bearer ${this.apiKey}`,
          "Content-Type": "application/json",
          "User-Agent": "raisin-node/0.1.0",
          ...headers,
        },
        body: body == null ? undefined : JSON.stringify(body),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        return {
          data: null,
          error: {
            name: (json as any).name ?? "error",
            message: (json as any).message ?? res.statusText,
            statusCode: res.status,
          },
        };
      }
      return { data: json as T, error: null };
    } catch (e: any) {
      return {
        data: null,
        error: { name: "network_error", message: e?.message ?? "network error", statusCode: 0 },
      };
    }
  }
}

class Emails {
  constructor(private client: Raisin) {}

  async send(params: SendEmailParams, opts?: { idempotencyKey?: string }) {
    let html = params.html;
    if (params.react != null && html == null) {
      try {
        const { render } = await import("@react-email/render");
        html = await render(params.react as any);
      } catch (e: any) {
        return {
          data: null,
          error: {
            name: "react_render_error",
            message: e?.message ?? "failed to render react email",
            statusCode: 400,
          },
        };
      }
    }
    const body = {
      from: params.from,
      to: asArray(params.to),
      cc: asArray(params.cc),
      bcc: asArray(params.bcc),
      reply_to: asArray(params.replyTo),
      subject: params.subject,
      html,
      text: params.text,
      headers: params.headers,
      tags: params.tags,
      scheduled_at: params.scheduledAt,
      template_id: params.templateId,
      attachments: params.attachments,
    };
    return this.client.request<Email>("POST", "/emails", body, {
      ...(opts?.idempotencyKey ? { "Idempotency-Key": opts.idempotencyKey } : {}),
    });
  }

  async attachments(emailId: string) {
    return this.client.request<{ data: AttachmentMeta[] }>("GET", `/emails/${emailId}/attachments`);
  }

  async getAttachment(emailId: string, attachmentId: string) {
    return this.client.request<AttachmentMeta & { content: string }>(
      "GET",
      `/emails/${emailId}/attachments/${attachmentId}`
    );
  }

  async get(id: string) {
    return this.client.request<Email>("GET", `/emails/${id}`);
  }

  async list(cursor?: string) {
    const q = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
    return this.client.request<{ data: Email[]; next_cursor: string }>("GET", `/emails${q}`);
  }

  async cancel(id: string) {
    return this.client.request<Email>("POST", `/emails/${id}/cancel`);
  }

  async batch(params: SendEmailParams[]) {
    const body = params.map((p) => ({
      from: p.from,
      to: asArray(p.to),
      subject: p.subject,
      html: p.html,
      text: p.text,
    }));
    return this.client.request<{ data: Email[] }>("POST", "/emails/batch", body);
  }
}

class Domains {
  constructor(private client: Raisin) {}
  create(name: string, region?: string) {
    return this.client.request("POST", "/domains", { name, region });
  }
  list() {
    return this.client.request("GET", "/domains");
  }
  get(id: string) {
    return this.client.request("GET", `/domains/${id}`);
  }
  verify(id: string) {
    return this.client.request("POST", `/domains/${id}/verify`);
  }
  remove(id: string) {
    return this.client.request("DELETE", `/domains/${id}`);
  }
}

class ApiKeys {
  constructor(private client: Raisin) {}
  create(name: string, permission?: string) {
    return this.client.request("POST", "/api-keys", { name, permission });
  }
  list() {
    return this.client.request("GET", "/api-keys");
  }
  remove(id: string) {
    return this.client.request("DELETE", `/api-keys/${id}`);
  }
}

class Webhooks {
  constructor(private client: Raisin) {}
  create(endpoint: string, events: string[]) {
    return this.client.request("POST", "/webhooks", { endpoint, events });
  }
  list() {
    return this.client.request("GET", "/webhooks");
  }
  remove(id: string) {
    return this.client.request("DELETE", `/webhooks/${id}`);
  }
  listEvents(id: string) {
    return this.client.request("GET", `/webhooks/${id}/events`);
  }
  listAttempts(webhookId: string, eventId: string) {
    return this.client.request("GET", `/webhooks/${webhookId}/events/${eventId}/attempts`);
  }
}

/**
 * Verify a Raisin webhook signature header (`Raisin-Signature: t=…,v1=…`).
 * Uses HMAC-SHA256 over `${t}.${rawBody}` with the endpoint signing secret.
 */
export async function verifyWebhookSignature(
  secret: string,
  signatureHeader: string,
  body: string | ArrayBuffer | Uint8Array,
  opts?: { maxSkewSeconds?: number },
): Promise<boolean> {
  const parts: Record<string, string> = {};
  for (const p of signatureHeader.split(",")) {
    const [k, v] = p.trim().split("=", 2);
    if (k && v) parts[k] = v;
  }
  const ts = parts.t;
  const sig = parts.v1;
  if (!ts || !sig) return false;

  const sec = Number(ts);
  if (!Number.isFinite(sec)) return false;
  const maxSkew = opts?.maxSkewSeconds ?? 300;
  if (Math.abs(Math.floor(Date.now() / 1000) - sec) > maxSkew) return false;

  const enc = new TextEncoder();
  let bodyBytes: Uint8Array;
  if (typeof body === "string") {
    bodyBytes = enc.encode(body);
  } else if (body instanceof ArrayBuffer) {
    bodyBytes = new Uint8Array(body);
  } else {
    bodyBytes = body;
  }

  const key = await crypto.subtle.importKey(
    "raw",
    enc.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const prefix = enc.encode(`${ts}.`);
  const msg = new Uint8Array(prefix.length + bodyBytes.length);
  msg.set(prefix, 0);
  msg.set(bodyBytes, prefix.length);
  const mac = await crypto.subtle.sign("HMAC", key, msg);
  const expected = [...new Uint8Array(mac)].map((b) => b.toString(16).padStart(2, "0")).join("");
  return timingSafeEqual(expected, sig);
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let out = 0;
  for (let i = 0; i < a.length; i++) out |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return out === 0;
}

class Contacts {
  constructor(private client: Raisin) {}
  create(email: string, opts?: { firstName?: string; lastName?: string }) {
    return this.client.request("POST", "/contacts", {
      email,
      first_name: opts?.firstName,
      last_name: opts?.lastName,
    });
  }
  list() {
    return this.client.request("GET", "/contacts");
  }
  remove(id: string) {
    return this.client.request("DELETE", `/contacts/${id}`);
  }
}

class Templates {
  constructor(private client: Raisin) {}
  create(body: { name: string; subject?: string; html?: string; text?: string }) {
    return this.client.request("POST", "/templates", body);
  }
  list() {
    return this.client.request("GET", "/templates");
  }
  publish(id: string) {
    return this.client.request("POST", `/templates/${id}/publish`);
  }
}

class Broadcasts {
  constructor(private client: Raisin) {}
  create(body: {
    from: string;
    subject: string;
    html?: string;
    segment_id?: string;
    name?: string;
  }) {
    return this.client.request("POST", "/broadcasts", body);
  }
  send(id: string) {
    return this.client.request("POST", `/broadcasts/${id}/send`);
  }
  list() {
    return this.client.request("GET", "/broadcasts");
  }
}

export default Raisin;
