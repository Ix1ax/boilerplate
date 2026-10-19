import { Outlet, useLocation, useNavigate } from 'react-router';

import { dashboardTour, useTour } from '@/features/onboarding';

import { hasRole, useCurrentUser, type UserRole } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import {
  Button,
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
  const { start, running } = useTour(
    dashboardTour,
    user?.id,
    location.pathname,
    true,
  );

  const tabs = TABS.filter(
    (tab) => !tab.roles || hasRole(user ?? null, tab.roles),
  );

  const activeTab =
    tabs.find((tab) => location.pathname.startsWith(tab.to))?.value ?? 'tasks';

  return (
    <Container className="py-6">
      <Section>
        <PageHeader
          title="Дашборд"
          description="Задачи и файлы"
          actions={
            <Button
              variant="outline"
              data-tour="tour-start"
              disabled={!user || running}
              onClick={() => void start()}
            >
              Обучение
            </Button>
          }
        />

        <Tabs
          value={activeTab}
          onValueChange={(value) => {
            const tab = tabs.find((item) => item.value === value);
            if (tab) navigate(tab.to);
          }}
        >
          <TabsList>
            {tabs.map((tab) => (
              <TabsTrigger
                key={tab.value}
                value={tab.value}
                data-tour={`tab-${tab.value}`}
              >
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
