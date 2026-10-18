import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { useTaskStats } from '@/entities/task';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Skeleton,
} from '@/shared/ui';

const formatDay = (iso: string) =>
  new Date(iso).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
  });

const StatCard = ({ label, value }: { label: string; value: number }) => (
  <Card size="sm">
    <CardHeader>
      <CardTitle className="text-sm font-normal text-muted-foreground">
        {label}
      </CardTitle>
    </CardHeader>
    <CardContent className="text-2xl font-semibold">{value}</CardContent>
  </Card>
);

export const AdminStats = () => {
  const { data, isPending, isError, refetch } = useTaskStats();

  if (isPending) return <Skeleton className="h-64 w-full" />;

  if (isError) {
    return (
      <div role="alert">
        Не удалось загрузить статистику.{' '}
        <button className="underline" onClick={() => void refetch()}>
          Повторить
        </button>
      </div>
    );
  }

  const chartData = data.by_day.map((day) => ({
    date: day.date,
    label: formatDay(day.date),
    count: day.count,
  }));

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 @md:grid-cols-4">
        <StatCard label="Всего задач" value={data.total} />
        <StatCard label="Готово" value={data.completed} />
        <StatCard label="Пользователей" value={data.owners} />
        <StatCard
          label="Нужна проверка ИИ"
          value={data.moderation.needs_review}
        />
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Создано задач за 14 дней</CardTitle>
        </CardHeader>
        <CardContent>
          <div
            className="h-64 w-full"
            role="img"
            aria-label="Задачи, созданные за последние 14 дней"
          >
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={chartData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid vertical={false} stroke="var(--border)" />
                <XAxis
                  dataKey="label"
                  tickLine={false}
                  axisLine={false}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 12 }}
                />
                <YAxis
                  allowDecimals={false}
                  tickLine={false}
                  axisLine={false}
                  width={28}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 12 }}
                />
                <Tooltip
                  cursor={{ fill: 'var(--muted)' }}
                  contentStyle={{
                    background: 'var(--popover)',
                    color: 'var(--popover-foreground)',
                    border: '1px solid var(--border)',
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  labelFormatter={(label) => label}
                  formatter={(value) => [value ?? 0, 'Задач']}
                />
                <Bar
                  dataKey="count"
                  fill="var(--primary)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
          <table className="sr-only">
            <caption>Задачи, созданные за последние 14 дней</caption>
            <thead>
              <tr>
                <th>Дата</th>
                <th>Количество</th>
              </tr>
            </thead>
            <tbody>
              {chartData.map((day) => (
                <tr key={day.date}>
                  <td>{day.date}</td>
                  <td>{day.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>
    </div>
  );
};
