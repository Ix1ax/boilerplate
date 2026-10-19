import { useParams } from 'react-router';

import {
  OrganizerProfile,
  OrganizerProfileError,
  OrganizerProfileLoading,
} from '@/widgets/organizer-profile';

import { useOrganizationProfile } from '@/entities/organization';

import { ApiError } from '@/shared/api';
import { Badge } from '@/shared/ui';

export const OrganizationPage = () => {
  const { orgId = '' } = useParams();
  const query = useOrganizationProfile(orgId);
  if (!orgId || query.isError)
    return (
      <OrganizerProfileError
        missing={
          !orgId ||
          (query.error instanceof ApiError && query.error.status === 404)
        }
        onRetry={() => void query.refetch()}
      />
    );
  if (query.isPending) return <OrganizerProfileLoading />;
  const { organization, stats, events } = query.data;
  return (
    <OrganizerProfile
      name={organization.name}
      eyebrow="Спортивный клуб"
      description={organization.description}
      stats={stats}
      events={events}
    >
      <div className="flex flex-col gap-3">
        <div>
          <Badge variant="success">Проверенная организация</Badge>
        </div>
        {organization.city && <p>{organization.city}</p>}
        {organization.contacts && (
          <p className="whitespace-pre-wrap break-words">
            Контакты: {organization.contacts}
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          {organization.sports.map((sport) => (
            <Badge key={sport.id} variant="soft">
              {sport.name}
            </Badge>
          ))}
        </div>
      </div>
    </OrganizerProfile>
  );
};
