package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/config"
	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type OutputService struct {
	channelRepo        *repository.ChannelRepository
	channelGroupRepo   *repository.ChannelGroupRepository
	streamRepo         *repository.StreamRepository
	channelProfileRepo *repository.ChannelProfileRepository
	streamProfileRepo  *repository.StreamProfileRepository
	epgDataRepo        *repository.EPGDataRepository
	programDataRepo    *repository.ProgramDataRepository
	adminUserID        string
	config             *config.Config
	log                zerolog.Logger
}

func NewOutputService(
	channelRepo *repository.ChannelRepository,
	channelGroupRepo *repository.ChannelGroupRepository,
	streamRepo *repository.StreamRepository,
	channelProfileRepo *repository.ChannelProfileRepository,
	streamProfileRepo *repository.StreamProfileRepository,
	epgDataRepo *repository.EPGDataRepository,
	programDataRepo *repository.ProgramDataRepository,
	adminUserID string,
	cfg *config.Config,
	log zerolog.Logger,
) *OutputService {
	return &OutputService{
		channelRepo:        channelRepo,
		channelGroupRepo:   channelGroupRepo,
		streamRepo:         streamRepo,
		channelProfileRepo: channelProfileRepo,
		streamProfileRepo:  streamProfileRepo,
		epgDataRepo:        epgDataRepo,
		programDataRepo:    programDataRepo,
		adminUserID:        adminUserID,
		config:             cfg,
		log:                log.With().Str("service", "output").Logger(),
	}
}

const placeholderLogo = `data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='200' viewBox='0 0 200 200'%3E%3Crect width='200' height='200' rx='20' fill='%23374151'/%3E%3Ctext x='100' y='115' font-family='sans-serif' font-size='80' fill='%239CA3AF' text-anchor='middle'%3ETV%3C/text%3E%3C/svg%3E`

func channelEPGID(ch models.Channel) string {
	if ch.TvgID != "" {
		return ch.TvgID
	}
	return fmt.Sprintf("tvproxy.%s", ch.ID)
}

func (s *OutputService) listChannels(ctx context.Context) ([]models.Channel, error) {
	if s.adminUserID != "" {
		return s.channelRepo.ListByUserID(ctx, s.adminUserID)
	}
	return s.channelRepo.List(ctx)
}

func (s *OutputService) listChannelGroups(ctx context.Context) ([]models.ChannelGroup, error) {
	if s.adminUserID != "" {
		return s.channelGroupRepo.ListByUserID(ctx, s.adminUserID)
	}
	return s.channelGroupRepo.List(ctx)
}

func (s *OutputService) GenerateM3U(ctx context.Context) (string, error) {
	baseURL := fmt.Sprintf("%s:%d", s.config.BaseURL, s.config.Port)
	return s.generateM3U(ctx, nil, baseURL)
}

// GenerateM3UForGroups generates an M3U playlist filtered to channels in the
// given groups. If groupIDs is empty, all enabled channels are included.
func (s *OutputService) GenerateM3UForGroups(ctx context.Context, groupIDs []string, baseURL string) (string, error) {
	if len(groupIDs) == 0 {
		return s.GenerateM3U(ctx)
	}
	groupSet := make(map[string]bool, len(groupIDs))
	for _, gid := range groupIDs {
		groupSet[gid] = true
	}
	return s.generateM3U(ctx, groupSet, baseURL)
}

func (s *OutputService) GenerateEPG(ctx context.Context) (string, error) {
	return s.generateEPG(ctx, nil)
}

// GenerateEPGForGroups generates XMLTV EPG data filtered to channels in the
// given groups. If groupIDs is empty, all data is included.
func (s *OutputService) GenerateEPGForGroups(ctx context.Context, groupIDs []string) (string, error) {
	if len(groupIDs) == 0 {
		return s.GenerateEPG(ctx)
	}
	groupSet := make(map[string]bool, len(groupIDs))
	for _, gid := range groupIDs {
		groupSet[gid] = true
	}
	return s.generateEPG(ctx, groupSet)
}

func (s *OutputService) generateM3U(ctx context.Context, groupFilter map[string]bool, baseURL string) (string, error) {
	channels, err := s.listChannels(ctx)
	if err != nil {
		return "", fmt.Errorf("listing channels: %w", err)
	}

	groups, err := s.listChannelGroups(ctx)
	if err != nil {
		return "", fmt.Errorf("listing channel groups: %w", err)
	}
	groupNames := make(map[string]string, len(groups))
	for _, g := range groups {
		groupNames[g.ID] = g.Name
	}

	var b strings.Builder
	b.WriteString("#EXTM3U\n")

	for _, ch := range channels {
		if !ch.IsEnabled {
			continue
		}
		if groupFilter != nil {
			if ch.ChannelGroupID == nil || !groupFilter[*ch.ChannelGroupID] {
				continue
			}
		}

		b.WriteString("#EXTINF:-1")

		// tvg-id so Plex can link to the EPG
		b.WriteString(fmt.Sprintf(" tvg-id=\"%s\"", channelEPGID(ch)))
		b.WriteString(fmt.Sprintf(" tvg-name=\"%s\"", ch.Name))

		// tvg-logo with placeholder fallback
		logo := ch.Logo
		if logo == "" {
			logo = placeholderLogo
		}
		b.WriteString(fmt.Sprintf(" tvg-logo=\"%s\"", logo))

		if ch.ChannelGroupID != nil {
			if name, ok := groupNames[*ch.ChannelGroupID]; ok {
				b.WriteString(fmt.Sprintf(" group-title=\"%s\"", name))
			}
		}

		b.WriteString(fmt.Sprintf(",%s\n", ch.Name))

		streamURL := ResolveChannelURL(ctx, &ch, baseURL, s.channelRepo, s.streamRepo, s.channelProfileRepo, s.streamProfileRepo, s.log)
		b.WriteString(streamURL + "\n")
	}

	return b.String(), nil
}

func (s *OutputService) generateEPG(ctx context.Context, groupFilter map[string]bool) (string, error) {
	channels, err := s.listChannels(ctx)
	if err != nil {
		return "", fmt.Errorf("listing channels: %w", err)
	}

	enabledTvgIDs := make(map[string]bool, len(channels))
	var noEPGChannels []models.Channel
	for _, ch := range channels {
		if !ch.IsEnabled {
			continue
		}
		if groupFilter != nil {
			if ch.ChannelGroupID == nil || !groupFilter[*ch.ChannelGroupID] {
				continue
			}
		}
		if ch.TvgID != "" {
			enabledTvgIDs[ch.TvgID] = true
		} else {
			noEPGChannels = append(noEPGChannels, ch)
		}
	}

	epgDataList, err := s.epgDataRepo.List(ctx)
	if err != nil {
		return "", fmt.Errorf("listing epg data: %w", err)
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE tv SYSTEM "xmltv.dtd">` + "\n")
	b.WriteString(`<tv generator-info-name="tvproxy">` + "\n")

	for _, epg := range epgDataList {
		if !enabledTvgIDs[epg.ChannelID] {
			continue
		}
		s.writeXMLChannel(&b, epg.ChannelID, epg.Name, epg.Icon)
	}

	for _, ch := range noEPGChannels {
		logo := ch.Logo
		if logo == "" {
			logo = placeholderLogo
		}
		s.writeXMLChannel(&b, channelEPGID(ch), ch.Name, logo)
	}

	for _, epg := range epgDataList {
		if !enabledTvgIDs[epg.ChannelID] {
			continue
		}
		programs, err := s.programDataRepo.ListByEPGDataID(ctx, epg.ID)
		if err != nil {
			s.log.Error().Err(err).Str("epg_data_id", epg.ID).Msg("failed to list programs")
			continue
		}
		for _, prog := range programs {
			s.writeXMLProgramme(&b, epg.ChannelID, prog)
		}
	}

	b.WriteString("</tv>\n")
	return b.String(), nil
}

func (s *OutputService) writeXMLChannel(b *strings.Builder, id, name, icon string) {
	b.WriteString(fmt.Sprintf(`  <channel id="%s">`, xmlEscape(id)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`    <display-name>%s</display-name>`, xmlEscape(name)))
	b.WriteString("\n")
	if icon != "" {
		b.WriteString(fmt.Sprintf(`    <icon src="%s" />`, xmlEscape(icon)))
		b.WriteString("\n")
	}
	b.WriteString("  </channel>\n")
}

func (s *OutputService) writeXMLProgramme(b *strings.Builder, channelID string, prog models.ProgramData) {
	start := prog.Start.Format("20060102150405 -0700")
	stop := prog.Stop.Format("20060102150405 -0700")

	b.WriteString(fmt.Sprintf(`  <programme start="%s" stop="%s" channel="%s">`,
		start, stop, xmlEscape(channelID)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`    <title>%s</title>`, xmlEscape(prog.Title)))
	b.WriteString("\n")

	if prog.Description != "" {
		b.WriteString(fmt.Sprintf(`    <desc>%s</desc>`, xmlEscape(prog.Description)))
		b.WriteString("\n")
	}
	if prog.Category != "" {
		b.WriteString(fmt.Sprintf(`    <category>%s</category>`, xmlEscape(prog.Category)))
		b.WriteString("\n")
	}
	if prog.EpisodeNum != "" {
		b.WriteString(fmt.Sprintf(`    <episode-num system="onscreen">%s</episode-num>`, xmlEscape(prog.EpisodeNum)))
		b.WriteString("\n")
	}
	if prog.Icon != "" {
		b.WriteString(fmt.Sprintf(`    <icon src="%s" />`, xmlEscape(prog.Icon)))
		b.WriteString("\n")
	}

	b.WriteString("  </programme>\n")
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
