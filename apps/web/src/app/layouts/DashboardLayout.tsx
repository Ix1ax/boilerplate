import { Outlet, useLocation, useNavigate } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import {
  Container,
  PageHeader,
  Section,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@/shared/ui';

const TABS = [
  { value: 'tasks', label: 'Задачи', to: STATIC_ROUTES.DASHBOARD_TASKS },
  { value: 'files', label: 'Файлы', to: STATIC_ROUTES.DASHBOARD_FILES },
];

export const DashboardLayout = () => {
  const location = useLocation();
  const navigate = useNavigate();

  const activeTab =
    TABS.find((tab) => location.pathname.startsWith(tab.to))?.value ?? 'tasks';

  return (
    <Container>
      <Section>
        <PageHeader title="Дашборд" description="Задачи и файлы" />

        <Tabs
          value={activeTab}
          onValueChange={(value) => {
            const tab = TABS.find((item) => item.value === value);
            if (tab) navigate(tab.to);
          }}
        >
          <TabsList>
            {TABS.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <Outlet />
      </Section>
    </Container>
  );
};
