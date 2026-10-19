import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/shared/ui';

export const AdminPage = () => (
  <>
    <h1 className="mb-6 text-3xl font-semibold">Панель администратора</h1>
    <Card>
      <CardHeader>
        <CardTitle>Управление платформой</CardTitle>
        <CardDescription>Статистика, журнал действий и отчёты.</CardDescription>
      </CardHeader>
      <CardContent>
        <Button asChild>
          <a href="/dashboard/admin">Открыть инструменты</a>
        </Button>
      </CardContent>
    </Card>
  </>
);
