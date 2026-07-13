"use client";

import Link from "next/link";
import { DashboardShell } from "@/components/DashboardShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function AgentPage() {
  return (
    <DashboardShell activePath="/app/dashboard/agent" pageTitle="ИИ-агент">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">ИИ-агент поддержки</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Первая линия по документации производителей. Ответы только с цитатами из мануалов.
        </p>
      </div>

      <DashboardEmpty title="Чат агента подключается на следующем этапе">
        Пока опишите проблему через{" "}
        <Link href="/app/dashboard/tickets" className="text-[#185fa5] underline">
          создание тикета
        </Link>{" "}
        или воспользуйтесь{" "}
        <Link href="/app/kb" className="text-[#185fa5] underline">
          базой знаний
        </Link>
        . Агент учитывает профиль установки и версии продуктов на объекте.
      </DashboardEmpty>

      <div className="mt-6">
        <DashboardPanel title="Как будет работать">
          <ul className="list-disc space-y-2 pl-5 text-[13px] leading-5 text-[#5f6b7a]">
            <li>Один вопрос за раз — уточнение дельты конфигурации перед ответом.</li>
            <li>Каждое утверждение — со ссылкой на страницу документации.</li>
            <li>Если в мануале нет ответа — честный отказ и предложение эскалации.</li>
          </ul>
        </DashboardPanel>
      </div>
    </DashboardShell>
  );
}
