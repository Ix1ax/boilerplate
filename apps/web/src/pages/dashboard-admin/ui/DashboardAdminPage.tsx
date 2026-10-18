import { DownloadIcon } from 'lucide-react';

import { env } from '@/shared/config';
import { Button, Card, CardContent, CardHeader, CardTitle } from '@/shared/ui';

/** Образец админской страницы: отчётные выгрузки. Доступ проверяет Go, вкладка — только удобство. */
export const DashboardAdminPage = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Отчёты</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-muted-foreground text-sm">
          Выгрузка открывается в Excel (UTF-8, разделитель «;»).
        </p>
        <Button
          render={
            <a href={`${env.VITE_API_URL}/admin/tasks/export.csv`} download />
          }
        >
          <DownloadIcon />
          Все задачи, CSV
        </Button>
      </CardContent>
    </Card>
  );
};
