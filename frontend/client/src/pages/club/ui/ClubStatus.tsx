import { Link } from 'react-router';
import { organizationStatusLabels } from '@starter/contracts';

import type { Organization } from '@/entities/organization';

import { STATIC_ROUTES } from '@/shared/constants';
import { Badge, Button } from '@/shared/ui';

interface ClubStatusProps {
  organization: Organization;
}

export const ClubStatus = ({ organization }: ClubStatusProps) => (
  <section className="flex flex-col items-start gap-3 rounded-3xl bg-card p-6">
    <Badge variant={organization.status === 'approved' ? 'success' : 'warning'}>
      {organizationStatusLabels[organization.status]}
    </Badge>
    <p className="text-sm text-muted-foreground">
      Заявка отправлена{' '}
      {new Date(organization.createdAt).toLocaleDateString('ru-RU')}
      {organization.reviewedAt &&
        ` · решение от ${new Date(organization.reviewedAt).toLocaleDateString('ru-RU')}`}
    </p>
    {organization.status === 'pending' && (
      <p>
        Заявка у модератора. После одобрения вы сможете публиковать встречи от
        имени клуба.
      </p>
    )}
    {organization.status === 'rejected' && (
      <>
        <p>Причина отказа: {organization.reviewReason}</p>
        <p>Исправьте профиль и отправьте его повторно.</p>
      </>
    )}
    {organization.status === 'approved' && (
      <Button size="lg" render={<Link to={STATIC_ROUTES.EVENT_NEW} />}>
        Создать встречу
      </Button>
    )}
  </section>
);
