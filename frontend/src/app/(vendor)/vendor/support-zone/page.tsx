import { OrgTypeGate } from "@/components/cabinet/OrgTypeGate";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { VendorShell } from "@/components/VendorShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function VendorSupportZonePage() {
  return (
    <VendorShell activePath="/app/vendor/support-zone" pageTitle="Зона поддержки">
      <VendorPageHeader
        title="Зона поддержки"
        subtitle="Границы ответственности производителя: версии, продукт vs прикладное, часы реакции. На MVP — импорт YAML через админку."
      />
      <OrgTypeGate allowed={["manufacturer"]} title="Раздел для производителей">
        <DashboardEmpty title="Политика зоны ещё не импортирована">
          Супервайзер платформы импортирует YAML с матрицей версий и правилами эскалации. Агент применяет политику до создания тикета; публичная версия — на странице производителя в базе знаний.
        </DashboardEmpty>
        <div className="mt-6">
          <DashboardPanel title="Пример содержания YAML">
            <pre className="overflow-x-auto rounded-lg bg-[#faf9f7] p-3 text-[11px] leading-5 text-[#5f6b7a]">{`products:
  - slug: plc-x
    supported_versions: ["2.1", "2.2", "2.3"]
    eol_versions: ["1.x"]
hours:
  timezone: Europe/Moscow
  business: "09:00-18:00"
boundaries:
  product_vs_application: "прикладной код — зона интегратора"`}</pre>
          </DashboardPanel>
        </div>
      </OrgTypeGate>
    </VendorShell>
  );
}
