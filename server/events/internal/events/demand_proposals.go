package events

import (
	"context"
	"math"
	"strings"
	"time"
)

type DemandProposal struct {
	ID                  string    `json:"id"`
	SportID             string    `json:"sport_id"`
	Level               string    `json:"level"`
	StartsAt            time.Time `json:"starts_at"`
	Area                string    `json:"area"`
	RequiresCoordinates bool      `json:"requires_coordinates"`
	Status              string    `json:"status"`
	EventID             *string   `json:"event_id"`
	ExpiresAt           time.Time `json:"expires_at"`
	MembersCount        int       `json:"members_count"`
	Declined            bool      `json:"declined"`
}

type ProposalClaim struct {
	Location string   `json:"location"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
}

type demandLiveStore interface {
	DemandProposals(context.Context, Actor) ([]DemandProposal, error)
	ClaimDemandProposal(context.Context, Actor, string, ProposalClaim) (Event, error)
	DeclineDemandProposal(context.Context, Actor, string) error
	Arrivals(context.Context, Actor, string) ([]Arrival, error)
	SetArrival(context.Context, Actor, string, ArrivalInput) (Arrival, error)
}

type demandCandidate struct {
	ID, UserID, SportID, Level, Area string
	Lat, Lng                         *float64
	Radius                           *int
	Parts                            []string
	Days                             []int
	ExpiresAt                        time.Time
}

func demandDistance(a, b demandCandidate) float64 {
	rad := math.Pi / 180
	v := math.Pow(math.Sin((*a.Lat-*b.Lat)*rad/2), 2) + math.Cos(*a.Lat*rad)*math.Cos(*b.Lat*rad)*math.Pow(math.Sin((*a.Lng-*b.Lng)*rad/2), 2)
	return 12742 * math.Asin(math.Sqrt(math.Min(1, v)))
}

// Проверяем именно предложенную точку каждого участника, а не только попарную близость районов.
func demandPlaceFits(anchor, other demandCandidate) bool {
	if anchor.Lat != nil && other.Lat != nil && other.Radius != nil {
		return demandDistance(anchor, other) <= float64(*other.Radius)
	}
	return strings.TrimSpace(anchor.Area) != "" && strings.EqualFold(strings.Join(strings.Fields(anchor.Area), " "), strings.Join(strings.Fields(other.Area), " "))
}

func demandAllows(c demandCandidate, slot time.Time, part string) bool {
	day := int(slot.Weekday())
	if day == 0 {
		day = 7
	}
	allowedDay, allowedPart := len(c.Days) == 0, len(c.Parts) == 0
	for _, d := range c.Days {
		if d == day {
			allowedDay = true
		}
	}
	for _, p := range c.Parts {
		if p == part {
			allowedPart = true
		}
	}
	return allowedDay && allowedPart && slot.Before(c.ExpiresAt)
}

func demandSlot(now time.Time, group []demandCandidate) (time.Time, bool) {
	local := now.In(localZone)
	for d := 0; d < 14; d++ {
		for _, part := range []struct {
			name string
			hour int
		}{{"morning", 10}, {"day", 14}, {"evening", 19}} {
			slot := time.Date(local.Year(), local.Month(), local.Day()+d, part.hour, 0, 0, 0, localZone)
			if slot.Before(now.Add(2 * time.Hour)) {
				continue
			}
			ok := true
			for _, c := range group {
				if !demandAllows(c, slot, part.name) {
					ok = false
					break
				}
			}
			if ok {
				return slot, true
			}
		}
	}
	return time.Time{}, false
}

// RefreshDemandProposals — фоновое задание: единственное место, где собираются новые предложения.
func (s *Service) RefreshDemandProposals(ctx context.Context) error {
	store, ok := s.Store.(interface {
		BuildDemandProposals(context.Context) error
	})
	if !ok {
		return nil
	}
	return store.BuildDemandProposals(ctx)
}
