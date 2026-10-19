import type {
  Event as EventDto,
  EventPage as EventPageDto,
  EventWeather as EventWeatherDto,
  HistoryEntry as HistoryEntryDto,
  MyEntry as MyEntryDto,
  MyParticipation as MyParticipationDto,
  Organizer as OrganizerDto,
  Participant as ParticipantDto,
} from '@starter/contracts';

import type { Camelize } from '@/shared/api';

// Входные схемы (создание/правка события) приходят от форм в том же виде, в котором их ждёт
// сервер, — snake_case здесь не переводим.
export type { EventInput, EventPatch } from '@starter/contracts';

export type Event = Camelize<EventDto>;
export type EventPage = Camelize<EventPageDto>;
export type EventWeather = Camelize<EventWeatherDto>;
export type HistoryEntry = Camelize<HistoryEntryDto>;
export type MyEntry = Camelize<MyEntryDto>;
export type MyParticipation = Camelize<MyParticipationDto>;
export type Organizer = Camelize<OrganizerDto>;
export type Participant = Camelize<ParticipantDto>;

export type EventLevel = Event['level'];
export type JoinMode = Event['joinMode'];
export type ParticipationStatus = MyParticipation['status'];

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

export type AdminEventsParams = {
  limit?: number;
  offset?: number;
  /** published, unpublished, deleted, cancelled — через запятую. */
  status?: string;
  q?: string;
};
