import { useParams } from 'react-router';

import {
  OrganizerProfile,
  OrganizerProfileError,
  OrganizerProfileLoading,
} from '@/widgets/organizer-profile';

import { usePersonProfile } from '@/entities/profile';

import { ApiError } from '@/shared/api';
import { Badge } from '@/shared/ui';

const LEVELS: Record<string, string> = {
  any: 'Любой уровень',
  beginner: 'Новичок',
  amateur: 'Любитель',
  pro: 'Профессионал',
};

export const PersonPage = () => {
  const { userId = '' } = useParams();
  const query = usePersonProfile(userId);

  if (!userId || query.isError) {
    return (
      <OrganizerProfileError
        missing={
          !userId ||
          (query.error instanceof ApiError && query.error.status === 404)
        }
        onRetry={() => void query.refetch()}
      />
    );
  }

  if (query.isPending) return <OrganizerProfileLoading />;

  const profile = query.data;

  return (
    <OrganizerProfile
      name={profile.name}
      eyebrow="Организатор встреч"
      description={profile.about}
      stats={profile.stats}
      events={profile.events}
    >
      <div className="flex flex-wrap gap-2">
        {profile.interests?.map((interest) => (
          <Badge key={interest} variant="soft">
            {interest}
          </Badge>
        ))}
      </div>
      {(profile.city || profile.level) && (
        <p>
          {[profile.city, LEVELS[profile.level]].filter(Boolean).join(' · ')}
        </p>
      )}
    </OrganizerProfile>
  );
};
