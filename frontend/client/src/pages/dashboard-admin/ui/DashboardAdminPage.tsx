import { DownloadIcon } from 'lucide-react';

import { AuditLogTable } from '@/widgets/admin-audit';
import { AdminStats } from '@/widgets/admin-dashboard';
import { AdminEventsPanel } from '@/widgets/admin-events';
import { AdminOrganizationsPanel } from '@/widgets/admin-organizations';
import { AdminReportsPanel } from '@/widgets/admin-reports';

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
    <Tabs defaultValue="organizations">
      <TabsList>
        <TabsTrigger value="organizations">Организации</TabsTrigger>
        <TabsTrigger value="events">Мероприятия</TabsTrigger>
        <TabsTrigger value="moderation-reports">Жалобы</TabsTrigger>
        <TabsTrigger value="overview">Обзор</TabsTrigger>
        <TabsTrigger value="audit">Журнал</TabsTrigger>
        <TabsTrigger value="export">Отчёты</TabsTrigger>
      </TabsList>
      <TabsContent value="organizations" className="mt-4">
        <AdminOrganizationsPanel />
      </TabsContent>
      <TabsContent value="events" className="mt-4">
        <AdminEventsPanel />
      </TabsContent>
      <TabsContent value="moderation-reports" className="mt-4">
        <AdminReportsPanel />
      </TabsContent>
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
      <TabsContent value="export" className="mt-4">
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
