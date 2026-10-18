import { DownloadIcon } from 'lucide-react';

import { AuditLogTable } from '@/widgets/admin-audit';
import { AdminStats } from '@/widgets/admin-dashboard';

import { env } from '@/shared/config';
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/shared/ui';

export const DashboardAdminPage = () => {
  return (
    <Tabs defaultValue="overview">
      <TabsList>
        <TabsTrigger value="overview">Обзор</TabsTrigger>
        <TabsTrigger value="audit">Журнал</TabsTrigger>
        <TabsTrigger value="reports">Отчёты</TabsTrigger>
      </TabsList>
      <TabsContent value="overview" className="mt-4">
        <AdminStats />
      </TabsContent>
      <TabsContent value="audit" className="mt-4">
        <Card>
          <CardHeader>
            <CardTitle>Журнал действий</CardTitle>
          </CardHeader>
          <CardContent>
            <AuditLogTable />
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="reports" className="mt-4">
        <Card>
          <CardHeader>
            <CardTitle>Отчёты</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Выгрузка открывается в Excel (UTF-8, разделитель «;»).
            </p>
            <Button
              render={
                <a
                  href={`${env.VITE_API_URL}/admin/tasks/export.csv`}
                  download
                />
              }
            >
              <DownloadIcon />
              Все задачи, CSV
            </Button>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
};
