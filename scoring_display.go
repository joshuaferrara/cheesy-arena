// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for scoring interface.

package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"
	"text/template"
)

// Renders the scoring interface which enables input of scores in real-time.
func ScoringDisplayHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)
	alliance := vars["alliance"]
	if alliance != "red" && alliance != "blue" {
		handleWebErr(w, fmt.Errorf("Invalid alliance '%s'.", alliance))
		return
	}

	template, err := template.ParseFiles("templates/scoring_display.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*EventSettings
		Alliance string
	}{eventSettings, alliance}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the scoring interface client to send control commands and receive status updates.
func ScoringDisplayWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !UserIsAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)
	alliance := vars["alliance"]
	if alliance != "red" && alliance != "blue" {
		handleWebErr(w, fmt.Errorf("Invalid alliance '%s'.", alliance))
		return
	}
	var score **RealtimeScore
	if alliance == "red" {
		score = &mainArena.redRealtimeScore
	} else {
		score = &mainArena.blueRealtimeScore
	}
	autoCommitted := false

	websocket, err := NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer websocket.Close()

	matchLoadTeamsListener := mainArena.matchLoadTeamsNotifier.Listen()
	defer close(matchLoadTeamsListener)
	matchTimeListener := mainArena.matchTimeNotifier.Listen()
	defer close(matchTimeListener)
	reloadDisplaysListener := mainArena.reloadDisplaysNotifier.Listen()
	defer close(reloadDisplaysListener)

	// Send the various notifications immediately upon connection.
	data := struct {
		Score *RealtimeScore
		AutoCommitted bool
	}{*score, autoCommitted}
	err = websocket.Write("score", data)
	if err != nil {
		log.Printf("Websocket error: %s", err)
		return
	}
	err = websocket.Write("matchTime", MatchTimeMessage{mainArena.MatchState, int(mainArena.lastMatchTimeSec)})
	if err != nil {
		log.Printf("Websocket error: %s", err)
		return
	}

	// Spin off a goroutine to listen for notifications and pass them on through the websocket.
	go func() {
		for {
			var messageType string
			var message interface{}
			select {
			case _, ok := <-matchLoadTeamsListener:
				if !ok {
					return
				}
				messageType = "score"
				message = *score
			case matchTimeSec, ok := <-matchTimeListener:
				if !ok {
					return
				}
				messageType = "matchTime"
				message = MatchTimeMessage{mainArena.MatchState, matchTimeSec.(int)}
			case _, ok := <-reloadDisplaysListener:
				if !ok {
					return
				}
				messageType = "reload"
				message = nil
			}
			err = websocket.Write(messageType, message)
			if err != nil {
				// The client has probably closed the connection; nothing to do here.
				return
			}
		}
	}()

	// Loop, waiting for commands and responding to them, until the client closes the connection.
	for {
		messageType, data, err := websocket.Read()
		if err != nil {
			if err == io.EOF {
				// Client has closed the connection; nothing to do here.
				return
			}
			log.Printf("Websocket error: %s", err)
			return
		}

		switch messageType {
		case "defenseCrossed":
			position, ok := data.(string)
			if !ok {
				websocket.WriteError("Defense position is not a string.")
				continue
			}
			intPosition, err := strconv.Atoi(position)
			if err != nil {
				websocket.WriteError(err.Error())
				continue
			}
			if (*score).CurrentScore.AutoDefensesCrossed[intPosition-1]+
				(*score).CurrentScore.DefensesCrossed[intPosition-1] < 2 {
				if !autoCommitted {
					(*score).CurrentScore.AutoDefensesCrossed[intPosition-1]++
				} else {
					(*score).CurrentScore.DefensesCrossed[intPosition-1]++
				}
			}
		case "undoDefenseCrossed":
			position, ok := data.(string)
			if !ok {
				websocket.WriteError("Defense position is not a string.")
				continue
			}
			intPosition, err := strconv.Atoi(position)
			if err != nil {
				websocket.WriteError(err.Error())
				continue
			}
			if !autoCommitted {
				if (*score).CurrentScore.AutoDefensesCrossed[intPosition-1] > 0 {
					(*score).CurrentScore.AutoDefensesCrossed[intPosition-1]--
				}
			} else {
				if (*score).CurrentScore.DefensesCrossed[intPosition-1] > 0 {
					(*score).CurrentScore.DefensesCrossed[intPosition-1]--
				}
			}
		case "autoDefenseReached":
			if !autoCommitted {
				if (*score).CurrentScore.AutoDefensesReached < 3 {
					(*score).CurrentScore.AutoDefensesReached++
				}
			}
		case "undoAutoDefenseReached":
			if !autoCommitted {
				if (*score).CurrentScore.AutoDefensesReached > 0 {
					(*score).CurrentScore.AutoDefensesReached--
				}
			}
		case "highGoal":
			if !autoCommitted {
				(*score).CurrentScore.AutoHighGoals++
			} else {
				(*score).CurrentScore.HighGoals++
			}
		case "undoHighGoal":
			if !autoCommitted {
				if (*score).CurrentScore.AutoHighGoals > 0 {
					(*score).CurrentScore.AutoHighGoals--
				}
			} else {
				if (*score).CurrentScore.HighGoals > 0 {
					(*score).CurrentScore.HighGoals--
				}
			}
		case "lowGoal":
			if !autoCommitted {
				(*score).CurrentScore.AutoLowGoals++
			} else {
				(*score).CurrentScore.LowGoals++
			}
		case "undoLowGoal":
			if !autoCommitted {
				if (*score).CurrentScore.AutoLowGoals > 0 {
					(*score).CurrentScore.AutoLowGoals--
				}
			} else {
				if (*score).CurrentScore.LowGoals > 0 {
					(*score).CurrentScore.LowGoals--
				}
			}
		case "challenge":
			if autoCommitted {
				if (*score).CurrentScore.Challenges < 3 {
					(*score).CurrentScore.Challenges++
				}
			}
		case "undoChallenge":
			if autoCommitted {
				if (*score).CurrentScore.Challenges > 0 {
					(*score).CurrentScore.Challenges--
				}
			}
		case "scale":
			if autoCommitted {
				if (*score).CurrentScore.Scales < 3 {
					(*score).CurrentScore.Scales++
				}
			}
		case "undoScale":
			if autoCommitted {
				if (*score).CurrentScore.Scales > 0 {
					(*score).CurrentScore.Scales--
				}
			}
		case "commit":
			if mainArena.MatchState != PRE_MATCH || mainArena.currentMatch.Type == "test" {
				autoCommitted = true
			}
		case "uncommitAuto":
			autoCommitted = false
		case "commitMatch":
			if mainArena.MatchState != POST_MATCH {
				// Don't allow committing the score until the match is over.
				websocket.WriteError("Cannot commit score: Match is not over.")
				continue
			}

			autoCommitted = true
			(*score).TeleopCommitted = true
			mainArena.scoringStatusNotifier.Notify(nil)
		default:
			websocket.WriteError(fmt.Sprintf("Invalid message type '%s'.", messageType))
			continue
		}

		mainArena.realtimeScoreNotifier.Notify(nil)

		// Send out the score again after handling the command, as it most likely changed as a result.
		data = struct {
			Score *RealtimeScore
			AutoCommitted bool
		}{*score, autoCommitted}
		err = websocket.Write("score", data)
		if err != nil {
			log.Printf("Websocket error: %s", err)
			return
		}
	}
}
