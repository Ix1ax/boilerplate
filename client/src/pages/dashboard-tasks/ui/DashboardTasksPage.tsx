import { useState } from 'react';
import { PlusIcon } from 'lucide-react';

import { CreateTaskWizard, TaskTable } from '@/features/tasks';

import { Button, Card, CardContent, CardHeader, CardTitle } from '@/shared/ui';

export const DashboardTasksPage = () => {
  const [wizardOpen, setWizardOpen] = useState(false);
  const [draftId, setDraftId] = useState<string | null>(null);

  const openWizard = (id: string | null = null) => {
    setDraftId(id);
    setWizardOpen(true);
  };

  return (
    <Card>
      <CardHeader className="flex items-center justify-between">
        <CardTitle>Задачи</CardTitle>
        <Button onClick={() => openWizard(null)}>
          <PlusIcon />
          Новая задача
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <TaskTable onContinueDraft={openWizard} />
      </CardContent>
      <CreateTaskWizard
        key={wizardOpen ? (draftId ?? 'new') : 'closed'}
        open={wizardOpen}
        onOpenChange={setWizardOpen}
        draftId={draftId}
      />
    </Card>
  );
};
