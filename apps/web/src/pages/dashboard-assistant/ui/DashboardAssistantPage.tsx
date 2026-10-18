import { AssistantChat } from '@/features/assistant';

import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui';

export const DashboardAssistantPage = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Ассистент</CardTitle>
      </CardHeader>
      <CardContent>
        <AssistantChat />
      </CardContent>
    </Card>
  );
};
