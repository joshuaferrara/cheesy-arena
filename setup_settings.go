// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web routes for configuring the event settings.

package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Shows the event settings editing page.
func SettingsGetHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	renderSettings(w, r, "")
}

// Saves the event settings.
func SettingsPostHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	eventSettings.Name = r.PostFormValue("name")
	eventSettings.Code = r.PostFormValue("code")
	match, _ := regexp.MatchString("^#([0-9A-Fa-f]{3}){1,2}$", r.PostFormValue("displayBackgroundColor"))
	if !match {
		renderSettings(w, r, "Display background color must be a valid hex color value.")
		return
	}
	eventSettings.DisplayBackgroundColor = r.PostFormValue("displayBackgroundColor")
	numAlliances, _ := strconv.Atoi(r.PostFormValue("numElimAlliances"))
	if numAlliances < 2 || numAlliances > 16 {
		renderSettings(w, r, "Number of alliances must be between 2 and 16.")
		return
	}

	eventSettings.NumElimAlliances = numAlliances
	eventSettings.SelectionRound2Order = r.PostFormValue("selectionRound2Order")
	eventSettings.SelectionRound3Order = r.PostFormValue("selectionRound3Order")
	eventSettings.TBADownloadEnabled = r.PostFormValue("TBADownloadEnabled") == "on"
	eventSettings.TbaPublishingEnabled = r.PostFormValue("tbaPublishingEnabled") == "on"
	eventSettings.TbaEventCode = r.PostFormValue("tbaEventCode")
	eventSettings.TbaSecretId = r.PostFormValue("tbaSecretId")
	eventSettings.TbaSecret = r.PostFormValue("tbaSecret")
	eventSettings.StemTvPublishingEnabled = r.PostFormValue("stemTvPublishingEnabled") == "on"
	eventSettings.StemTvEventCode = r.PostFormValue("stemTvEventCode")
	eventSettings.NetworkSecurityEnabled = r.PostFormValue("networkSecurityEnabled") == "on"
	eventSettings.ApType = r.PostFormValue("apType")
	eventSettings.ApAddress = r.PostFormValue("apAddress")
	eventSettings.ApUsername = r.PostFormValue("apUsername")
	eventSettings.ApPassword = r.PostFormValue("apPassword")
	eventSettings.SwitchAddress = r.PostFormValue("switchAddress")
	eventSettings.SwitchPassword = r.PostFormValue("switchPassword")
	eventSettings.BandwidthMonitoringEnabled = r.PostFormValue("bandwidthMonitoringEnabled") == "on"
	eventSettings.AdminPassword = r.PostFormValue("adminPassword")
	eventSettings.ReaderPassword = r.PostFormValue("readerPassword")
	eventSettings.RedDefenseLightsAddress = r.PostFormValue("redDefenseLightsAddress")
	eventSettings.BlueDefenseLightsAddress = r.PostFormValue("blueDefenseLightsAddress")

	initialTowerStrength, _ := strconv.Atoi(r.PostFormValue("initialTowerStrength"))
	if initialTowerStrength < 1 {
		renderSettings(w, r, "Initial tower strength must be at least 1.")
		return
	}
	eventSettings.InitialTowerStrength = initialTowerStrength

	err := db.SaveEventSettings(eventSettings)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Set up the light controller connections again in case the address changed.
	err = mainArena.lights.SetupConnections()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	http.Redirect(w, r, "/setup/settings", 302)
}

// Sends a copy of the event database file to the client as a download.
func SaveDbHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	dbFile, err := os.Open(db.path)
	defer dbFile.Close()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	filename := fmt.Sprintf("%s-%s.db", strings.Replace(eventSettings.Name, " ", "_", -1),
		time.Now().Format("20060102150405"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	http.ServeContent(w, r, "", time.Now(), dbFile)
}

// Accepts an event database file as an upload and loads it.
func RestoreDbHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	file, _, err := r.FormFile("databaseFile")
	if err != nil {
		renderSettings(w, r, "No database backup file was specified.")
		return
	}

	// Write the file to a temporary location on disk and verify that it can be opened as a database.
	tempFile, err := ioutil.TempFile(".", "uploaded-db-")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)
	_, err = io.Copy(tempFile, file)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	tempFile.Close()
	tempDb, err := OpenDatabase(tempFilePath)
	if err != nil {
		renderSettings(w, r, "Could not read uploaded database backup file. Please verify that it a valid "+
			"database file.")
		return
	}
	tempDb.Close()

	// Back up the current database.
	err = db.Backup("pre_restore")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Replace the current database with the new one.
	db.Close()
	err = os.Remove(eventDbPath)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	err = os.Rename(tempFilePath, eventDbPath)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	initDb()

	http.Redirect(w, r, "/setup/settings", 302)
}

// Deletes all data except for the team list.
func ClearDbHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	// Back up the database.
	err := db.Backup("pre_clear")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	err = db.TruncateMatches()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	err = db.TruncateMatchResults()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	err = db.TruncateRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	err = db.TruncateAllianceTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	http.Redirect(w, r, "/setup/settings", 302)
}

func renderSettings(w http.ResponseWriter, r *http.Request, errorMessage string) {
	template, err := template.ParseFiles("templates/setup_settings.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*EventSettings
		ErrorMessage string
	}{eventSettings, errorMessage}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}
