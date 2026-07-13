"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useMemo, useState } from "react";

type AuthMode = "login" | "register";

type AuthCardProps = {
  mode: AuthMode;
};

type TokenResponse = {
  data?: {
    access_token?: string;
    refresh_token?: string;
    role?: string;
  };
  error?: {
    message?: string;
  };
};

const passwordRules = [
  {
    label: "Не менее 12 символов",
    test: (value: string) => value.length >= 12,
  },
  {
    label: "Строчная буква (a-z)",
    test: (value: string) => /[a-zа-я]/.test(value),
  },
  {
    label: "Заглавная буква (A-Z)",
    test: (value: string) => /[A-ZА-Я]/.test(value),
  },
  {
    label: "Цифра (0-9)",
    test: (value: string) => /\d/.test(value),
  },
  {
    label: "Спецсимвол (!@#$...)",
    test: (value: string) => /[^a-zA-Zа-яА-Я0-9]/.test(value),
    optional: true,
  },
];

function routeForRole(role?: string) {
  if (role === "superadmin") {
    return "/app/admin";
  }
  if (role === "support_engineer" || role === "admin") {
    return "/app/vendor";
  }
  return "/app/dashboard";
}

function passwordScore(password: string) {
  return passwordRules.filter((rule) => rule.test(password)).length;
}

function RuleMark({ passed, optional }: { passed: boolean; optional?: boolean }) {
  return (
    <span
      className={
        passed
          ? "grid h-4 w-4 place-items-center rounded-full border border-[#6aa844] text-[10px] text-[#4f8f2f]"
          : "grid h-4 w-4 place-items-center rounded-full border border-[#cfd7df] text-[10px] text-[#9aa5b1]"
      }
      aria-hidden="true"
    >
      {passed ? "✓" : optional ? "" : "•"}
    </span>
  );
}

export function AuthCard({ mode }: AuthCardProps) {
  const router = useRouter();
  const isRegister = mode === "register";
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [orgName, setOrgName] = useState("");
  const [consent, setConsent] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [status, setStatus] = useState<"idle" | "submitting">("idle");
  const [error, setError] = useState("");

  const score = useMemo(() => passwordScore(password), [password]);
  const requiredPasswordOk = passwordRules
    .filter((rule) => !rule.optional)
    .every((rule) => rule.test(password));
  const passwordsMatch = !isRegister || (confirmPassword !== "" && password === confirmPassword);
  const canSubmit =
    status !== "submitting" &&
    email.trim() !== "" &&
    password !== "" &&
    (!isRegister || (requiredPasswordOk && passwordsMatch && consent));

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) {
      return;
    }
    setStatus("submitting");
    setError("");

    const payload = isRegister
      ? {
          email: email.trim().toLowerCase(),
          password,
          full_name: fullName.trim(),
          org_name: orgName.trim(),
        }
      : {
          email: email.trim().toLowerCase(),
          password,
        };

    try {
      const response = await fetch(`/api/v1/auth/${isRegister ? "register" : "login"}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const body = (await response.json()) as TokenResponse;

      if (!response.ok || !body.data?.access_token) {
        setError(body.error?.message || "Не удалось выполнить вход");
        return;
      }

      sessionStorage.setItem("asutport_access_token", body.data.access_token);
      if (body.data.refresh_token) {
        sessionStorage.setItem("asutport_refresh_token", body.data.refresh_token);
      }
      router.push(routeForRole(body.data.role));
    } catch {
      setError("Сервис авторизации временно недоступен");
    } finally {
      setStatus("idle");
    }
  }

  return (
    <main className="auth-page min-h-screen px-4 py-8 text-[#1f2933]">
      <div className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-[448px] items-center">
        <section className="w-full rounded-2xl border border-[#dfe5eb] bg-white px-8 py-9 shadow-[0_18px_60px_rgba(15,23,42,0.08)]">
          <Link href="/app/kb" className="inline-flex items-center gap-3" aria-label="ASUTPORT">
            <span className="grid h-9 w-9 place-items-center rounded-xl bg-[#0f2f2b] font-logo text-sm font-bold text-[#3fc8b7]">
              A
            </span>
            <span>
              <span className="block font-logo text-sm font-bold tracking-[0.12em] text-[#111827]">
                ASUTPORT
              </span>
              <span className="block text-xs text-[#6b7280]">техническая поддержка АСУ ТП</span>
            </span>
          </Link>

          <div className="mt-8">
            <h1 className="text-xl font-semibold text-[#111827]">
              {isRegister ? "Регистрация" : "Вход"}
            </h1>
            <p className="mt-2 text-sm leading-6 text-[#6b7280]">
              {isRegister
                ? "Создайте кабинет эксплуатации и начните вести профиль установки."
                : "Войдите в кабинет клиента, производителя или администратора."}
            </p>
          </div>

          <form className="mt-7 space-y-5" onSubmit={submit}>
            <label className="block">
              <span className="text-sm font-medium text-[#111827]">Email</span>
              <input
                type="email"
                autoComplete="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="mt-2 h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
              />
            </label>

            {isRegister ? (
              <div className="grid gap-5 sm:grid-cols-2">
                <label className="block">
                  <span className="text-sm font-medium text-[#111827]">ФИО</span>
                  <input
                    type="text"
                    autoComplete="name"
                    value={fullName}
                    onChange={(event) => setFullName(event.target.value)}
                    className="mt-2 h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                  />
                </label>
                <label className="block">
                  <span className="text-sm font-medium text-[#111827]">Организация</span>
                  <input
                    type="text"
                    autoComplete="organization"
                    value={orgName}
                    onChange={(event) => setOrgName(event.target.value)}
                    className="mt-2 h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                  />
                </label>
              </div>
            ) : null}

            <label className="block">
              <span className="text-sm font-medium text-[#111827]">Пароль</span>
              <span className="relative mt-2 block">
                <input
                  type={showPassword ? "text" : "password"}
                  autoComplete={isRegister ? "new-password" : "current-password"}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 pr-11 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((value) => !value)}
                  className="absolute inset-y-0 right-0 grid w-11 place-items-center text-[#77828f] hover:text-[#111827]"
                  aria-label={showPassword ? "Скрыть пароль" : "Показать пароль"}
                >
                  <EyeIcon />
                </button>
              </span>
            </label>

            {isRegister ? (
              <div>
                <div className="grid grid-cols-4 gap-1">
                  {[0, 1, 2, 3].map((index) => (
                    <span
                      key={index}
                      className={
                        index < Math.min(score, 4)
                          ? "h-1 rounded-full bg-[#7db7e8]"
                          : "h-1 rounded-full bg-[#d6d3ca]"
                      }
                    />
                  ))}
                </div>
                <p className="mt-2 text-xs font-medium text-[#2877a8]">
                  {score >= 4 ? "Хороший" : "Слабый пароль"}
                </p>
                <ul className="mt-3 space-y-2">
                  {passwordRules.map((rule) => {
                    const passed = rule.test(password);
                    return (
                      <li key={rule.label} className="flex items-center gap-2 text-xs text-[#667085]">
                        <RuleMark passed={passed} optional={rule.optional} />
                        {rule.label}
                      </li>
                    );
                  })}
                </ul>

                <label className="mt-5 block">
                  <span className="text-sm font-medium text-[#111827]">Подтвердите пароль</span>
                  <span className="relative mt-2 block">
                    <input
                      type={showConfirm ? "text" : "password"}
                      autoComplete="new-password"
                      value={confirmPassword}
                      onChange={(event) => setConfirmPassword(event.target.value)}
                      className="h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 pr-11 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                    />
                    <button
                      type="button"
                      onClick={() => setShowConfirm((value) => !value)}
                      className="absolute inset-y-0 right-0 grid w-11 place-items-center text-[#77828f] hover:text-[#111827]"
                      aria-label={showConfirm ? "Скрыть пароль" : "Показать пароль"}
                    >
                      <EyeIcon />
                    </button>
                  </span>
                  {confirmPassword !== "" && !passwordsMatch ? (
                    <span className="mt-2 block text-xs text-[#b42318]">Пароли не совпадают</span>
                  ) : null}
                </label>

                <label className="mt-5 flex gap-3 text-sm leading-6 text-[#4b5563]">
                  <input
                    type="checkbox"
                    checked={consent}
                    onChange={(event) => setConsent(event.target.checked)}
                    className="mt-1 h-4 w-4 rounded border-[#cfd7df] accent-[#0d9488]"
                  />
                  <span>
                    Согласие с{" "}
                    <Link href="/app/kb" className="text-[#246b82] hover:text-[#0d9488]">
                      политикой обработки персональных данных
                    </Link>
                  </span>
                </label>
              </div>
            ) : null}

            {error ? (
              <div className="rounded-lg border border-[#f0b8b8] bg-[#fff5f5] px-3 py-2 text-sm text-[#b42318]">
                {error}
              </div>
            ) : null}

            <button
              type="submit"
              disabled={!canSubmit}
              className="h-11 w-full rounded-lg bg-[#0d9488] text-sm font-semibold text-white transition hover:bg-[#0f766e] disabled:cursor-not-allowed disabled:bg-[#858585]"
            >
              {status === "submitting"
                ? "Проверяем..."
                : isRegister
                  ? "Зарегистрироваться"
                  : "Войти"}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-[#6b7280]">
            {isRegister ? "Уже есть аккаунт?" : "Нет аккаунта?"}{" "}
            <Link
              href={isRegister ? "/app/login" : "/app/register"}
              className="font-medium text-[#246b82] hover:text-[#0d9488]"
            >
              {isRegister ? "Войти" : "Зарегистрироваться"}
            </Link>
          </p>
        </section>
      </div>
    </main>
  );
}

function EyeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"
        stroke="currentColor"
        strokeWidth="1.8"
      />
    </svg>
  );
}
