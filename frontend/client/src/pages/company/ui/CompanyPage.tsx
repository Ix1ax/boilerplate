import { useState } from 'react';

import {
  CompanyRequestForm,
  useCloseCompanyRequest,
} from '@/features/company-request';

import { type CompanyRequest,useCompanyRequests } from '@/entities/company';

import { Button, HeroBand, Notice } from '@/shared/ui';

import { CompanyRequestCard } from './CompanyRequestCard';

export const CompanyPage = () => {
  const [selected, setSelected] = useState<string | null>(null);
  const [editing, setEditing] = useState<CompanyRequest | undefined>();
  const list = useCompanyRequests();
  const close = useCloseCompanyRequest();

  return (
    <>
      <HeroBand
        title="Собрать компанию"
        eyebrow="спорт вместе"
        description="Расскажите, чем и когда хотите заниматься. Подберём встречи и сообщим о новых."
      />
      <div className="mx-auto grid w-full max-w-[1280px] gap-7 px-4 pt-7 pb-18 sm:px-8 lg:grid-cols-[360px_1fr]">
        <CompanyRequestForm
          editing={editing}
          onCancel={editing ? () => setEditing(undefined) : undefined}
          onCreated={(request) => {
            setSelected(request.id);
            setEditing(undefined);
          }}
        />
        <section className="flex min-w-0 flex-col gap-4">
          <h2 className="title-display text-[26px]">Мои запросы</h2>
          {list.isPending && <p role="status">Загружаем запросы…</p>}
          {list.isError && (
            <Notice tone="danger" role="alert">
              Не удалось загрузить запросы.{' '}
              <Button onClick={() => void list.refetch()}>Повторить</Button>
            </Notice>
          )}
          {list.data?.items.length === 0 && (
            <p>Пока нет запросов. Создайте первый, чтобы найти компанию.</p>
          )}
          {close.isError && (
            <Notice tone="danger" role="alert">
              {close.error.message}
            </Notice>
          )}
          {list.data?.items.map((item) => (
            <CompanyRequestCard
              key={item.id}
              item={item}
              selected={selected === item.id}
              closing={close.isPending}
              onToggle={() =>
                setSelected(selected === item.id ? null : item.id)
              }
              onEdit={() => {
                setEditing(item);
                window.scrollTo({ top: 0, behavior: 'smooth' });
              }}
              onClose={() =>
                close.mutate(item.id, { onSuccess: () => setSelected(null) })
              }
            />
          ))}
        </section>
      </div>
    </>
  );
};
