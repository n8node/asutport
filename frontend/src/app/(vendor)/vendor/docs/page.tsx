import { OrgTypeGate } from "@/components/cabinet/OrgTypeGate";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { VendorShell } from "@/components/VendorShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function VendorDocsPage() {
  return (
    <VendorShell activePath="/app/vendor/docs" pageTitle="Документация">
      <VendorPageHeader
        title="Документация продукта"
        subtitle="PDF-мануалы для ИИ-агента: загрузка, парсинг страниц, индексация по версиям."
      />
      <OrgTypeGate allowed={["manufacturer"]} title="Раздел для производителей">
        <DashboardEmpty title="Пайплайн документации подключается на онбординге">
          После активации организации суперадмин загружает комплект PDF в S3; платформа рендерит страницы, извлекает таблицы и формулы, строит эмбеддинги.
          Статус парсинга и golden set — в кабинете администратора.
        </DashboardEmpty>
        <div className="mt-6 grid gap-4 lg:grid-cols-2">
          <DashboardPanel title="Что загружать">
            <ul className="list-disc space-y-2 pl-5 text-[13px] leading-5 text-[#5f6b7a]">
              <li>Руководства пользователя и администрирования по версиям продуктов.</li>
              <li>Один файл — одна версия; повторная загрузка того же hash не жжёт токены.</li>
              <li>Лицензионные ограничения: бакет приватный, доступ только через платформу.</li>
            </ul>
          </DashboardPanel>
          <DashboardPanel title="Следующий шаг">
            <p className="text-[13px] leading-5 text-[#6f6a62]">
              На пилоте загрузку выполняет супервайзер ASUTPORT. UI загрузки для производителя — после стабилизации пайплайна vision-парсинга.
            </p>
          </DashboardPanel>
        </div>
      </OrgTypeGate>
    </VendorShell>
  );
}
