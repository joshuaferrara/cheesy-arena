// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchReview(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()

	match1 := Match{Type: "practice", DisplayName: "1", Status: "complete", Winner: "R"}
	match2 := Match{Type: "practice", DisplayName: "2"}
	match3 := Match{Type: "qualification", DisplayName: "1", Status: "complete", Winner: "B"}
	match4 := Match{Type: "elimination", DisplayName: "SF1-1", Status: "complete", Winner: "T"}
	match5 := Match{Type: "elimination", DisplayName: "SF1-2"}
	db.CreateMatch(&match1)
	db.CreateMatch(&match2)
	db.CreateMatch(&match3)
	db.CreateMatch(&match4)
	db.CreateMatch(&match5)

	// Check that all matches are listed on the page.
	recorder := getHttpResponse("/match_review")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "P1")
	assert.Contains(t, recorder.Body.String(), "P2")
	assert.Contains(t, recorder.Body.String(), "Q1")
	assert.Contains(t, recorder.Body.String(), "SF1-1")
	assert.Contains(t, recorder.Body.String(), "SF1-2")
}

func TestMatchReviewEditExistingResult(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	mainArena.Setup()

	match := Match{Type: "elimination", DisplayName: "QF4-3", Status: "complete", Winner: "R", Red1: 1001,
		Red2: 1002, Red3: 1003, Blue1: 1004, Blue2: 1005, Blue3: 1006}
	db.CreateMatch(&match)
	matchResult := buildTestMatchResult(match.Id, 1)
	matchResult.MatchType = match.Type
	db.CreateMatchResult(&matchResult)
	createTestAlliances(db, 2)

	recorder := getHttpResponse("/match_review")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "QF4-3")
	assert.Contains(t, recorder.Body.String(), "176") // The red score
	assert.Contains(t, recorder.Body.String(), "113") // The blue score

	// Check response for non-existent match.
	recorder = getHttpResponse(fmt.Sprintf("/match_review/%d/edit", 12345))
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such match")

	recorder = getHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id))
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "QF4-3")

	// Update the score to something else.
	postBody := "redScoreJson={\"AutoLowGoals\":5}&blueScoreJson={\"DefensesCrossed\":[2,2,1,2,2]," +
		"\"Fouls\":[{\"TeamId\":973,\"Rule\":\"G22\"}]}&redCardsJson={\"105\":\"yellow\"}&blueCardsJson={}"
	recorder = postHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id), postBody)
	assert.Equal(t, 302, recorder.Code)

	// Check for the updated scores back on the match list page.
	recorder = getHttpResponse("/match_review")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "QF4-3")
	assert.Contains(t, recorder.Body.String(), "30") // The red score
	assert.Contains(t, recorder.Body.String(), "65") // The blue score
}

func TestMatchReviewCreateNewResult(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()

	match := Match{Type: "elimination", DisplayName: "QF4-3", Status: "complete", Winner: "R", Red1: 1001,
		Red2: 1002, Red3: 1003, Blue1: 1004, Blue2: 1005, Blue3: 1006}
	db.CreateMatch(&match)
	createTestAlliances(db, 2)

	recorder := getHttpResponse("/match_review")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "QF4-3")
	assert.NotContains(t, recorder.Body.String(), "176") // The red score
	assert.NotContains(t, recorder.Body.String(), "113") // The blue score

	recorder = getHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id))
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "QF4-3")

	// Update the score to something else.
	postBody := "redScoreJson={\"AutoDefensesReached\":3}&blueScoreJson={\"HighGoals\":3," +
		"\"Fouls\":[{\"TeamId\":973,\"Rule\":\"G22\"}]}&redCardsJson={\"105\":\"yellow\"}&blueCardsJson={}"
	recorder = postHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id), postBody)
	assert.Equal(t, 302, recorder.Code)

	// Check for the updated scores back on the match list page.
	recorder = getHttpResponse("/match_review")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "QF4-3")
	assert.Contains(t, recorder.Body.String(), "11") // The red score
	assert.Contains(t, recorder.Body.String(), "15") // The blue score
}
