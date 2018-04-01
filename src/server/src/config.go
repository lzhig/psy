/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 10:10:59
 * @modify date 2018-01-19 10:10:59
 * @desc [description]
 */

package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
)

type serverConfig struct {
	Addr     string `json:"addr"`
	MaxUsers uint32 `json:"max_users"`
	Gm       string `json:"gm"`
}

type mysqlConfig struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
	Db       string `json:"db"`
	MaxConns int    `json:"max_conns"`
}

type roomConfig struct {
	RoomNameLen            int      `json:"room_name_len"`
	CreditPoints           []uint32 `json:"credit_points"`
	RoomRate               uint32   `json:"room_rate"`
	CountCreated           uint32   `json:"count_created"`
	MaxTablePlayers        uint32   `json:"max_table_players"`
	DealCardsNum           uint32   `json:"deal_cards_num"`
	MaxPlayers             uint32   `json:"max_players"`
	MaxBetRate             uint32   `json:"max_bet_rate"`
	StatesCountdown        []uint32 `json:"states_countdown"`
	ScoreboardCountPerTime uint32   `json:"scoreboard_count_per_time"`
}

type userConfig struct {
	InitDiamonds       uint32 `json:"init_diamonds"`
	FacebookAvatarType string `json:"facebook_avatar_type"`
}

// Config type
type Config struct {
	Version string       `json:"version"`
	Debug   bool         `json:"debug"`
	CPUNum  int          `json:"cpu_num"`
	Server  serverConfig `json:"server"`
	Mysql   mysqlConfig  `json:"mysql"`
	Room    roomConfig   `json:"room"`
	User    userConfig   `json:"user"`
}

// Load function
func (obj *Config) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, obj)
}
