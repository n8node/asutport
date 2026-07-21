"use client";

import { useCallback, useEffect, useState } from "react";

import { OrgTypeGate } from "@/components/cabinet/OrgTypeGate";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { VendorShell } from "@/components/VendorShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";
import { authFetch } from "@/lib/auth-session";

type ApiError = { error?: { message?: string } };
type Product = { id: string; name: string; slug: string };
type DocSource = {
  id: string;
  filename: string;
  version: string;
  status: string;
  page_count: number;
  chunk_count: number;
  product_name?: string;
  error_message?: string;
};
type RAGHit = {
  chunk_id: string;
  doc_source_id: string;
  content_md: string;
  page_number: number;
  score: number;
  from_keyword: boolean;
  product_name?: string;
  version: string;
};

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await authFetch(`/api/v1${path}`, options);
  const body = (await res.json()) as { data?: T } & ApiError;
  if (!res.ok) throw new Error(body.error?.message || "request failed");
  return body.data as T;
}

export default function VendorDocsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [sources, setSources] = useState<DocSource[]>([]);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [productId, setProductId] = useState("");
  const [version, setVersion] = useState("1.0");
  const [file, setFile] = useState<File | null>(null);
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<RAGHit[]>([]);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const reload = useCallback(async () => {
    const [p, d] = await Promise.all([api<Product[]>("/vendor/products"), api<DocSource[]>("/vendor/docs")]);
    setProducts(p || []);
    setSources(d || []);
    if (p?.[0] && !productId) setProductId(p[0].id);
  }, [productId]);

  useEffect(() => {
    reload().catch((e) => setErr(e instanceof Error ? e.message : "load failed"));
  }, [reload]);

  async function createProduct() {
    setBusy(true);
    setErr("");
    try {
      await api("/vendor/products", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), slug: slug.trim(), kind: "other" }),
      });
      setName("");
      setSlug("");
      await reload();
      setMsg("Продукт создан");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "error");
    } finally {
      setBusy(false);
    }
  }

  async function upload() {
    if (!file || !productId) return;
    setBusy(true);
    setErr("");
    try {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("product_id", productId);
      fd.append("version", version);
      const res = await authFetch("/api/v1/vendor/docs/upload", { method: "POST", body: fd });
      const body = (await res.json()) as ApiError;
      if (!res.ok) throw new Error(body.error?.message || "upload failed");
      setFile(null);
      await reload();
      setMsg("Документ в очереди на векторизацию");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "error");
    } finally {
      setBusy(false);
    }
  }

  async function search() {
    setBusy(true);
    try {
      const res = await api<RAGHit[]>("/vendor/rag/search", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query, product_ids: productId ? [productId] : [], version, top_k: 8 }),
      });
      setHits(res || []);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "search failed");
    } finally {
      setBusy(false);
    }
  }

  async function openPage(sourceId: string, page: number) {
    try {
      const res = await api<{ url: string }>(`/vendor/docs/${sourceId}/pages/${page}/url`);
      if (res?.url) window.open(res.url, "_blank", "noopener,noreferrer");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "page url failed");
    }
  }

  return (
    <VendorShell activePath="/app/vendor/docs" pageTitle="Документация">
      <VendorPageHeader
        title="Документация продукта"
        subtitle="PDF и текст → эмбеддинги → поиск по мануалам с привязкой к странице."
      />
      <OrgTypeGate allowed={["manufacturer"]} title="Раздел для производителей">
        {msg ? <p className="mb-3 text-[13px] text-[#2f5d1e]">{msg}</p> : null}
        {err ? <p className="mb-3 text-[13px] text-[#9b1c1c]">{err}</p> : null}

        <div className="grid gap-4 lg:grid-cols-2">
          <DashboardPanel title="Продукты">
            <div className="space-y-2">
              <input
                className="w-full rounded-lg border border-[#d7d2ca] px-3 py-2 text-[13px]"
                placeholder="Название"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              <input
                className="w-full rounded-lg border border-[#d7d2ca] px-3 py-2 font-mono text-[12px]"
                placeholder="slug"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
              />
              <button
                type="button"
                disabled={busy}
                className="rounded-lg bg-[#18212f] px-3 py-2 text-[12px] text-white"
                onClick={() => void createProduct()}
              >
                Создать
              </button>
              <ul className="mt-2 space-y-1 text-[12px] text-[#5f6b7a]">
                {products.map((p) => (
                  <li key={p.id}>
                    {p.name} <span className="font-mono text-[11px]">({p.slug})</span>
                  </li>
                ))}
              </ul>
            </div>
          </DashboardPanel>

          <DashboardPanel title="Загрузка">
            <div className="space-y-2">
              <select
                className="w-full rounded-lg border border-[#d7d2ca] px-3 py-2 text-[13px]"
                value={productId}
                onChange={(e) => setProductId(e.target.value)}
              >
                <option value="">Продукт</option>
                {products.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
              <input
                className="w-full rounded-lg border border-[#d7d2ca] px-3 py-2 font-mono text-[12px]"
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                placeholder="версия"
              />
              <input
                type="file"
                accept=".pdf,.md,.txt"
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
              <button
                type="button"
                disabled={busy || !file || !productId}
                className="rounded-lg bg-[#185fa5] px-3 py-2 text-[12px] text-white disabled:opacity-50"
                onClick={() => void upload()}
              >
                Загрузить и векторизовать
              </button>
            </div>
          </DashboardPanel>
        </div>

        <div className="mt-4">
          <DashboardPanel title="Индексированные документы">
            {sources.length === 0 ? (
              <DashboardEmpty title="Пока нет документов">
                Создайте продукт и загрузите PDF или Markdown.
              </DashboardEmpty>
            ) : (
              <ul className="space-y-2 text-[13px]">
                {sources.map((s) => (
                  <li key={s.id} className="flex flex-wrap justify-between gap-2 border-t border-[#eeeae3] pt-2">
                    <span>
                      {s.filename} · {s.product_name} · v{s.version}
                    </span>
                    <span className="font-mono text-[11px] text-[#6f6a62]">
                      {s.status} · {s.page_count} стр. / {s.chunk_count} чанков
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </DashboardPanel>
        </div>

        <div className="mt-4">
          <DashboardPanel title="Поиск по документации">
            <div className="flex gap-2">
              <input
                className="w-full rounded-lg border border-[#d7d2ca] px-3 py-2 text-[13px]"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Вопрос по мануалу"
              />
              <button
                type="button"
                className="rounded-lg bg-[#18212f] px-3 py-2 text-[12px] text-white"
                onClick={() => void search()}
              >
                Найти
              </button>
            </div>
            <ul className="mt-3 space-y-2">
              {hits.map((h) => (
                <li key={h.chunk_id} className="rounded-lg border border-[#eeeae3] bg-[#faf8f5] p-3 text-[12px]">
                  <div className="mb-1 text-[11px] text-[#6f6a62]">
                    {h.product_name} · стр. {h.page_number} · {h.score.toFixed(3)} ·{" "}
                    {h.from_keyword ? "keyword" : "vector"}{" "}
                    <button
                      type="button"
                      className="text-[#185fa5] underline"
                      onClick={() => void openPage(h.doc_source_id, h.page_number)}
                    >
                      страница
                    </button>
                  </div>
                  <p className="whitespace-pre-wrap">{h.content_md.slice(0, 400)}</p>
                </li>
              ))}
            </ul>
          </DashboardPanel>
        </div>
      </OrgTypeGate>
    </VendorShell>
  );
}
