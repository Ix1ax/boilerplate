import { useParams } from 'react-router';
import { organizationStatusLabels } from '@starter/contracts';

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
          <Badge
            variant={organization.status === 'approved' ? 'success' : 'warning'}
          >
            {organizationStatusLabels[organization.status]}
          </Badge>
          <p className="mt-2 text-sm text-muted-foreground">
            Допуск даёт право публиковать встречи на платформе и не является
            оценкой качества занятий. Контакты не публикуются — задайте вопрос в
            обсуждении встречи.
          </p>
        </div>
        {organization.city && <p>{organization.city}</p>}
        {organization.site && /^https?:\/\//.test(organization.site) && (
          <a
            className="underline"
            href={organization.site}
            target="_blank"
            rel="noreferrer"
          >
            Сайт клуба
          </a>
        )}
        {!!organization.zones?.length && (
          <p>Районы: {organization.zones.join(', ')}</p>
        )}
        {!!organization.venues?.length && (
          <ul className="list-disc pl-5">
            {organization.venues.map((venue) => (
              <li key={venue}>{venue}</li>
            ))}
          </ul>
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
