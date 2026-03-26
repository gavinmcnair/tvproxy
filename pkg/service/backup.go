package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gavinmcnair/tvproxy/pkg/models"
)

type BackupData struct {
	Version             int                          `json:"version"`
	CreatedAt           time.Time                    `json:"created_at"`
	Users               []backupUser                 `json:"users"`
	Settings            map[string]string            `json:"settings"`
	Profiles            []models.StreamProfile       `json:"profiles"`
	Clients             []models.Client              `json:"clients"`
	Logos               []models.Logo                `json:"logos"`
	M3UAccounts         []models.M3UAccount          `json:"m3u_accounts"`
	EPGSources          []models.EPGSource           `json:"epg_sources"`
	HDHRDevices         []models.HDHRDevice          `json:"hdhr_devices"`
	Channels            []models.Channel             `json:"channels"`
	ChannelGroups       []models.ChannelGroup        `json:"channel_groups"`
	ScheduledRecordings []models.ScheduledRecording  `json:"scheduled_recordings"`
}

type backupUser struct {
	models.User
	PasswordHash string `json:"password_hash"`
}

func (r *DataResetter) Backup() ([]byte, error) {
	ctx := context.Background()

	users, _ := r.userStore.List(ctx)
	backupUsers := make([]backupUser, len(users))
	for i, u := range users {
		backupUsers[i] = backupUser{User: u, PasswordHash: u.PasswordHash}
	}

	settings, _ := r.settingsStore.List(ctx)
	settingsMap := make(map[string]string, len(settings))
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	profiles, _ := r.profileStore.List(ctx)
	clients, _ := r.clientStore.List(ctx)
	logos, _ := r.logoStore.List(ctx)
	m3uAccounts, _ := r.m3uAccountStore.List(ctx)
	epgSources, _ := r.epgSourceStore.List(ctx)
	hdhrDevices, _ := r.hdhrStore.List(ctx)
	channels, _ := r.channelStore.List(ctx)
	channelGroups, _ := r.channelGroupSt.List(ctx)
	scheduledRecs, _ := r.scheduledRecSt.List(ctx)

	data := BackupData{
		Version:             1,
		CreatedAt:           time.Now(),
		Users:               backupUsers,
		Settings:            settingsMap,
		Profiles:            profiles,
		Clients:             clients,
		Logos:               logos,
		M3UAccounts:         m3uAccounts,
		EPGSources:          epgSources,
		HDHRDevices:         hdhrDevices,
		Channels:            channels,
		ChannelGroups:       channelGroups,
		ScheduledRecordings: scheduledRecs,
	}

	return json.MarshalIndent(data, "", "  ")
}

func (r *DataResetter) Restore(raw []byte) error {
	var data BackupData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parsing backup: %w", err)
	}
	if data.Version != 1 {
		return fmt.Errorf("unsupported backup version: %d", data.Version)
	}

	if err := r.HardReset(); err != nil {
		return fmt.Errorf("resetting before restore: %w", err)
	}

	ctx := context.Background()

	r.userStore.ClearAndSave()
	for _, u := range data.Users {
		user := u.User
		user.PasswordHash = u.PasswordHash
		r.userStore.Create(ctx, &user)
	}

	for k, v := range data.Settings {
		r.settingsStore.Set(ctx, k, v)
	}

	r.profileStore.ClearAndSave()
	for _, p := range data.Profiles {
		p := p
		r.profileStore.Create(ctx, &p)
	}

	r.clientStore.ClearAndSave()
	for _, c := range data.Clients {
		c := c
		r.clientStore.Create(ctx, &c)
	}

	for _, l := range data.Logos {
		l := l
		r.logoStore.Create(ctx, &l)
	}

	for _, a := range data.M3UAccounts {
		a := a
		r.m3uAccountStore.Create(ctx, &a)
	}

	for _, s := range data.EPGSources {
		s := s
		r.epgSourceStore.Create(ctx, &s)
	}

	for _, d := range data.HDHRDevices {
		d := d
		r.hdhrStore.Create(ctx, &d)
	}

	for _, g := range data.ChannelGroups {
		g := g
		r.channelGroupSt.Create(ctx, &g)
	}

	for _, ch := range data.Channels {
		ch := ch
		r.channelStore.Create(ctx, &ch)
	}

	for _, rec := range data.ScheduledRecordings {
		rec := rec
		r.scheduledRecSt.Create(ctx, &rec)
	}

	return nil
}

