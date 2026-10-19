import { useState } from 'react';
import { Link } from 'react-router';

import { ClubDemandPanel } from '@/widgets/club-demand';

import { ClubForm } from '@/features/organization-profile';

import { useMyEvents } from '@/entities/event';
import { useMyOrganization } from '@/entities/organization';

import { DYNAMIC_ROUTES, STATIC_ROUTES } from '@/shared/constants';
import {
  Button,
  HeroBand,
  Input,
  Notice,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@/shared/ui';

import { ClubStatus } from './ClubStatus';

export const ClubPage = () => {
  const organization = useMyOrganization();
  const events = useMyEvents();
  const [tab, setTab] = useState('events');
  const [search, setSearch] = useState('');

  return (
    <>
      <HeroBand
        title="Кабинет клуба"
        eyebrow="организаторам"
        description="Представьте свой клуб. После проверки можно создавать встречи от имени организации."
      />
      <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-5 px-4 pt-7 pb-18 sm:px-8">
        {organization.isPending && (
          <p role="status">Загружаем профиль клуба…</p>
        )}
        {organization.isError && (
          <Notice tone="danger" role="alert">
            Не удалось загрузить профиль.
            <Button onClick={() => void organization.refetch()}>
              Повторить
            </Button>
          </Notice>
        )}
        {!organization.isPending && !organization.isError && (
          <>
            {organization.data && (
              <ClubStatus organization={organization.data} />
            )}
            {organization.data?.status === 'approved' ? (
              <>
                <Tabs value={tab} onValueChange={setTab}>
                  <TabsList
                    aria-label="Кабинет клуба"
                    className="flex flex-wrap"
                  >
                    <TabsTrigger value="events">События</TabsTrigger>
                    <TabsTrigger value="application">
                      Заявка и профиль
                    </TabsTrigger>
                    <TabsTrigger value="demand">Спрос рядом</TabsTrigger>
                  </TabsList>
                </Tabs>
                <Button
                  variant="outline"
                  className="self-start"
                  render={
                    <Link
                      to={DYNAMIC_ROUTES.ORGANIZATION(organization.data.id)}
                    />
                  }
                >
                  Страница клуба
                </Button>
                {tab === 'application' && (
                  <ClubForm organization={organization.data} />
                )}
                {tab === 'demand' && (
                  <ClubDemandPanel sports={organization.data.sports} />
                )}
                {tab === 'events' && (
                  <section className="grid gap-4">
                    <div className="flex flex-wrap items-center justify-between gap-4">
                      <h2 className="title-display text-2xl">Встречи клуба</h2>
                      <Button render={<Link to={STATIC_ROUTES.EVENT_NEW} />}>
                        Встреча от клуба
                      </Button>
                    </div>
                    <label className="grid gap-2">
                      Поиск по встречам клуба
                      <Input
                        type="search"
                        value={search}
                        onChange={(event) => setSearch(event.target.value)}
                        placeholder="Название встречи"
                      />
                    </label>
                    {events.isPending && (
                      <p role="status">Загружаем встречи…</p>
                    )}
                    {events.isError && (
                      <Notice role="alert" tone="danger">
                        Не удалось загрузить встречи.{' '}
                        <Button onClick={() => void events.refetch()}>
                          Повторить
                        </Button>
                      </Notice>
                    )}
                    {events.data?.items
                      .filter(
                        (event) =>
                          event.organizer.type === 'organization' &&
                          event.organizer.id === organization.data?.id &&
                          event.title
                            .toLocaleLowerCase()
                            .includes(search.toLocaleLowerCase()),
                      )
                      .map((event) => (
                        <article
                          key={event.id}
                          className="flex flex-wrap items-center justify-between gap-4 rounded-3xl bg-card p-6"
                        >
                          <div>
                            <h3 className="text-xl font-bold">{event.title}</h3>
                            <p>
                              {event.status === 'published'
                                ? 'Опубликована'
                                : 'Снята'}{' '}
                              · в составе: {event.confirmedCount} · заявок:{' '}
                              {event.pendingCount}
                            </p>
                          </div>
                          <Button
                            render={
                              <Link
                                to={DYNAMIC_ROUTES.EVENT_MANAGE(event.id)}
                              />
                            }
                          >
                            Управлять
                          </Button>
                        </article>
                      ))}
                    {events.data &&
                      !events.data.items.some(
                        (event) =>
                          event.organizer.type === 'organization' &&
                          event.organizer.id === organization.data?.id &&
                          event.title
                            .toLocaleLowerCase()
                            .includes(search.toLocaleLowerCase()),
                      ) && (
                        <p>
                          Ничего не нашлось. Измените запрос или создайте
                          встречу.
                        </p>
                      )}
                  </section>
                )}
              </>
            ) : (
              <>
                <ol
                  className="grid gap-3 rounded-3xl bg-card p-6 sm:grid-cols-3"
                  aria-label="Этапы допуска"
                >
                  <li>
                    <b>1. Заявка</b>
                    <p>Расскажите о клубе и направлениях.</p>
                  </li>
                  <li>
                    <b>2. Проверка</b>
                    <p>
                      Администратор рассмотрит заявку. До решения публичной
                      страницы нет.
                    </p>
                  </li>
                  <li>
                    <b>3. Публикация</b>
                    <p>После допуска встречи публикуются сразу.</p>
                  </li>
                </ol>
                <ClubForm organization={organization.data ?? null} />
              </>
            )}
          </>
        )}
      </div>
    </>
  );
};
