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
  readonly contactProperties: ContactProperties;
  readonly topics: Topics;
  readonly templates: Templates;
  readonly broadcasts: Broadcasts;
  readonly suppressions: Suppressions;
  readonly automations: Automations;
  readonly ipPools: IPPools;
  readonly oauth: OAuth;

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
    this.contactProperties = new ContactProperties(this);
    this.topics = new Topics(this);
    this.templates = new Templates(this);
    this.broadcasts = new Broadcasts(this);
    this.suppressions = new Suppressions(this);
    this.automations = new Automations(this);
    this.ipPools = new IPPools(this);
    this.oauth = new OAuth(this);
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

  listReceived() {
    return this.client.request("GET", "/emails/received");
  }
  getReceived(id: string) {
    return this.client.request("GET", `/emails/received/${id}`);
  }
  receivedAttachments(id: string) {
    return this.client.request<{ data: AttachmentMeta[] }>("GET", `/emails/received/${id}/attachments`);
  }
  getReceivedAttachment(id: string, attachmentId: string) {
    return this.client.request<AttachmentMeta & { content: string }>(
      "GET",
      `/emails/received/${id}/attachments/${attachmentId}`
    );
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
  regions() {
    return this.client.request("GET", "/domains/regions");
  }
  get(id: string) {
    return this.client.request("GET", `/domains/${id}`);
  }
  verify(id: string) {
    return this.client.request("POST", `/domains/${id}/verify`);
  }
  claim(id: string) {
    return this.client.request("POST", `/domains/${id}/claim`);
  }
  confirmClaim(id: string) {
    return this.client.request("POST", `/domains/${id}/claim/confirm`);
  }
  setBIMI(id: string, body: { svg_url: string; selector?: string; vmc_url?: string }) {
    return this.client.request("POST", `/domains/${id}/bimi`, body);
  }
  setReceiving(id: string, enabled: boolean) {
    return this.client.request("POST", `/domains/${id}/receiving`, { enabled });
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
  create(email: string, opts?: { firstName?: string; lastName?: string; properties?: Record<string, unknown> }) {
    return this.client.request("POST", "/contacts", {
      email,
      first_name: opts?.firstName,
      last_name: opts?.lastName,
      properties: opts?.properties,
    });
  }
  update(id: string, body: { first_name?: string; last_name?: string; unsubscribed?: boolean; properties?: Record<string, unknown> }) {
    return this.client.request("PATCH", `/contacts/${id}`, body);
  }
  list() {
    return this.client.request("GET", "/contacts");
  }
  remove(id: string) {
    return this.client.request("DELETE", `/contacts/${id}`);
  }
  listTopics(contactId: string) {
    return this.client.request("GET", `/contacts/${contactId}/topics`);
  }
  setTopic(contactId: string, topicId: string, subscribed = true) {
    return this.client.request("PUT", `/contacts/${contactId}/topics/${topicId}`, {
      subscribed,
    });
  }
}

class ContactProperties {
  constructor(private client: Raisin) {}
  create(key: string, type: "string" | "number" = "string") {
    return this.client.request("POST", "/contact-properties", { key, type });
  }
  list() {
    return this.client.request("GET", "/contact-properties");
  }
  remove(id: string) {
    return this.client.request("DELETE", `/contact-properties/${id}`);
  }
}

class Topics {
  constructor(private client: Raisin) {}
  create(body: {
    name: string;
    description?: string;
    default_subscription?: "opt_in" | "opt_out";
  }) {
    return this.client.request("POST", "/topics", body);
  }
  list() {
    return this.client.request("GET", "/topics");
  }
  get(id: string) {
    return this.client.request("GET", `/topics/${id}`);
  }
  remove(id: string) {
    return this.client.request("DELETE", `/topics/${id}`);
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
    topic_id?: string;
    name?: string;
    scheduled_at?: string;
  }) {
    return this.client.request("POST", "/broadcasts", body);
  }
  send(id: string, body: { scheduled_at?: string; immediate?: boolean } = {}) {
    return this.client.request("POST", `/broadcasts/${id}/send`, body);
  }
  list() {
    return this.client.request("GET", "/broadcasts");
  }
}

class Suppressions {
  constructor(private client: Raisin) {}
  create(email: string, reason = "manual") {
    return this.client.request("POST", "/suppressions", { email, reason });
  }
  list() {
    return this.client.request("GET", "/suppressions");
  }
  remove(id: string) {
    return this.client.request("DELETE", `/suppressions/${id}`);
  }
}

class Automations {
  constructor(private client: Raisin) {}
  create(body: {
    name: string;
    trigger_type: string;
    steps?: { type: string; config?: Record<string, unknown> }[];
    trigger_filter?: Record<string, unknown>;
  }) {
    return this.client.request("POST", "/automations", body);
  }
  list() {
    return this.client.request("GET", "/automations");
  }
  get(id: string) {
    return this.client.request("GET", `/automations/${id}`);
  }
  enable(id: string, enabled: boolean) {
    return this.client.request("PATCH", `/automations/${id}`, { enabled });
  }
  update(
    id: string,
    body: {
      enabled?: boolean;
      name?: string;
      description?: string;
      trigger_type?: string;
      trigger_filter?: Record<string, unknown>;
      steps?: { type: string; config?: Record<string, unknown> }[];
    },
  ) {
    return this.client.request("PATCH", `/automations/${id}`, body);
  }
  remove(id: string) {
    return this.client.request("DELETE", `/automations/${id}`);
  }
  runs(id: string) {
    return this.client.request("GET", `/automations/${id}/runs`);
  }
}

class IPPools {
  constructor(private client: Raisin) {}
  create(name: string, region?: string) {
    return this.client.request("POST", "/ip-pools", { name, region });
  }
  list() {
    return this.client.request("GET", "/ip-pools");
  }
  get(id: string) {
    return this.client.request("GET", `/ip-pools/${id}`);
  }
  assignDomain(id: string, domainId: string) {
    return this.client.request("POST", `/ip-pools/${id}/assign-domain`, { domain_id: domainId });
  }
  pause(id: string) {
    return this.client.request("POST", `/ip-pools/${id}/pause`);
  }
  resume(id: string) {
    return this.client.request("POST", `/ip-pools/${id}/resume`);
  }
  warmupTick(id: string) {
    return this.client.request("POST", `/ip-pools/${id}/warmup/tick`);
  }
  remove(id: string) {
    return this.client.request("DELETE", `/ip-pools/${id}`);
  }
}

class OAuth {
  constructor(private client: Raisin) {}
  createApp(body: { name: string; redirect_uris: string[]; scopes?: string[] }) {
    return this.client.request("POST", "/oauth/apps", body);
  }
  listApps() {
    return this.client.request("GET", "/oauth/apps");
  }
  deleteApp(id: string) {
    return this.client.request("DELETE", `/oauth/apps/${id}`);
  }
  /** Exchange authorization code or refresh token (no API key). */
  async token(body: {
    grant_type: "authorization_code" | "refresh_token";
    client_id: string;
    client_secret: string;
    code?: string;
    redirect_uri?: string;
    refresh_token?: string;
  }) {
    try {
      const res = await fetch(`${this.client.baseUrl}/oauth/token`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "User-Agent": "raisin-node/0.1.0",
        },
        body: JSON.stringify(body),
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
      return { data: json, error: null };
    } catch (e: any) {
      return {
        data: null,
        error: { name: "network_error", message: e?.message ?? "network error", statusCode: 0 },
      };
    }
  }
}

export default Raisin;
