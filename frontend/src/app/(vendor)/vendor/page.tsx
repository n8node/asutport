"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { VendorShell } from "@/components/VendorShell";
import { fetchAccountProfile, vendorOrgLabel } from "@/lib/cabinet-routing";
import { fetchVendorDashboard, fetchVendorTickets } from "@/lib/vendor-dashboard";
import { DashboardPanel } from "@/components/dashboard/Ui";

export default function VendorPage() {
  const router = useRouter();
  const [orgType, setOrgType] = useState("");
  const [openCount, setOpenCount] = useState<number | null>(null);
  const [waitingOnVendor, setWaitingOnVendor] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void fetchAccountProfile().then((profile) => {
      if (profile?.org?.review_status === "pending_review") {
        router.replace("/app/vendor/onboarding");
        return;
      }
      setOrgType(profile?.org?.type ?? "");
    });
  }, [router]);

  useEffect(() => {
    void Promise.all([fetchVendorDashboard(), fetchVendorTickets()])
      .then(([summary, tickets]) => {
        setOpenCount(summary?.open_escalations_count ?? 0);
        setWaitingOnVendor(tickets.filter((t) => t.status === "waiting_vendor").length);
      })
      .catch(() => {
        setOpenCount(0);
        setWaitingOnVendor(0);
      })
      .finally(() => setLoading(false));
  }, []);

  const isManufacturer = orgType === "manufacturer";
  const isIntegrator = orgType === "integrator";

  return (
    <VendorShell activePath="/app/vendor" pageTitle="Сводка">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Сводка</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          {isManufacturer
            ? "Кабинет производителя: эскалации по дефектам, документация для агента, зона поддержки."
            : isIntegrator
              ? "Кабинет интегратора: эскалации по прикладному коду ваших проектов на объектах клиентов."
              : "Кабинет поставщика: эскалации по гарантии и коммерции по вашим отгрузкам."}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
          <div className="text-[10px] font-medium uppercase tracking-[0.08em] text-[#9a948c]">Открытые эскалации</div>
          <div className="mt-2 font-mono text-3xl font-medium text-[#18212f]">{loading ? "…" : openCount ?? 0}</div>
        </div>
        <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
          <div className="text-[10px] font-medium uppercase tracking-[0.08em] text-[#9a948c]">Мяч у вас</div>
          <div className="mt-2 font-mono text-3xl font-medium text-[#ba7517]">{loading ? "…" : waitingOnVendor}</div>
        </div>
      </div>

      <div className="mt-6 flex flex-wrap gap-2">
        <ActionLink href="/app/vendor/tickets" primary>
          Открыть очередь
        </ActionLink>
        {isManufacturer ? (
          <>
            <ActionLink href="/app/vendor/docs">Документация</ActionLink>
            <ActionLink href="/app/vendor/support-zone">Зона поддержки</ActionLink>
          </>
        ) : null}
        <ActionLink href="/app/kb">База знаний</ActionLink>
      </div>

      <div className="mt-8 grid gap-6 lg:grid-cols-2">
        <DashboardPanel title="С чего начать">
          <ol className="space-y-3 text-[13px] leading-5 text-[#5f6b7a]">
            <li>1. Дождитесь активации организации платформой (если ещё на проверке).</li>
            <li>
              2. Отвечайте в{" "}
              <Link href="/app/vendor/tickets" className="text-[#185fa5] underline">
                очереди эскалаций
              </Link>{" "}
              — SLA-таймер идёт с момента эскалации.
            </li>
            {isManufacturer ? (
              <li>3. Согласуйте загрузку PDF-документации и YAML зоны поддержки с менеджером ASUTPORT.</li>
            ) : (
              <li>3. Убедитесь, что клиенты указали ваше имя как {isIntegrator ? "интегратора" : "поставщика"} в профиле установки.</li>
            )}
          </ol>
        </DashboardPanel>

        <DashboardPanel title="Ваша роль на платформе">
          <p className="text-[13px] leading-5 text-[#6f6a62]">
            Тип организации: <strong>{vendorOrgLabel(orgType) || "—"}</strong>.
            {isManufacturer
              ? " Вы получаете тикеты типа «дефект» и «стыковой», когда клиент указал ваш продукт на установке."
              : isIntegrator
                ? " Вы получаете тикеты «прикладной код», когда клиент указал вас как интегратора проекта."
                : " Вы получаете тикеты «гарантия», когда клиент указал вас как поставщика в серийниках."}
          </p>
        </DashboardPanel>
      </div>
    </VendorShell>
  );
}

function ActionLink({ children, href, primary = false }: { children: React.ReactNode; href: string; primary?: boolean }) {
  return (
    <Link
      href={href}
      className={
        primary
          ? "rounded-full bg-[#18212f] px-4 py-2 text-[12px] font-medium text-white hover:opacity-90"
          : "rounded-full border border-[#d7d2ca] px-4 py-2 text-[12px] font-medium text-[#18212f] hover:bg-[#ebe9e4]"
      }
    >
      {children}
    </Link>
  );
}
