package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/ffmpeg"
	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// ClientService handles client detection and management.
type ClientService struct {
	clientRepo        *repository.ClientRepository
	streamProfileRepo *repository.StreamProfileRepository
	log               zerolog.Logger
}

// NewClientService creates a new ClientService.
func NewClientService(
	clientRepo *repository.ClientRepository,
	streamProfileRepo *repository.StreamProfileRepository,
	log zerolog.Logger,
) *ClientService {
	return &ClientService{
		clientRepo:        clientRepo,
		streamProfileRepo: streamProfileRepo,
		log:               log.With().Str("service", "client").Logger(),
	}
}

// MatchClient checks request headers against enabled clients and returns the
// matching stream profile, or nil if no client matches.
func (s *ClientService) MatchClient(ctx context.Context, r *http.Request) (*models.StreamProfile, error) {
	clients, err := s.clientRepo.ListEnabledWithRules(ctx)
	if err != nil {
		return nil, err
	}

	for _, client := range clients {
		if len(client.MatchRules) == 0 {
			continue
		}
		if matchesAllRules(r, client.MatchRules) {
			profile, err := s.streamProfileRepo.GetByID(ctx, client.StreamProfileID)
			if err != nil {
				s.log.Warn().Err(err).Int64("client_id", client.ID).Str("client", client.Name).Msg("client matched but stream profile not found")
				continue
			}
			s.log.Info().Str("client", client.Name).Int64("profile_id", profile.ID).Str("profile", profile.Name).Msg("client detected")
			return profile, nil
		}
	}

	return nil, nil
}

// ListClients returns all clients with their match rules.
func (s *ClientService) ListClients(ctx context.Context) ([]models.Client, error) {
	return s.clientRepo.List(ctx)
}

// GetClient returns a client by ID.
func (s *ClientService) GetClient(ctx context.Context, id int64) (*models.Client, error) {
	return s.clientRepo.GetByID(ctx, id)
}

// CreateClient creates a new client with an auto-created stream profile.
// The profile is created first; if client creation fails, the profile is cleaned up.
func (s *ClientService) CreateClient(ctx context.Context, client *models.Client, rules []models.ClientMatchRule) error {
	// Auto-create a stream profile for this client
	args := ffmpeg.ComposeStreamProfileArgs("m3u", "none", "copy", "mpegts")
	profile := &models.StreamProfile{
		Name:       client.Name,
		StreamMode: "ffmpeg",
		SourceType: "m3u",
		HWAccel:    "none",
		VideoCodec: "copy",
		Container:  "mpegts",
		Command:    "ffmpeg",
		Args:       args,
		IsClient:   true,
	}
	if err := s.streamProfileRepo.Create(ctx, profile); err != nil {
		return fmt.Errorf("creating stream profile: %w", err)
	}

	client.StreamProfileID = profile.ID

	if err := s.clientRepo.Create(ctx, client); err != nil {
		// Clean up the auto-created profile since client creation failed
		s.streamProfileRepo.Delete(ctx, profile.ID) //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("creating client: %w", err)
	}

	if err := s.clientRepo.SetMatchRules(ctx, client.ID, rules); err != nil {
		return fmt.Errorf("setting match rules: %w", err)
	}

	return nil
}

// UpdateClient updates a client and optionally its match rules.
func (s *ClientService) UpdateClient(ctx context.Context, client *models.Client, rules []models.ClientMatchRule) error {
	if err := s.clientRepo.Update(ctx, client); err != nil {
		return fmt.Errorf("updating client: %w", err)
	}

	if rules != nil {
		if err := s.clientRepo.SetMatchRules(ctx, client.ID, rules); err != nil {
			return fmt.Errorf("updating match rules: %w", err)
		}
	}

	return nil
}

// DeleteClient deletes a client and cleans up the orphaned stream profile if applicable.
func (s *ClientService) DeleteClient(ctx context.Context, id int64) error {
	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting client: %w", err)
	}

	profileID := client.StreamProfileID

	if err := s.clientRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting client: %w", err)
	}

	// Clean up orphaned stream profile (only if not referenced by other clients and not a system profile)
	profile, profileErr := s.streamProfileRepo.GetByID(ctx, profileID)
	if profileErr == nil && !profile.IsSystem {
		referenced, refErr := s.clientRepo.IsStreamProfileReferenced(ctx, profileID)
		if refErr == nil && !referenced {
			s.streamProfileRepo.Delete(ctx, profileID) //nolint:errcheck // best-effort cleanup
		}
	}

	return nil
}

// matchesAllRules checks if all match rules are satisfied by the request headers (AND logic).
func matchesAllRules(r *http.Request, rules []models.ClientMatchRule) bool {
	for _, rule := range rules {
		if !matchRule(r, rule) {
			return false
		}
	}
	return true
}

// matchRule checks a single rule against request headers.
func matchRule(r *http.Request, rule models.ClientMatchRule) bool {
	headerValue := r.Header.Get(rule.HeaderName)

	switch rule.MatchType {
	case "exists":
		return headerValue != ""
	case "contains":
		return strings.Contains(headerValue, rule.MatchValue)
	case "equals":
		return headerValue == rule.MatchValue
	case "prefix":
		return strings.HasPrefix(headerValue, rule.MatchValue)
	default:
		return false
	}
}
