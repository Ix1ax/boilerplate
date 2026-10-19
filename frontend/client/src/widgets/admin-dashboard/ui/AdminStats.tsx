import { Link } from 'react-router';

import { useAdminDashboard } from '@/entities/admin';

import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Skeleton,
} from '@/shared/ui';

export const AdminStats = () => {
  const { data, isPending, isError, refetch } = useAdminDashboard();
  if (isPending) return <Skeleton className="h-64 w-full" />;
  if (isError)
    return (
      <div role="alert">
        Не удалось загрузить статистику.{' '}
        <Button onClick={() => void refetch()}>Повторить</Button>
      </div>
    );
  return (
    <div className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          ['Ждут допуска', data.organizations.pending],
          ['Предстоящие встречи', data.events.upcoming],
          ['Подтверждённые участники', data.participations.confirmed],
          ['Активные запросы компании', data.companyRequests.active],
          ['Снятые встречи', data.events.unpublished],
          ['Скрытые сообщения', data.comments.hidden],
          ['Впервые на встрече', data.participations.firstTime],
          ['Состав собран', data.quorum.reached],
        ].map(([label, value]) => (
          <Card key={label}>
            <CardHeader>
              <CardTitle className="text-sm">{label}</CardTitle>
            </CardHeader>
            <CardContent className="text-3xl font-bold">{value}</CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Новые встречи и записи за 14 дней</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr>
                <th>Дата</th>
                <th>Встречи</th>
                <th>Записи</th>
              </tr>
            </thead>
            <tbody>
              {data.series.map((day) => (
                <tr key={day.date}>
                  <td>{day.date}</td>
                  <td>{day.events}</td>
                  <td>{day.participations}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Популярные направления</CardTitle>
          </CardHeader>
          <CardContent>
            {data.topSports.map((sport) => (
              <p key={sport.sport}>
                {sport.sport}: {sport.events}
              </p>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Давние заявки</CardTitle>
          </CardHeader>
          <CardContent>
            {data.pendingOrganizations.map((org) => (
              <p key={org.id}>
                <Link
                  className="underline"
                  to={`/dashboard/admin?tab=organizations&org_id=${org.id}`}
                >
                  {org.name}
                </Link>{' '}
                · {org.ownerName} ·{' '}
                {new Date(org.createdAt).toLocaleDateString('ru-RU')}
              </p>
            ))}
          </CardContent>
        </Card>
      </div>
      <p className="text-sm text-muted-foreground">
        Статистика обновляется из событий платформы. Последнее изменение:{' '}
        {data.updatedAt
          ? new Date(data.updatedAt).toLocaleString('ru-RU')
          : 'ещё не получено'}
        .
      </p>
    </div>
  );
};
