import { OrgTypeGate } from "@/components/cabinet/OrgTypeGate";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { VendorShell } from "@/components/VendorShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function VendorKbDraftsPage() {
  return (
    <VendorShell activePath="/app/vendor/kb-drafts" pageTitle="Черновики KB">
      <VendorPageHeader
        title="Черновики базы знаний"
        subtitle="Статьи, сгенерированные из закрытых тикетов: проверка, правки, публикация в /kb."
      />
      <OrgTypeGate allowed={["manufacturer"]} title="Раздел для производителей">
        <DashboardEmpty title="Черновиков пока нет">
          Когда тикет закрыт с решением, платформа предложит черновик статьи (анонимизированный). Инженер производителя утверждает текст — он попадает в ретривал агента и в публичную базу знаний после модерации.
        </DashboardEmpty>
        <div className="mt-6">
          <DashboardPanel title="Workflow">
            <ol className="list-decimal space-y-2 pl-5 text-[13px] leading-5 text-[#5f6b7a]">
              <li>Тикет закрыт → черновик в очереди утверждения.</li>
              <li>Инженер правит формулировки и версии продуктов.</li>
              <li>После утверждения — модерация платформы и публикация в /kb.</li>
            </ol>
          </DashboardPanel>
        </div>
      </OrgTypeGate>
    </VendorShell>
  );
}
