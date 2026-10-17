import { CreateTaskForm, TaskList } from '@/features/tasks';

import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui';

export const DashboardTasksPage = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Задачи</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <CreateTaskForm />
        <TaskList />
      </CardContent>
    </Card>
  );
};
