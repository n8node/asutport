"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useMemo, useState } from "react";
import { homeRouteFromLogin } from "@/lib/cabinet-routing";

type AuthMode = "login" | "register";

type AuthCardProps = {
  mode: AuthMode;
};

type TokenResponse = {
  data?: {
    access_token?: string;
    refresh_token?: string;
    role?: string;
    org_type?: string;
    review_status?: string;
    email_verification_required?: boolean;
    message?: string;
    email?: string;
  };
  error?: {
    code?: string;
    message?: string;
  };
};

type AccountType = "client_personal" | "client_org" | "manufacturer" | "vendor" | "integrator";

const accountTypes: Array<{
  value: AccountType;
  label: string;
  description: string;
}> = [
  {
    value: "client_personal",
    label: "Мне нужна поддержка",
    description: "Личный кабинет без привязки к организации.",
  },
  {
    value: "client_org",
    label: "Представляю эксплуатацию",
    description: "Клиентская организация или предприятие.",
  },
  {
    value: "manufacturer",
    label: "Производитель",
    description: "Заявка на подключение производителя.",
  },
  {
    value: "vendor",
    label: "Поставщик / вендор",
    description: "Дилер, поставщик, продавец лицензий или гарантий.",
  },
  {
    value: "integrator",
    label: "Интегратор",
    description: "Проектная или внедренческая организация.",
  },
];

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
  const [accountType, setAccountType] = useState<AccountType>("client_personal");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [orgName, setOrgName] = useState("");
  const [inn, setInn] = useState("");
  const [website, setWebsite] = useState("");
  const [contactPhone, setContactPhone] = useState("");
  const [reviewComment, setReviewComment] = useState("");
  const [consent, setConsent] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [status, setStatus] = useState<"idle" | "submitting">("idle");
  const [error, setError] = useState("");
  const [verifyNotice, setVerifyNotice] = useState("");

  const score = useMemo(() => passwordScore(password), [password]);
  const requiredPasswordOk = passwordRules
    .filter((rule) => !rule.optional)
    .every((rule) => rule.test(password));
  const isPersonal = accountType === "client_personal";
  const isB2BPending = ["manufacturer", "vendor", "integrator"].includes(accountType);
  const passwordsMatch = !isRegister || (confirmPassword !== "" && password === confirmPassword);
  const canSubmit =
    status !== "submitting" &&
    email.trim() !== "" &&
    password !== "" &&
    (!isRegister ||
      (requiredPasswordOk &&
        passwordsMatch &&
        consent &&
        (isPersonal || orgName.trim() !== "") &&
        (!isB2BPending || inn.trim() !== "")));

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) {
      return;
    }
    setStatus("submitting");
    setError("");
    setVerifyNotice("");

    const payload = isRegister
      ? {
          email: email.trim().toLowerCase(),
          password,
          full_name: fullName.trim(),
          account_type: accountType,
          org_name: orgName.trim(),
          inn: inn.trim(),
          website: website.trim(),
          contact_phone: contactPhone.trim(),
          review_comment: reviewComment.trim(),
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

      if (isRegister && response.ok && body.data?.email_verification_required) {
        setVerifyNotice(
          body.data.message ||
            `На ${body.data.email || email} отправлено письмо. Перейдите по ссылке с id_reg=77… для подтверждения.`,
        );
        return;
      }

      if (!response.ok || !body.data?.access_token) {
        if (body.error?.code === "EMAIL_NOT_VERIFIED") {
          setError(
            body.error.message ||
              "Подтвердите email по ссылке из письма (id_reg=77…) перед входом.",
          );
        } else {
          setError(body.error?.message || "Не удалось выполнить вход");
        }
        return;
      }

      sessionStorage.setItem("asutport_access_token", body.data.access_token);
      if (body.data.refresh_token) {
        sessionStorage.setItem("asutport_refresh_token", body.data.refresh_token);
      }
      router.push(
        homeRouteFromLogin(body.data.role, body.data.org_type, body.data.review_status),
      );
    } catch {
      setError("Сервис авторизации временно недоступен");
    } finally {
      setStatus("idle");
    }
  }

  return (
    <main className="auth-page min-h-screen px-4 py-8 text-[#1f2933]">
      <div className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-[560px] items-center">
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
                ? "Создайте личный кабинет или отправьте заявку на подключение организации."
                : "Войдите в кабинет клиента, производителя или администратора."}
            </p>
          </div>

          <form className="mt-7 space-y-5" onSubmit={submit}>
            {isRegister ? (
              <fieldset>
                <legend className="text-sm font-medium text-[#111827]">Кто вы?</legend>
                <div className="mt-2 grid gap-2">
                  {accountTypes.map((item) => (
                    <label
                      key={item.value}
                      className={
                        accountType === item.value
                          ? "cursor-pointer rounded-lg border border-[#0d9488] bg-[#ecfdf9] px-3 py-2"
                          : "cursor-pointer rounded-lg border border-[#d8dee6] bg-white px-3 py-2 hover:border-[#0d9488]/60"
                      }
                    >
                      <span className="flex items-start gap-3">
                        <input
                          type="radio"
                          name="account_type"
                          value={item.value}
                          checked={accountType === item.value}
                          onChange={() => setAccountType(item.value)}
                          className="mt-1 accent-[#0d9488]"
                        />
                        <span>
                          <span className="block text-sm font-medium text-[#111827]">
                            {item.label}
                          </span>
                          <span className="mt-0.5 block text-xs text-[#667085]">
                            {item.description}
                          </span>
                        </span>
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>
            ) : null}

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
                {!isPersonal ? (
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
                ) : null}
              </div>
            ) : null}

            {isRegister && !isPersonal ? (
              <div className="grid gap-5 sm:grid-cols-2">
                <label className="block">
                  <span className="text-sm font-medium text-[#111827]">
                    ИНН {isB2BPending ? "" : "(необязательно)"}
                  </span>
                  <input
                    type="text"
                    value={inn}
                    onChange={(event) => setInn(event.target.value)}
                    className="mt-2 h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                  />
                </label>
                <label className="block">
                  <span className="text-sm font-medium text-[#111827]">Телефон</span>
                  <input
                    type="tel"
                    autoComplete="tel"
                    value={contactPhone}
                    onChange={(event) => setContactPhone(event.target.value)}
                    className="mt-2 h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                  />
                </label>
                <label className="block sm:col-span-2">
                  <span className="text-sm font-medium text-[#111827]">Сайт компании</span>
                  <input
                    type="url"
                    value={website}
                    onChange={(event) => setWebsite(event.target.value)}
                    className="mt-2 h-11 w-full rounded-lg border border-[#cfd7df] bg-white px-3 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                    placeholder="https://"
                  />
                </label>
                {isB2BPending ? (
                  <label className="block sm:col-span-2">
                    <span className="text-sm font-medium text-[#111827]">Комментарий к заявке</span>
                    <textarea
                      value={reviewComment}
                      onChange={(event) => setReviewComment(event.target.value)}
                      className="mt-2 min-h-24 w-full rounded-lg border border-[#cfd7df] bg-white px-3 py-2 text-sm text-[#111827] outline-none transition focus:border-[#0d9488] focus:ring-4 focus:ring-[#0d9488]/10"
                      placeholder="Какие продукты, клиенты или компетенции хотите подключить?"
                    />
                    <span className="mt-2 block text-xs text-[#667085]">
                      После регистрации заявка попадёт на проверку платформы. До активации реальные эскалации недоступны.
                    </span>
                  </label>
                ) : null}
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

            {verifyNotice ? (
              <div className="rounded-lg border border-[#b9e6ce] bg-[#ecfdf3] px-3 py-2 text-sm text-[#3b6d11]">
                {verifyNotice}
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
