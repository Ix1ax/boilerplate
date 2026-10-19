export type {
  Event,
  EventInput,
  EventLevel,
  EventPage,
  EventWeather,
  JoinMode,
  MyEntry,
  MyParticipation,
  Organizer,
  ParticipationStatus,
  Participant,
} from '@starter/contracts';

export type EventListParams = {
  limit?: number;
  offset?: number;
  q?: string;
  /** UUID направлений через запятую. */
  sport?: string;
  mode?: 'open' | 'request';
  level?: 'any' | 'beginner' | 'amateur' | 'pro';
  /** Дата `2026-10-20` или RFC 3339. */
  from?: string;
  to?: string;
  organization?: string;
};
