import { Link } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, Container, PageHeader, Section } from '@/shared/ui';

export const HomePage = () => (
  <Container>
    <Section>
      <PageHeader
        title="Hackathon starter"
        description="Стартовый проект: задачи, приватные файлы и вход через Keycloak."
      />
      <Button render={<Link to={STATIC_ROUTES.DASHBOARD} />}>
        Открыть рабочую область
      </Button>
    </Section>
  </Container>
);
