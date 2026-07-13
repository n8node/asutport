"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

type VerifyResponse = {
  data?: {
    status?: string;
    message?: string;
    review_status?: string;
  };
  error?: {
    message?: string;
  };
};

export function ConfirmRegistrationContent() {
  const searchParams = useSearchParams();
  const regID = searchParams.get("id_reg") || "";
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!regID || !regID.startsWith("77")) {
      setStatus("error");
      setMessage("Некорректная ссылка подтверждения.");
      return;
    }

    void fetch(`/api/v1/auth/verify-registration?id_reg=${encodeURIComponent(regID)}`)
      .then(async (response) => {
        const body = (await response.json()) as VerifyResponse;
        if (!response.ok) {
          setStatus("error");
          setMessage(body.error?.message || "Ссылка недействительна или уже использована.");
          return;
        }
        setStatus("success");
        setMessage(
          body.data?.review_status === "pending_review"
            ? body.data.message || "Email подтверждён. Войдите и откройте раздел «Статус компании»."
            : body.data?.message || "Email подтверждён.",
        );
      })
      .catch(() => {
        setStatus("error");
        setMessage("Сервис подтверждения временно недоступен.");
      });
  }, [regID]);

  return (
    <main className="auth-page min-h-screen px-4 py-8 text-[#1f2933]">
      <div className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-[520px] items-center">
        <section className="w-full rounded-2xl border border-[#dfe5eb] bg-white px-8 py-9 shadow-[0_18px_60px_rgba(15,23,42,0.08)]">
          <h1 className="text-xl font-semibold text-[#111827]">Подтверждение регистрации</h1>
          {status === "loading" ? (
            <p className="mt-4 text-sm text-[#6b7280]">Проверяем ссылку...</p>
          ) : null}
          {status !== "loading" ? (
            <p
              className={`mt-4 text-sm leading-6 ${
                status === "success" ? "text-[#3b6d11]" : "text-[#b42318]"
              }`}
            >
              {message}
            </p>
          ) : null}
          <p className="mt-6 text-center text-sm">
            <Link href="/app/login" className="font-medium text-[#246b82] hover:text-[#0d9488]">
              Перейти ко входу
            </Link>
          </p>
        </section>
      </div>
    </main>
  );
}
