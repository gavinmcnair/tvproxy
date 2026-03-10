package service

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// ResolveStreamMode determines the stream delivery mode for a channel.
//
// Resolution chain:
//  1. Channel → ChannelProfile → StreamProfile.StreamMode
//  2. No profile found → default "proxy"
//
// Returns the resolved mode ("direct", "proxy", or "ffmpeg") and the
// associated StreamProfile (may be nil if no profile is assigned).
func ResolveStreamMode(
	ctx context.Context,
	channel *models.Channel,
	channelProfileRepo *repository.ChannelProfileRepository,
	streamProfileRepo *repository.StreamProfileRepository,
	log zerolog.Logger,
) (string, *models.StreamProfile) {
	profile := lookupStreamProfile(ctx, channel, channelProfileRepo, streamProfileRepo, log)

	if profile != nil {
		return profile.StreamMode, profile
	}

	// No profile found — default to "proxy"
	return "proxy", nil
}

// lookupStreamProfile follows Channel → ChannelProfile → StreamProfile.
// Returns nil if no profile is assigned or cannot be found.
func lookupStreamProfile(
	ctx context.Context,
	channel *models.Channel,
	channelProfileRepo *repository.ChannelProfileRepository,
	streamProfileRepo *repository.StreamProfileRepository,
	log zerolog.Logger,
) *models.StreamProfile {
	if channel.ChannelProfileID == nil {
		return nil
	}

	chanProfile, err := channelProfileRepo.GetByID(ctx, *channel.ChannelProfileID)
	if err != nil {
		log.Warn().Err(err).Int64("channel_profile_id", *channel.ChannelProfileID).Msg("channel profile not found")
		return nil
	}

	if chanProfile.StreamProfile == "" {
		return nil
	}

	streamProfile, err := streamProfileRepo.GetByName(ctx, chanProfile.StreamProfile)
	if err != nil {
		log.Warn().Err(err).Str("stream_profile", chanProfile.StreamProfile).Msg("stream profile not found")
		return nil
	}

	return streamProfile
}
