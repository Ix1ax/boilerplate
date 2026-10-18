import { FileList, UploadFileForm } from '@/features/files';

import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui';

export const DashboardFilesPage = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Файлы</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <UploadFileForm />
        <FileList />
      </CardContent>
    </Card>
  );
};
