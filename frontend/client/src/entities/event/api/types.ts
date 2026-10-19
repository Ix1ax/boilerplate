import type { MyEntry, Participant } from '../model/types';

export type ListParticipantsResponse = {
  items: Participant[];
};

export type GetMyParticipationsResponse = {
  items: MyEntry[];
};

export type RemindParticipantsResponse = {
  sent: number;
};
