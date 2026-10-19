-- +goose Up
-- Предложение не записывает человека на встречу: участие всегда подтверждается отдельно.
CREATE TABLE demand_proposals (
 id uuid PRIMARY KEY, sport_id uuid NOT NULL REFERENCES sports(id), level text NOT NULL,
 starts_at timestamptz NOT NULL, area text NOT NULL, lat double precision, lng double precision,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','claimed')),
 event_id uuid REFERENCES events(id), expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE demand_proposal_members (
 proposal_id uuid NOT NULL REFERENCES demand_proposals(id), request_id uuid NOT NULL REFERENCES company_requests(id),
 user_id text NOT NULL, declined boolean NOT NULL DEFAULT false, PRIMARY KEY(proposal_id,user_id)
);
CREATE INDEX demand_proposal_members_user ON demand_proposal_members(user_id,proposal_id);
CREATE INDEX demand_proposal_members_request ON demand_proposal_members(request_id);
CREATE TABLE event_arrivals (
 event_id uuid NOT NULL REFERENCES events(id), user_id text NOT NULL,
 status text NOT NULL CHECK(status IN ('arriving','late','here')), eta_minutes int NOT NULL DEFAULT 0 CHECK(eta_minutes BETWEEN 0 AND 120),
 description text NOT NULL DEFAULT '' CHECK(char_length(description)<=200), updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(event_id,user_id), CHECK(status='late' OR eta_minutes=0)
);
-- +goose Down
DROP TABLE event_arrivals;
DROP TABLE demand_proposal_members;
DROP TABLE demand_proposals;
