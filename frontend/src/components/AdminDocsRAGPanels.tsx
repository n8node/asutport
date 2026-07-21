"use client";

import { useCallback, useEffect, useState } from "react";

import { authFetch } from "@/lib/auth-session";

type ApiError = { error?: { message?: string } };

type LLMSettings = {
  enabled: boolean;
  provider: string;
  base_url: string;
  has_api_key: boolean;
  qualify_model: string;
  answer_model: string;
  vision_model: string;
  kb_model: string;
  embed_model: string;
  embed_dim: number;
  last_test_ok: boolean;
  last_test_latency_ms: number;
  last_test_at?: string;
  env_key_present: boolean;
};

type Product = {
  id: string;
  name: string;
  slug: string;
  kind: string;
  manufacturer_org_id: string;
};

type DocSource = {
  id: string;
  filename: string;
  version: string;
  status: string;
  page_count: number;
  chunk_count: number;
  product_name?: string;
  product_slug?: string;
  error_message?: string;
  tokens_total: number;
  content_hash: string;
};

type RAGHit = {
  chunk_id: string;
  content_md: string;
  page_number: number;
  score: number;
  from_keyword: boolean;
  filename?: string;
  product_name?: string;
  version: string;
};

const inputClass =
  "mt-1.5 w-full rounded-lg border border-[#d7d2ca] bg-white px-3 py-2 text-[13px] text-[#18212f] outline-none focus:border-[#2563eb] focus:ring-1 focus:ring-[#2563eb]";
const monoInputClass = `${inputClass} font-mono text-[12px]`;
const labelClass = "block text-[12px] font-medium text-[#4b5563]";

async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await authFetch(`/api/v1${path}`, options);
  const body = (await response.json()) as { data?: T } & ApiError;
  if (!response.ok) {
    throw new Error(body.error?.message || "request failed");
  }
  return body.data as T;
}

export function AdminDocsRAGPanels() {
  const [llm, setLlm] = useState<LLMSettings | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [models, setModels] = useState<string[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [sources, setSources] = useState<DocSource[]>([]);
  const [orgs, setOrgs] = useState<Array<{ id: string; name: string }>>([]);
  const [selectedOrg, setSelectedOrg] = useState("");
  const [productName, setProductName] = useState("");
  const [productSlug, setProductSlug] = useState("");
  const [uploadProductId, setUploadProductId] = useState("");
  const [uploadVersion, setUploadVersion] = useState("1.0");
  const [file, setFile] = useState<File | null>(null);
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<RAGHit[]>([]);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");

  const load = useCallback(async () => {
    setError("");
    const [llmData, orgList] = await Promise.all([
      api<LLMSettings>("/admin/settings/llm"),
      api<Array<{ id: string; name: string; type: string }>>("/admin/orgs?type=manufacturer&limit=100").catch(() => []),
    ]);
    setLlm(llmData);
    const manufacturers = (Array.isArray(orgList) ? orgList : [])
      .filter((o) => !o.type || o.type === "manufacturer")
      .map((o) => ({ id: o.id, name: o.name }));
    setOrgs(manufacturers);
    if (!selectedOrg && manufacturers[0]) {
      setSelectedOrg(manufacturers[0].id);
    }
  }, [selectedOrg]);

  const loadDocs = useCallback(async (orgId: string) => {
    if (!orgId) {
      setProducts([]);
      setSources([]);
      return;
    }
    const [prods, docs] = await Promise.all([
      api<Product[]>(`/admin/products?manufacturer_org_id=${orgId}`),
      api<DocSource[]>(`/admin/docs?manufacturer_org_id=${orgId}`),
    ]);
    setProducts(prods || []);
    setSources(docs || []);
    if (prods?.[0] && !uploadProductId) {
      setUploadProductId(prods[0].id);
    }
  }, [uploadProductId]);

  useEffect(() => {
    load().catch((e) => setError(e instanceof Error ? e.message : "load failed"));
  }, [load]);

  useEffect(() => {
    if (selectedOrg) {
      loadDocs(selectedOrg).catch((e) => setError(e instanceof Error ? e.message : "docs load failed"));
    }
  }, [selectedOrg, loadDocs]);

  async function saveLLM() {
    if (!llm) return;
    setBusy("save-llm");
    setError("");
    try {
      const body: Record<string, unknown> = {
        enabled: llm.enabled,
        provider: llm.provider,
        base_url: llm.base_url,
        qualify_model: llm.qualify_model,
        answer_model: llm.answer_model,
        vision_model: llm.vision_model,
        kb_model: llm.kb_model,
        embed_model: llm.embed_model,
      };
      if (apiKey.trim()) body.api_key = apiKey.trim();
      const next = await api<LLMSettings>("/admin/settings/llm", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      setLlm(next);
      setApiKey("");
      setMessage("Настройки шлюза ИИ сохранены");
    } catch (e) {
      setError(e instanceof Error ? e.message : "save failed");
    } finally {
      setBusy("");
    }
  }

  async function testLLM() {
    if (!llm) return;
    setBusy("test-llm");
    setError("");
    try {
      const res = await api<{ ok: boolean; latency_ms: number; message: string }>("/admin/settings/llm/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          base_url: llm.base_url,
          api_key: apiKey.trim() || undefined,
        }),
      });
      await load();
      setMessage(res.ok ? `Соединение OK (${res.latency_ms} мс)` : res.message);
    } catch (e) {
      setError(e instanceof Error ? e.message : "test failed");
    } finally {
      setBusy("");
    }
  }

  async function refreshModels() {
    setBusy("models");
    setError("");
    try {
      const res = await api<{ models: string[]; recommended: Record<string, string> }>("/admin/settings/llm/models");
      setModels(res.models || []);
      setMessage(`Загружено моделей: ${res.models?.length ?? 0}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "models failed");
    } finally {
      setBusy("");
    }
  }

  async function applyRecommended() {
    if (!llm || models.length === 0) return;
    setBusy("rec");
    try {
      const res = await api<{ models: string[]; recommended: Record<string, string> }>("/admin/settings/llm/models");
      const rec = res.recommended || {};
      setLlm({
        ...llm,
        qualify_model: rec.qualify_model || llm.qualify_model,
        answer_model: rec.answer_model || llm.answer_model,
        vision_model: rec.vision_model || llm.vision_model,
        kb_model: rec.kb_model || llm.kb_model,
        embed_model: rec.embed_model || llm.embed_model,
      });
      setModels(res.models || models);
      setMessage("Рекомендованные модели подставлены — сохраните");
    } catch (e) {
      setError(e instanceof Error ? e.message : "recommended failed");
    } finally {
      setBusy("");
    }
  }

  async function createProduct() {
    if (!selectedOrg || !productName.trim() || !productSlug.trim()) return;
    setBusy("product");
    try {
      await api("/admin/products", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          manufacturer_org_id: selectedOrg,
          name: productName.trim(),
          slug: productSlug.trim(),
          kind: "other",
        }),
      });
      setProductName("");
      setProductSlug("");
      await loadDocs(selectedOrg);
      setMessage("Продукт создан");
    } catch (e) {
      setError(e instanceof Error ? e.message : "create product failed");
    } finally {
      setBusy("");
    }
  }

  async function uploadDoc() {
    if (!file || !uploadProductId) return;
    setBusy("upload");
    setError("");
    try {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("product_id", uploadProductId);
      fd.append("version", uploadVersion);
      fd.append("manufacturer_org_id", selectedOrg);
      const response = await authFetch("/api/v1/admin/docs/upload", { method: "POST", body: fd });
      const body = (await response.json()) as { data?: DocSource } & ApiError;
      if (!response.ok) throw new Error(body.error?.message || "upload failed");
      setFile(null);
      setMessage(`Документ принят: ${body.data?.status}`);
      await loadDocs(selectedOrg);
    } catch (e) {
      setError(e instanceof Error ? e.message : "upload failed");
    } finally {
      setBusy("");
    }
  }

  async function searchRAG() {
    if (query.trim().length < 2) return;
    setBusy("search");
    try {
      const res = await api<RAGHit[]>("/admin/rag/search", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          query,
          product_ids: uploadProductId ? [uploadProductId] : [],
          version: uploadVersion,
          top_k: 8,
        }),
      });
      setHits(res || []);
      setMessage(`Найдено фрагментов: ${res?.length ?? 0}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "search failed");
    } finally {
      setBusy("");
    }
  }

  if (!llm) {
    return <p className="text-[13px] text-[#6f6a62]">Загрузка настроек ИИ…</p>;
  }

  const modelsUnlocked = llm.has_api_key && llm.last_test_ok;

  return (
    <div className="space-y-3">
      {message ? (
        <div className="rounded-lg border border-[#c8e0c0] bg-[#f3faf0] px-3 py-2 text-[13px] text-[#2f5d1e]">{message}</div>
      ) : null}
      {error ? (
        <div className="rounded-lg border border-[#f0c0c0] bg-[#fff5f5] px-3 py-2 text-[13px] text-[#9b1c1c]">{error}</div>
      ) : null}

      <section id="llm" className="rounded-[12px] border border-[#e8e4dc] bg-white p-5">
        <h2 className="text-[15px] font-medium text-[#18212f]">Шлюз ИИ (платформа)</h2>
        <p className="mt-1 text-[12px] text-[#6f6a62]">
          Ключ хранится в БД (шифр). При выключении используется ключ из окружения сервера.
          Размерность эмбеддингов: {llm.embed_dim}.
        </p>
        {llm.env_key_present ? (
          <p className="mt-2 rounded-lg border border-[#e8e4dc] bg-[#faf8f5] px-3 py-2 text-[12px] text-[#6f6a62]">
            В окружении сервера уже задан ключ — он используется как fallback.
          </p>
        ) : null}

        <label className="mt-4 flex items-center gap-2 text-[13px]">
          <input
            type="checkbox"
            checked={llm.enabled}
            onChange={(e) => setLlm({ ...llm, enabled: e.target.checked })}
          />
          <span>Хранить credentials платформы в БД</span>
        </label>

        <label className={`mt-3 ${labelClass}`}>Base URL</label>
        <input
          className={monoInputClass}
          value={llm.base_url}
          onChange={(e) => setLlm({ ...llm, base_url: e.target.value })}
          placeholder="https://openrouter.ai/api/v1"
        />

        <label className={`mt-3 ${labelClass}`}>API-ключ</label>
        <input
          className={monoInputClass}
          type="password"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          placeholder={llm.has_api_key ? "Оставьте пустым, чтобы не менять" : "Вставьте ключ"}
        />

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded-lg bg-[#18212f] px-3 py-2 text-[12px] font-medium text-white disabled:opacity-50"
            disabled={busy === "save-llm"}
            onClick={() => void saveLLM()}
          >
            {busy === "save-llm" ? "Сохранение…" : "Сохранить"}
          </button>
          <button
            type="button"
            className="rounded-lg border border-[#d7d2ca] bg-white px-3 py-2 text-[12px] font-medium text-[#18212f] disabled:opacity-50"
            disabled={busy === "test-llm" || (!llm.has_api_key && !apiKey.trim())}
            onClick={() => void testLLM()}
          >
            {busy === "test-llm" ? "Проверка…" : "Проверить соединение"}
          </button>
        </div>
        <p className="mt-2 text-[11px] text-[#8a857d]">
          Сначала сохраните ключ, затем успешный тест — после этого можно выбирать модели.
          {llm.last_test_ok ? ` Последний тест OK (${llm.last_test_latency_ms} мс).` : ""}
        </p>

        <div className={`mt-5 rounded-[10px] border p-4 ${modelsUnlocked ? "border-[#e8e4dc]" : "border-dashed border-[#d7d2ca] opacity-70"}`}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-[14px] font-medium text-[#18212f]">Модели по ролям</h3>
            <div className="flex gap-2">
              <button
                type="button"
                className="rounded-lg border border-[#d7d2ca] px-2.5 py-1.5 text-[11px] font-medium disabled:opacity-50"
                disabled={!modelsUnlocked || busy === "models"}
                onClick={() => void refreshModels()}
              >
                Обновить список
              </button>
              <button
                type="button"
                className="rounded-lg border border-[#d7d2ca] px-2.5 py-1.5 text-[11px] font-medium disabled:opacity-50"
                disabled={!modelsUnlocked || models.length === 0 || busy === "rec"}
                onClick={() => void applyRecommended()}
              >
                Применить рекомендуемые
              </button>
            </div>
          </div>
          <p className="mt-2 rounded-lg border border-[#c8d8f0] bg-[#f0f4fb] px-3 py-2 text-[11px] leading-snug text-[#1d3a6b]">
            Дешёвая модель — квалификация (SGR); умная — ответы и статьи; vision — парсинг страниц;
            embeddings — только text-embedding-3-large (3072).
          </p>
          <datalist id="asutport-llm-models">
            {models.map((m) => (
              <option key={m} value={m} />
            ))}
          </datalist>
          {(
            [
              ["qualify_model", "Квалификация (SGR)"],
              ["answer_model", "Ответ агента"],
              ["vision_model", "Vision (документация)"],
              ["kb_model", "Черновики базы знаний"],
              ["embed_model", "Embeddings (3072)"],
            ] as const
          ).map(([key, label]) => (
            <div key={key} className="mt-3">
              <label className={labelClass}>{label}</label>
              <input
                className={monoInputClass}
                list="asutport-llm-models"
                disabled={!modelsUnlocked}
                value={llm[key]}
                onChange={(e) => setLlm({ ...llm, [key]: e.target.value })}
              />
            </div>
          ))}
          <button
            type="button"
            className="mt-4 rounded-lg bg-[#185fa5] px-3 py-2 text-[12px] font-medium text-white disabled:opacity-50"
            disabled={!modelsUnlocked || busy === "save-llm"}
            onClick={() => void saveLLM()}
          >
            Сохранить модели
          </button>
        </div>
      </section>

      <section id="docs" className="rounded-[12px] border border-[#e8e4dc] bg-white p-5">
        <h2 className="text-[15px] font-medium text-[#18212f]">Документация и RAG</h2>
        <p className="mt-1 text-[12px] text-[#6f6a62]">
          Загрузка PDF/MD/TXT → S3 → извлечение текста → эмбеддинги → гибридный поиск.
        </p>

        <label className={`mt-3 ${labelClass}`}>Производитель</label>
        <select
          className={inputClass}
          value={selectedOrg}
          onChange={(e) => {
            setSelectedOrg(e.target.value);
            setUploadProductId("");
          }}
        >
          <option value="">Выберите организацию</option>
          {orgs.map((o) => (
            <option key={o.id} value={o.id}>
              {o.name}
            </option>
          ))}
        </select>

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          <div>
            <label className={labelClass}>Новый продукт — название</label>
            <input className={inputClass} value={productName} onChange={(e) => setProductName(e.target.value)} />
          </div>
          <div>
            <label className={labelClass}>Slug</label>
            <input className={monoInputClass} value={productSlug} onChange={(e) => setProductSlug(e.target.value)} />
          </div>
          <div className="flex items-end">
            <button
              type="button"
              className="w-full rounded-lg border border-[#d7d2ca] px-3 py-2 text-[12px] font-medium disabled:opacity-50"
              disabled={!selectedOrg || busy === "product"}
              onClick={() => void createProduct()}
            >
              Создать продукт
            </button>
          </div>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          <div>
            <label className={labelClass}>Продукт для загрузки</label>
            <select className={inputClass} value={uploadProductId} onChange={(e) => setUploadProductId(e.target.value)}>
              <option value="">—</option>
              {products.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} ({p.slug})
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className={labelClass}>Версия</label>
            <input className={monoInputClass} value={uploadVersion} onChange={(e) => setUploadVersion(e.target.value)} />
          </div>
          <div>
            <label className={labelClass}>Файл</label>
            <input
              className="mt-1.5 block w-full text-[12px]"
              type="file"
              accept=".pdf,.md,.txt,application/pdf,text/plain,text/markdown"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </div>
        </div>
        <button
          type="button"
          className="mt-3 rounded-lg bg-[#18212f] px-3 py-2 text-[12px] font-medium text-white disabled:opacity-50"
          disabled={!file || !uploadProductId || busy === "upload"}
          onClick={() => void uploadDoc()}
        >
          {busy === "upload" ? "Загрузка…" : "Загрузить и векторизовать"}
        </button>

        <div className="mt-5 overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-[12px]">
            <thead className="text-[#8a857d]">
              <tr className="border-b border-[#eeeae3]">
                <th className="py-2 pr-2 font-medium">Файл</th>
                <th className="py-2 pr-2 font-medium">Продукт</th>
                <th className="py-2 pr-2 font-medium">Версия</th>
                <th className="py-2 pr-2 font-medium">Статус</th>
                <th className="py-2 pr-2 font-medium">Стр./чанки</th>
              </tr>
            </thead>
            <tbody>
              {sources.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-3 text-[#6f6a62]">
                    Документов пока нет
                  </td>
                </tr>
              ) : (
                sources.map((s) => (
                  <tr key={s.id} className="border-b border-[#f3f0ea]">
                    <td className="py-2 pr-2 font-mono text-[11px]">{s.filename}</td>
                    <td className="py-2 pr-2">{s.product_name}</td>
                    <td className="py-2 pr-2 font-mono">{s.version}</td>
                    <td className="py-2 pr-2">
                      <span className="rounded-full border border-[#d7d2ca] px-2 py-0.5">{s.status}</span>
                      {s.error_message ? (
                        <span className="mt-1 block text-[10px] text-[#9b1c1c]">{s.error_message}</span>
                      ) : null}
                    </td>
                    <td className="py-2 pr-2 font-mono">
                      {s.page_count}/{s.chunk_count}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="mt-5 border-t border-[#eeeae3] pt-4">
          <h3 className="text-[14px] font-medium text-[#18212f]">Проверка RAG</h3>
          <div className="mt-2 flex gap-2">
            <input
              className={inputClass}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Например: таблица регистров Modbus"
            />
            <button
              type="button"
              className="shrink-0 rounded-lg bg-[#185fa5] px-3 py-2 text-[12px] font-medium text-white disabled:opacity-50"
              disabled={busy === "search"}
              onClick={() => void searchRAG()}
            >
              Найти
            </button>
          </div>
          <ul className="mt-3 space-y-2">
            {hits.map((h) => (
              <li key={h.chunk_id} className="rounded-lg border border-[#eeeae3] bg-[#faf8f5] p-3 text-[12px]">
                <div className="mb-1 flex flex-wrap gap-2 text-[11px] text-[#6f6a62]">
                  <span>
                    {h.product_name} · v{h.version} · стр. {h.page_number}
                  </span>
                  <span className="font-mono">score {h.score.toFixed(3)}</span>
                  <span>{h.from_keyword ? "keyword" : "vector"}</span>
                </div>
                <p className="whitespace-pre-wrap text-[#18212f]">{h.content_md.slice(0, 500)}</p>
              </li>
            ))}
          </ul>
        </div>
      </section>
    </div>
  );
}
