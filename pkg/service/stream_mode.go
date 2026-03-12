package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// ResolveStreamMode determines the stream delivery mode for a channel.
//
// Resolution chain:
//  1. Channel -> ChannelProfile -> StreamProfile.StreamMode
//  2. No profile found -> default "proxy"
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

// ResolveSourceURL returns the URL of the first active stream for the channel.
// Returns empty string if no active stream is found.
func ResolveSourceURL(ctx context.Context, channelID string, channelRepo *repository.ChannelRepository, streamRepo *repository.StreamRepository) string {
	streams, err := channelRepo.GetStreams(ctx, channelID)
	if err != nil || len(streams) == 0 {
		return ""
	}
	for _, cs := range streams {
		stream, err := streamRepo.GetByID(ctx, cs.StreamID)
		if err != nil || !stream.IsActive {
			continue
		}
		return stream.URL
	}
	return ""
}

// ResolveChannelURL returns the stream URL for a channel. For direct-mode
// channels it resolves the upstream source URL; otherwise it returns the
// proxy URL via baseURL.
func ResolveChannelURL(
	ctx context.Context,
	ch *models.Channel,
	baseURL string,
	channelRepo *repository.ChannelRepository,
	streamRepo *repository.StreamRepository,
	channelProfileRepo *repository.ChannelProfileRepository,
	streamProfileRepo *repository.StreamProfileRepository,
	log zerolog.Logger,
) string {
	streamURL := fmt.Sprintf("%s/channel/%s", baseURL, ch.ID)
	mode, _ := ResolveStreamMode(ctx, ch, channelProfileRepo, streamProfileRepo, log)
	if mode == "direct" {
		if src := ResolveSourceURL(ctx, ch.ID, channelRepo, streamRepo); src != "" {
			return src
		}
	}
	return streamURL
}

// lookupStreamProfile follows Channel -> ChannelProfile -> StreamProfile.
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
		log.Warn().Err(err).Str("channel_profile_id", *channel.ChannelProfileID).Msg("channel profile not found")
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
