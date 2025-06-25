package main

import (
	"encoding/json"
	"fmt"
	"github.com/oliverbestmann/union-station/fetch"
	"net/url"
	"strconv"
)

type Leaderboard struct {
	Items []LeaderboardItem
}

type LeaderboardItem struct {
	Player string `json:"player"`
	Score  int    `json:"score"`
}

func ReportHighscore(seed uint64, player string, score int) Promise[Leaderboard, struct{}] {
	values := url.Values{}
	values.Set("player", player)
	values.Set("score", strconv.Itoa(score))

	seedStr := strconv.Itoa(int(seed))
	uri := "https://highscore.narf.zone/games/union-station:dev:" + seedStr + "?" + values.Encode()

	return AsyncTask(func(yield func(struct{})) (result Leaderboard) {
		err := json.NewDecoder(fetch.Post(uri)).Decode(&result.Items)
		if err != nil {
			fmt.Printf("[err] decoding leaderboard response failed: %s", err)
			result.Items = []LeaderboardItem{{Player: player, Score: score}}
		}

		return
	})
}
