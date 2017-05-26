// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model and datastore read/write methods for event-level configuration.

package main

type EventSettings struct {
	Id                         int
	Name                       string
	Code                       string
	DisplayBackgroundColor     string
	NumElimAlliances           int
	SelectionRound2Order       string
	SelectionRound3Order       string
	TBADownloadEnabled         bool
	TbaPublishingEnabled       bool
	TbaEventCode               string
	TbaSecretId                string
	TbaSecret                  string
	NetworkSecurityEnabled     bool
    ApType                     string
	ApAddress                  string
	ApUsername                 string
	ApPassword                 string
	SwitchAddress              string
	SwitchPassword             string
	BandwidthMonitoringEnabled bool
	AdminPassword              string
	ReaderPassword             string
	RedDefenseLightsAddress    string
	BlueDefenseLightsAddress   string
	InitialTowerStrength       int
	StemTvPublishingEnabled    bool
	StemTvEventCode            string
}

const eventSettingsId = 0

func (database *Database) GetEventSettings() (*EventSettings, error) {
	eventSettings := new(EventSettings)
	err := database.eventSettingsMap.Get(eventSettings, eventSettingsId)
	if err != nil {
		// Database record doesn't exist yet; create it now.
		eventSettings.Name = "Untitled Event"
		eventSettings.Code = "UE"
		eventSettings.DisplayBackgroundColor = "#00ff00"
		eventSettings.NumElimAlliances = 8
		eventSettings.SelectionRound2Order = "L"
		eventSettings.SelectionRound3Order = ""
		eventSettings.TBADownloadEnabled = true

		// Game-specific default settings.
		eventSettings.InitialTowerStrength = 10

		err = database.eventSettingsMap.Insert(eventSettings)
		if err != nil {
			return nil, err
		}
	}
	return eventSettings, nil
}

func (database *Database) SaveEventSettings(eventSettings *EventSettings) error {
	eventSettings.Id = eventSettingsId
	_, err := database.eventSettingsMap.Update(eventSettings)
	return err
}
