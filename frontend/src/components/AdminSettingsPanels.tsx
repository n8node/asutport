"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { ReactNode } from "react";

import { authFetch, defaultCorsHints } from "@/lib/auth-session";

type ApiError = { error?: { message?: string } };

type S3Settings = {
  enabled: boolean;
  endpoint: string;
  bucket: string;
  region: string;
  access_key_id: string;
  has_secret: boolean;
  use_path_style: boolean;
};

type S3CorsHints = {
  allowed_origins: string[];
  cors_xml: string;
};

type SMTPEncryption = "none" | "ssl" | "tls";

type SMTPSettings = {
  enabled: boolean;
  from_email: string;
  from_name: string;
  force_from_email: boolean;
  force_from_name: boolean;
  reply_to_from_email: boolean;
  host: string;
  port: number;
  encryption: SMTPEncryption;
  auto_tls: boolean;
  auth: boolean;
  username: string;
};

type SMTPView = {
  settings: SMTPSettings;
  password_set: boolean;
  password_hint: string;
  yandex_preset_host: string;
  yandex_preset_port: number;
};

const defaultS3: S3Settings = {
  enabled: false,
  endpoint: "",
  bucket: "",
  region: "ru-1",
  access_key_id: "",
  has_secret: false,
  use_path_style: true,
};

const defaultSMTP: SMTPView = {
  settings: {
    enabled: false,
    from_email: "",
    from_name: "ASUTPORT",
    force_from_email: true,
    force_from_name: true,
    reply_to_from_email: true,
    host: "",
    port: 465,
    encryption: "ssl",
    auto_tls: true,
    auth: true,
    username: "",
  },
  password_set: false,
  password_hint: "",
  yandex_preset_host: "smtp.yandex.ru",
  yandex_preset_port: 465,
};

const inputClass =
  "mt-1.5 w-full rounded-lg border border-[#d7d2ca] bg-white px-3 py-2 text-[13px] text-[#18212f] outline-none focus:border-[#2563eb] focus:ring-1 focus:ring-[#2563eb]";
const monoInputClass = `${inputClass} font-mono text-[12px]`;
const labelClass = "block text-[12px] font-medium text-[#4b5563]";
const sectionTitleClass = "text-[11px] font-medium uppercase tracking-[0.08em] text-[#8a857d]";

function isAuthError(message: string) {
  const lower = message.toLowerCase();
  return lower.includes("missing or invalid authentication") || lower.includes("unauthorized");
}

async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await authFetch(`/api/v1${path}`, options);
  const body = (await response.json()) as ({ data?: T } & ApiError);
  if (!response.ok) {
    throw new Error(body.error?.message || "request failed");
  }
  return body.data as T;
}

export function AdminSettingsPanels() {
  const [s3, setS3] = useState<S3Settings>(defaultS3);
  const [s3Secret, setS3Secret] = useState("");
  const [cors, setCors] = useState<S3CorsHints>(defaultCorsHints());
  const [authError, setAuthError] = useState(false);
  const [smtpView, setSMTPView] = useState<SMTPView>(defaultSMTP);
  const [smtpPassword, setSMTPPassword] = useState("");
  const [testTo, setTestTo] = useState("");
  const [loading, setLoading] = useState(true);
  const [savingS3, setSavingS3] = useState(false);
  const [testingS3, setTestingS3] = useState(false);
  const [savingSMTP, setSavingSMTP] = useState(false);
  const [testingSMTP, setTestingSMTP] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api<S3Settings>("/admin/settings/s3"),
      api<S3CorsHints>("/admin/settings/s3/cors-hints"),
      api<SMTPView>("/admin/settings/smtp"),
    ])
      .then(([s3Data, corsData, smtpData]) => {
        if (cancelled) return;
        setS3(s3Data);
        setCors(corsData);
        setSMTPView(smtpData);
      })
      .catch((err) => {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : "Не удалось загрузить настройки";
        if (isAuthError(message)) {
          setAuthError(true);
          setError("Сессия истекла или вы не вошли как суперадмин. Войдите снова — CORS и Test connection появятся после авторизации.");
        } else {
          setError(message);
        }
        setCors(defaultCorsHints());
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  function patchS3(patch: Partial<S3Settings>) {
    setS3((prev) => ({ ...prev, ...patch }));
    setMessage("");
  }

  function patchSMTP(patch: Partial<SMTPSettings>) {
    setSMTPView((prev) => ({ ...prev, settings: { ...prev.settings, ...patch } }));
    setMessage("");
  }

  async function saveS3() {
    setSavingS3(true);
    setError("");
    setMessage("");
    try {
      const body: Record<string, unknown> = {
        enabled: s3.enabled,
        endpoint: s3.endpoint,
        bucket: s3.bucket,
        region: s3.region,
        access_key_id: s3.access_key_id,
        use_path_style: s3.use_path_style,
      };
      if (s3Secret.trim()) {
        body.secret_access_key = s3Secret.trim();
      }
      const next = await api<S3Settings>("/admin/settings/s3", {
        method: "PATCH",
        body: JSON.stringify(body),
      });
      setS3(next);
      setS3Secret("");
      setMessage("S3-настройки сохранены");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Не удалось сохранить S3";
      setAuthError(isAuthError(message));
      setError(message);
    } finally {
      setSavingS3(false);
    }
  }

  async function testS3() {
    setTestingS3(true);
    setError("");
    setMessage("");
    try {
      await api<{ ok: boolean }>("/admin/settings/s3/test", { method: "POST" });
      setMessage("S3 подключение успешно проверено");
    } catch (err) {
      const message = err instanceof Error ? err.message : "S3 test failed";
      setAuthError(isAuthError(message));
      setError(message);
    } finally {
      setTestingS3(false);
    }
  }

  function applyYandexPreset() {
    patchSMTP({
      host: smtpView.yandex_preset_host,
      port: smtpView.yandex_preset_port,
      encryption: "ssl",
      auto_tls: true,
      auth: true,
    });
  }

  async function saveSMTP() {
    setSavingSMTP(true);
    setError("");
    setMessage("");
    try {
      const payload: Record<string, unknown> = { settings: smtpView.settings };
      if (smtpPassword.trim()) {
        payload.password = smtpPassword.trim();
      }
      const next = await api<SMTPView>("/admin/settings/smtp", {
        method: "PATCH",
        body: JSON.stringify(payload),
      });
      setSMTPView(next);
      setSMTPPassword("");
      setMessage("Email-настройки сохранены");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить SMTP");
    } finally {
      setSavingSMTP(false);
    }
  }

  async function testSMTP() {
    if (!testTo.trim()) return;
    setTestingSMTP(true);
    setError("");
    setMessage("");
    try {
      if (smtpPassword.trim()) {
        await saveSMTP();
      }
      const result = await api<{ ok: boolean; message: string }>("/admin/settings/smtp/test", {
        method: "POST",
        body: JSON.stringify({ to: testTo.trim() }),
      });
      if (result.ok) {
        setMessage(result.message);
      } else {
        setError(result.message);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить тестовое письмо");
    } finally {
      setTestingSMTP(false);
    }
  }

  if (loading) {
    return <p className="text-[13px] text-[#6f6a62]">Загружаю настройки S3 и SMTP...</p>;
  }

  return (
    <div id="settings" className="space-y-6">
      {error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">
          {error}
          {authError ? (
            <div className="mt-2">
              <Link href="/app/login" className="font-medium text-[#185fa5] underline">
                Войти снова
              </Link>
            </div>
          ) : null}
        </div>
      ) : null}
      {message ? (
        <div className="rounded-lg border border-green-200 bg-green-50 px-3 py-2 text-[13px] text-green-900">{message}</div>
      ) : null}

      <div id="s3" className="space-y-5">
        <div>
          <h2 className="text-[18px] font-semibold text-[#18212f]">S3-compatible object storage</h2>
          <p className="mt-1 text-[13px] text-[#6f6a62]">
            Объектное хранилище для документации, страниц, вложений тикетов и слепков конфигурации.
          </p>
        </div>

        <SettingsBlock title="S3-compatible storage">
          <p className="mb-4 text-[12px] text-[#6f6a62]">
            Работает с провайдерами, которые открывают S3 API. Для Beget и MinIO обычно нужен path-style.
            Кнопка <strong>Test connection</strong> проверяет доступ с сервера — CORS для неё не нужен.
          </p>
          <Check checked={s3.enabled} onChange={(v) => patchS3({ enabled: v })}>Enable storage for ASUTPORT uploads</Check>
          <Field label="Endpoint (URL)" className="mt-4">
            <input className={monoInputClass} value={s3.endpoint} onChange={(e) => patchS3({ endpoint: e.target.value })} placeholder="https://s3.ru1.storage.beget.cloud" />
          </Field>
          <Field label="Bucket" className="mt-4">
            <input className={monoInputClass} value={s3.bucket} onChange={(e) => patchS3({ bucket: e.target.value })} />
          </Field>
          <Field label="Region" className="mt-4">
            <input className={monoInputClass} value={s3.region} onChange={(e) => patchS3({ region: e.target.value })} />
          </Field>
          <Field label="Access Key ID" className="mt-4">
            <input className={monoInputClass} value={s3.access_key_id} onChange={(e) => patchS3({ access_key_id: e.target.value })} />
          </Field>
          <Field label={`Secret Access Key ${s3.has_secret ? "(leave empty to keep current)" : ""}`} className="mt-4">
            <input className={monoInputClass} type="password" autoComplete="new-password" value={s3Secret} onChange={(e) => setS3Secret(e.target.value)} />
          </Field>
          <div className="mt-4">
            <Check checked={s3.use_path_style} onChange={(v) => patchS3({ use_path_style: v })}>Use path-style addressing (typical for MinIO and many S3-compatible hosts)</Check>
          </div>
          <div className="mt-6 flex flex-wrap gap-2">
            <button type="button" disabled={savingS3} onClick={() => void saveS3()} className="rounded-lg bg-[#18212f] px-4 py-2 text-[13px] font-medium text-white disabled:opacity-50">
              {savingS3 ? "Saving..." : "Save"}
            </button>
            <button type="button" disabled={testingS3} onClick={() => void testS3()} className="rounded-lg border border-[#d7d2ca] px-4 py-2 text-[13px] disabled:opacity-50">
              {testingS3 ? "Testing..." : "Test connection"}
            </button>
          </div>
        </SettingsBlock>

        <SettingsBlock title="CORS on the bucket">
          <p className="text-[12px] leading-5 text-[#6f6a62]">
            CORS настраивается в панели Beget S3 для прямых браузерных загрузок. Origins приложения:{" "}
            {cors.allowed_origins.join(", ")}
          </p>
          <textarea
            readOnly
            value={cors.cors_xml}
            spellCheck={false}
            className="mt-3 min-h-[220px] w-full rounded-lg border border-[#d7d2ca] bg-[#f7f6f2] px-3 py-2 font-mono text-[11px] text-[#18212f] outline-none"
          />
        </SettingsBlock>
      </div>

      <div id="smtp" className="space-y-5 border-t border-[#dedbd3] pt-6">
        <div>
          <h2 className="text-[18px] font-semibold text-[#18212f]">Email / SMTP</h2>
          <p className="mt-1 text-[13px] text-[#6f6a62]">
            Настройка внешнего SMTP-сервера для писем подтверждения и уведомлений.
          </p>
        </div>

        <SettingsBlock>
          <label className="flex items-center gap-2 text-[13px] text-[#18212f]">
            <input type="checkbox" checked={smtpView.settings.enabled} onChange={(e) => patchSMTP({ enabled: e.target.checked })} />
            Включить отправку email в системе
          </label>
        </SettingsBlock>

        <SettingsBlock title="Быстрый пресет">
          <button type="button" onClick={applyYandexPreset} className="rounded-lg border border-[#d7d2ca] px-4 py-2 text-[13px] hover:bg-[#ebe9e4]">
            Применить Yandex SMTP
          </button>
        </SettingsBlock>

        <SettingsBlock title="Настройки отправителя">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Эл. адрес отправителя">
              <input className={inputClass} value={smtpView.settings.from_email} onChange={(e) => patchSMTP({ from_email: e.target.value })} />
            </Field>
            <Field label="Имя отправителя">
              <input className={inputClass} value={smtpView.settings.from_name} onChange={(e) => patchSMTP({ from_name: e.target.value })} />
            </Field>
          </div>
          <div className="mt-4 space-y-2 text-[13px]">
            <Check checked={smtpView.settings.force_from_email} onChange={(v) => patchSMTP({ force_from_email: v })}>Всегда использовать этот адрес отправителя</Check>
            <Check checked={smtpView.settings.force_from_name} onChange={(v) => patchSMTP({ force_from_name: v })}>Всегда использовать это имя отправителя</Check>
            <Check checked={smtpView.settings.reply_to_from_email} onChange={(v) => patchSMTP({ reply_to_from_email: v })}>Использовать этот адрес как Reply-to</Check>
          </div>
        </SettingsBlock>

        <SettingsBlock title="SMTP подключение">
          <div className="grid gap-4 sm:grid-cols-[1fr_120px]">
            <Field label="SMTP Host">
              <input className={inputClass} value={smtpView.settings.host} onChange={(e) => patchSMTP({ host: e.target.value })} />
            </Field>
            <Field label="SMTP Port">
              <input className={inputClass} type="number" value={smtpView.settings.port} onChange={(e) => patchSMTP({ port: Number(e.target.value) || 465 })} />
            </Field>
          </div>

          <div className="mt-4">
            <span className={labelClass}>Шифрование</span>
            <div className="mt-2 flex gap-4 text-[13px]">
              {(["none", "ssl", "tls"] as SMTPEncryption[]).map((item) => (
                <label key={item} className="flex items-center gap-1.5 uppercase">
                  <input type="radio" checked={smtpView.settings.encryption === item} onChange={() => patchSMTP({ encryption: item })} />
                  {item}
                </label>
              ))}
            </div>
          </div>

          <div className="mt-4 space-y-2 text-[13px]">
            <Check checked={smtpView.settings.auto_tls} onChange={(v) => patchSMTP({ auto_tls: v })}>Use Auto TLS</Check>
            <Check checked={smtpView.settings.auth} onChange={(v) => patchSMTP({ auth: v })}>Authentication</Check>
          </div>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <Field label="SMTP Username">
              <input className={inputClass} value={smtpView.settings.username} onChange={(e) => patchSMTP({ username: e.target.value })} />
            </Field>
            <Field label="SMTP Password">
              <input
                className={inputClass}
                type="password"
                autoComplete="new-password"
                value={smtpPassword}
                placeholder={smtpView.password_set ? `Новый пароль (текущий: ${smtpView.password_hint || "••••"})` : "SMTP password"}
                onChange={(e) => setSMTPPassword(e.target.value)}
              />
            </Field>
          </div>
        </SettingsBlock>

        <SettingsBlock title="Тестовая отправка">
          <div className="flex flex-wrap gap-2">
            <input className={`${inputClass} mt-0 min-w-[240px] flex-1`} type="email" value={testTo} onChange={(e) => setTestTo(e.target.value)} placeholder="email для тестовой отправки" />
            <button type="button" disabled={testingSMTP || !testTo.trim()} onClick={() => void testSMTP()} className="rounded-lg border border-[#d7d2ca] px-4 py-2 text-[13px] disabled:opacity-50">
              {testingSMTP ? "Отправка..." : "Отправить"}
            </button>
          </div>
        </SettingsBlock>

        <button type="button" disabled={savingSMTP} onClick={() => void saveSMTP()} className="rounded-lg bg-[#185fa5] px-4 py-2 text-[13px] font-medium text-white disabled:opacity-50">
          {savingSMTP ? "Сохраняю..." : "Сохранить email-настройки"}
        </button>
      </div>
    </div>
  );
}

function SettingsBlock({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <section className="rounded-[12px] border border-[#dedbd3] bg-white p-5">
      {title ? <h3 className={`${sectionTitleClass} mb-4`}>{title}</h3> : null}
      {children}
    </section>
  );
}

function Field({ label, children, className = "" }: { label: string; children: ReactNode; className?: string }) {
  return (
    <label className={`${labelClass} ${className}`}>
      {label}
      {children}
    </label>
  );
}

function Check({ checked, onChange, children }: { checked: boolean; onChange: (value: boolean) => void; children: ReactNode }) {
  return (
    <label className="flex items-center gap-2 text-[13px] text-[#18212f]">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {children}
    </label>
  );
}
