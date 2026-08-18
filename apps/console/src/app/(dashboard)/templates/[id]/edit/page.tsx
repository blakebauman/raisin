"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { BackLink, Field, MailFrame, Msg, PageHeader, SectionLabel, TableSkeleton } from "@/components/ui";

type Template = {
  id: string;
  name: string;
  subject?: string;
  html?: string;
  react_source?: string;
  editor_json?: { blocks?: { type: string; text?: string; href?: string }[] };
};

type Block = { type: string; text?: string; href?: string };

const STARTER_BLOCKS: Block[] = [
  { type: "heading", text: "Hello {{name}}" },
  { type: "paragraph", text: "Thanks for signing up." },
  { type: "button", text: "Open app", href: "https://app.raisin.run" },
];

function blocksToHTML(blocks: Block[]) {
  return blocks
    .map((b) => {
      if (b.type === "heading") return `<h1 style="font-family:Georgia,serif;font-size:28px;color:#111">${b.text || ""}</h1>`;
      if (b.type === "button")
        return `<p><a href="${b.href || "#"}" style="display:inline-block;background:#f97316;color:#000;padding:12px 18px;border-radius:6px;text-decoration:none;font-weight:600">${b.text || "Click"}</a></p>`;
      return `<p style="font-family:system-ui,sans-serif;color:#333;line-height:1.5">${b.text || ""}</p>`;
    })
    .join("\n");
}

function blocksToReact(blocks: Block[]) {
  const body = blocks
    .map((b) => {
      if (b.type === "heading") return `      <Heading>${JSON.stringify(b.text || "")}</Heading>`;
      if (b.type === "button")
        return `      <Button href={${JSON.stringify(b.href || "#")}}>${JSON.stringify(b.text || "Click")}</Button>`;
      return `      <Text>${JSON.stringify(b.text || "")}</Text>`;
    })
    .join("\n");
  return `import { Html, Body, Heading, Text, Button } from '@react-email/components';\n\nexport default function Email() {\n  return (\n    <Html>\n      <Body>\n${body}\n      </Body>\n    </Html>\n  );\n}\n`;
}

export default function TemplateEditorPage() {
  const params = useParams();
  const id = params.id as string;
  const router = useRouter();
  const [tpl, setTpl] = useState<Template | null>(null);
  const [blocks, setBlocks] = useState<Block[]>(STARTER_BLOCKS);
  const [name, setName] = useState("");
  const [subject, setSubject] = useState("");
  const [msg, setMsg] = useState("");
  const [tab, setTab] = useState<"visual" | "react" | "html">("visual");

  useEffect(() => {
    if (!id) return;
    apiFetch<Template>(`/templates/${id}`)
      .then((t) => {
        setTpl(t);
        setName(t.name);
        setSubject(t.subject || "");
        if (t.editor_json?.blocks?.length) setBlocks(t.editor_json.blocks);
      })
      .catch((e) => setMsg(e.message));
  }, [id]);

  const html = blocksToHTML(blocks);
  const react = blocksToReact(blocks);

  async function save() {
    setMsg("");
    try {
      await apiFetch(`/templates/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          name,
          subject,
          html,
          react_source: react,
          editor_json: { blocks },
        }),
      });
      setMsg("Saved");
    } catch (e: unknown) {
      setMsg(e instanceof Error ? e.message : "save failed");
    }
  }

  async function publish() {
    await save();
    await apiFetch(`/templates/${id}/publish`, { method: "POST", body: "{}" });
    setMsg("Published");
    router.refresh();
  }

  if (!tpl) return <TableSkeleton rows={10} />;

  return (
    <div>
      <BackLink href="/templates">Templates</BackLink>
      <div className="mt-3">
        <PageHeader
          title={name || "Editor"}
          description="Edit blocks, then save or publish."
          actions={
            <div className="flex gap-2">
              <button type="button" onClick={save} className="btn-secondary">
                Save
              </button>
              <button type="button" onClick={publish} className="btn-primary">
                Publish
              </button>
            </div>
          }
        />
      </div>

      <div className="mb-6 grid max-w-2xl gap-3 sm:grid-cols-2">
        <Field label="Name" htmlFor="ed-name">
          <input
            id="ed-name"
            className="field"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </Field>
        <Field label="Subject" htmlFor="ed-subject">
          <input
            id="ed-subject"
            className="field"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
          />
        </Field>
      </div>
      <Msg>{msg}</Msg>

      <div className="mb-3 flex gap-1 text-[13px]">
        {(["visual", "react", "html"] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`h-8 rounded-full px-3.5 capitalize ${
              tab === t ? "bg-white/[0.06] text-zinc-50" : "text-[var(--muted)] hover:text-zinc-300"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="grid gap-8 lg:grid-cols-2">
        <section>
          {tab === "visual" && (
            <div className="space-y-3">
              {blocks.map((b, i) => (
                <div key={i} className="grid gap-2 border-t border-[var(--border)] pt-3 first:border-t-0 first:pt-0">
                  <select
                    className="field"
                    value={b.type}
                    onChange={(e) => {
                      const next = [...blocks];
                      next[i] = { ...next[i], type: e.target.value };
                      setBlocks(next);
                    }}
                  >
                    <option value="heading">Heading</option>
                    <option value="paragraph">Paragraph</option>
                    <option value="button">Button</option>
                  </select>
                  <input
                    className="field"
                    value={b.text || ""}
                    onChange={(e) => {
                      const next = [...blocks];
                      next[i] = { ...next[i], text: e.target.value };
                      setBlocks(next);
                    }}
                  />
                  {b.type === "button" && (
                    <input
                      className="field font-mono text-xs"
                      value={b.href || ""}
                      onChange={(e) => {
                        const next = [...blocks];
                        next[i] = { ...next[i], href: e.target.value };
                        setBlocks(next);
                      }}
                      placeholder="https://"
                    />
                  )}
                </div>
              ))}
              <button
                type="button"
                className="btn-ghost text-xs"
                onClick={() => setBlocks([...blocks, { type: "paragraph", text: "New block" }])}
              >
                Add block
              </button>
            </div>
          )}
          {tab === "react" && (
            <pre className="overflow-auto whitespace-pre-wrap font-mono text-xs text-zinc-300">{react}</pre>
          )}
          {tab === "html" && (
            <textarea className="field min-h-80 font-mono text-xs" value={html} readOnly />
          )}
        </section>
        <section>
          <SectionLabel>Preview</SectionLabel>
          <MailFrame html={html} />
        </section>
      </div>
    </div>
  );
}
