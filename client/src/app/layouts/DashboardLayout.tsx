import { Outlet, useLocation, useNavigate } from 'react-router';

import { hasRole, useCurrentUser, type UserRole } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import {
  Container,
  PageHeader,
  Section,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@/shared/ui';

type Tab = {
  value: string;
  label: string;
  to: string;
  roles?: readonly UserRole[];
};

const TABS: Tab[] = [
  { value: 'tasks', label: 'Задачи', to: STATIC_ROUTES.DASHBOARD_TASKS },
  { value: 'files', label: 'Файлы', to: STATIC_ROUTES.DASHBOARD_FILES },
  {
    value: 'assistant',
    label: 'Ассистент',
    to: STATIC_ROUTES.DASHBOARD_ASSISTANT,
  },
  { value: 'live', label: 'Лайв', to: STATIC_ROUTES.DASHBOARD_LIVE },
  {
    value: 'admin',
    label: 'Админка',
    to: STATIC_ROUTES.DASHBOARD_ADMIN,
    roles: ['admin'],
  },
];

export const DashboardLayout = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const { data: user } = useCurrentUser();
  // Вкладки по ролям — только удобство интерфейса; доступ к данным проверяет Go.
  const tabs = TABS.filter(
    (tab) => !tab.roles || hasRole(user ?? null, tab.roles),
  );

  const activeTab =
    tabs.find((tab) => location.pathname.startsWith(tab.to))?.value ?? 'tasks';

  return (
    <Container>
      <Section>
        <PageHeader title="Дашборд" description="Задачи и файлы" />

        <Tabs
          value={activeTab}
          onValueChange={(value) => {
            const tab = tabs.find((item) => item.value === value);
            if (tab) navigate(tab.to);
          }}
        >
          <TabsList>
            {tabs.map((tab) => (
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
