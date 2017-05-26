// Copyright 2014 Team 254. All Rights Reserved.
// Author: josh@ferrara.space (Joshua Ferrara 5805)
//
// Methods for configuring a EnGenius ENH210EXT AP.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"sync"
)

var engeniusTelnetPort = 23
var engeniusMutex sync.Mutex

// Sets up wireless networks for the given set of teams.
func ConfigureTeamWifiEngenius(red1, red2, red3, blue1, blue2, blue3 *Team) error {
	// Make sure multiple configurations aren't being set at the same time.
	engeniusMutex.Lock()
	defer engeniusMutex.Unlock()

	for _, team := range []*Team{red1, red2, red3, blue1, blue2, blue3} {
		if team != nil && (len(team.WpaKey) < 8 || len(team.WpaKey) > 63) {
			return fmt.Errorf("Invalid WPA key '%s' configured for team %d.", team.WpaKey, team.Id)
		}
	}

	addSsidsCommand := ""
	replaceSsid := func(team *Team, vlan int, iface int) {
		if team == nil {
			return
		}
		addSsidsCommand += fmt.Sprintf("wless2 network ssidp %d ssidact 1\n", iface)						// Enable SSID, if it isn't already enabled
		addSsidsCommand += fmt.Sprintf("wless2 network ssidp %d ssid %d\n", iface, team.Id)					// Set SSID
		addSsidsCommand += fmt.Sprintf("wless2 network ssidp %d apsecu 5\n", iface)							// Enable PSK + TKIP/AES security
		addSsidsCommand += fmt.Sprintf("wless2 network ssidp %d apsecu 5 passp %s\n", iface, team.WpaKey)	// Set WPA key
		addSsidsCommand += fmt.Sprintf("wless2 network ssidp %d apsecu 5 accept\n", iface)					// Commit security changes
	}

	replaceSsid(red1, red1Vlan, 2)	// Start replacing SSIDs. EnGenius SSIDs are identified by integers, allowing us to easily replace settings.
	replaceSsid(red2, red2Vlan, 3)
	replaceSsid(red3, red3Vlan, 4)
	replaceSsid(blue1, blue1Vlan, 5)
	replaceSsid(blue2, blue2Vlan, 6)
	replaceSsid(blue3, blue3Vlan, 7)

	// Build and run the overall command to do everything in a single telnet session.
	command := addSsidsCommand
	if len(command) > 0 {
		_, err := runEngeniusConfigCommand(addSsidsCommand)
		if err != nil {
			return err
		}
	}

	return nil
}

// Logs into the EnGenius AP, runs the command, and returns the output. Disconnects from AP after command has been run.
func runEngeniusCommand(command string) (string, error) {
	// Open a Telnet connection to the AP.
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", eventSettings.ApAddress, engeniusTelnetPort))
	if err != nil {
		return "", err
	}
	defer conn.Close()
 
	// Login to the AP, send the command, and log out all at once.
	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("%s\n%s\n%s\nexit\n", eventSettings.ApUsername, eventSettings.ApPassword, command))
	if err != nil {
		return "", err
	}
	err = writer.Flush()
	if err != nil {
		return "", err
	}

	// Read the response.
	var reader bytes.Buffer
	_, err = reader.ReadFrom(conn)
	if err != nil {
		return "", err
	}
	return reader.String(), nil
}

// Logs into the EnGenius AP, runs the configuration command, and uses the EnGenius magic key to drop into shell and restart
// the wireless networking. This allows for fast reconfiguration of the device, rather than having to wait for the entire
// device to restart.
func runEngeniusConfigCommand(command string) (string, error) {
	return runEngeniusCommand(fmt.Sprintf("%s\n1d68d24ea0d9bb6e19949676058f1b93\n/etc/init.d/wireless restart\n",command))
}
