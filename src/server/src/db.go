package main

import (
	"database/sql"
	"fmt"

	"./msg"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lzhig/rapidgo/base"
)

type mysqlDB struct {
	db *sql.DB
}

func (obj *mysqlDB) open(addr, username, password, dbname string) error {
	var err error
	obj.db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true", username, password, addr, dbname))
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) close() {
	if obj.db != nil {
		obj.db.Close()
	}
}

// 查找fbid，如果存在更新name
func (obj *mysqlDB) getUIDFacebook(fbID string) (uint32, error) {
	var uid uint32
	err := obj.db.QueryRow("select uid from facebook_users where fbid=?", fbID).Scan(&uid)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		base.LogError("[mysql][getUIDFacebook] query uid, error:", err, "fbID:", fbID)
		return 0, err
	default:
		// _, err := obj.db.Exec("update users set name=? where uid=?", name, uid)
		// if err != nil {
		// 	base.LogWarn("[mysql][getUIDFacebook] update name, error:", err, "uid:", uid, ", name:", name)
		// }

		return uid, nil
	}
}

func (obj *mysqlDB) AddFacebookUser(fbID string, uid uint32) error {
	_, err := obj.db.Exec("insert into facebook_users (uid,fbid) values(?,?)",
		uid, fbID)
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) UpdateName(uid uint32, name string) error {
	_, err := obj.db.Exec("update users set name=? where uid=? and name<>? ",
		name, uid, name)
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) CreateUser(name, avatar string, diamonds uint32, platformID uint32) (uint32, error) {
	result, err := obj.db.Exec("insert into users (`name`,avatar,diamonds,platform,regtime,logintime) values(?,?,?,?,UNIX_TIMESTAMP(NOW()),UNIX_TIMESTAMP(NOW()))",
		name, avatar, diamonds, platformID)
	if err != nil {
		return 0, err
	}

	id, _ := result.LastInsertId()
	return uint32(id), nil
}

func (obj *mysqlDB) createFacebookUser(fbID, name string, diamonds uint32) (uint32, error) {
	stmt, err := obj.db.Prepare("CALL create_facebook_user(?,?,?,@uid)")
	if err != nil {
		base.LogError("[mysql][createFacebookUser] failed to prepare sp, error:", err, "fbID:", fbID, ", name:", name)
		return 0, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(fbID, name, diamonds)
	if err != nil {
		base.LogError("[mysql][createFacebookUser] failed to exec, error:", err, "fbID:", fbID, ", name:", name)
		return 0, err
	}

	stmt1, err := obj.db.Prepare("select @uid as uid")
	if err != nil {
		base.LogError("[mysql][createFacebookUser] failed to prepare select, error:", err, "fbID:", fbID, ", name:", name)
		return 0, err
	}
	defer stmt1.Close()

	var uid uint32
	err = stmt1.QueryRow().Scan(&uid)
	if err != nil {
		base.LogError("[mysql][createFacebookUser] failed to scan, error:", err, "fbID:", fbID, ", name:", name)
		return 0, err
	}

	return uid, nil
}

func (obj *mysqlDB) getUserData(uid uint32) (name, avatar string, diamonds uint32, err error) {
	if err := obj.db.QueryRow("select name,avatar,diamonds from users where uid=?", uid).Scan(&name, &avatar, &diamonds); err != nil {
		base.LogError("[mysql][getUserData] error:", err, ",uid:", uid)
		return "", "", 0, err
	}

	return
}

func (obj *mysqlDB) saveUserDiamonds(uid, diamonds uint32) error {
	_, err := obj.db.Exec("update users set diamonds=? where uid=?", diamonds, uid)
	return err
}

func (obj *mysqlDB) createRoom(num int, name string, uid, hands, minBet, maxBet, creditPoints uint32, isShare bool) (uint32, error) {
	result, err := obj.db.Exec("insert into room_records (name,number,owner_uid,hands,played_hands,is_share,min_bet,max_bet,credit_points,create_time,close_time,closed) values(?,?,?,?,0,?,?,?,?,UNIX_TIMESTAMP(NOW()),0,false)",
		name, num, uid, hands, isShare, minBet, maxBet, creditPoints)
	if err != nil {
		return 0, err
	}

	id, _ := result.LastInsertId()
	return uint32(id), nil
}

func (obj *mysqlDB) getRoomNumberUsed() ([]int, error) {
	rows, err := obj.db.Query("select number from room_records where closed=false")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]int, 0, 1024)
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, err
		}
		r = append(r, num)
	}
	return r, nil
}

func (obj *mysqlDB) getRoomCreatedCount(uid uint32) (count uint32, err error) {
	err = obj.db.QueryRow("select count(*) from room_records where owner_uid=? and closed=false", uid).Scan(&count)
	return
}

func (obj *mysqlDB) loadRoom(num uint32) (name string, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints uint32, isShare bool, err error) {
	err = obj.db.QueryRow("select room_id,name,owner_uid,hands,played_hands,is_share,min_bet,max_bet,credit_points from room_records where number=? and closed=false",
		num).Scan(&roomID, &name, &uid, &hands, &playedHands, &isShare, &minBet, &maxBet, &creditPoints)
	return
}

func (obj *mysqlDB) loadScoreboard(roomID uint32) ([]*msg.ScoreboardItem, error) {
	rows, err := obj.db.Query("select a.uid,a.score,b.name,b.avatar from scoreboard as a,users as b where a.roomid=? and a.uid=b.uid order by a.score desc;", roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]*msg.ScoreboardItem, 0, 16)
	for rows.Next() {
		var uid uint32
		var score int32
		var name string
		var avatar string
		if err := rows.Scan(&uid, &score, &name, &avatar); err != nil {
			return nil, err
		}
		r = append(r, &msg.ScoreboardItem{Uid: uid, Score: score, Name: name, Avatar: avatar})
	}
	return r, nil
}

func (obj *mysqlDB) addScoreboardItem(roomID, uid uint32, score int32) error {
	_, err := obj.db.Exec("insert into scoreboard (roomid,uid,score) values(?,?,?)",
		roomID, uid, score)
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) UpdateScoreboardItem(roomID, uid uint32, score int32) error {
	_, err := obj.db.Exec("update scoreboard set score=score+? where roomid=? and uid=?",
		score, roomID, uid)
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) SaveRoundResult(roomID, round uint32, result string) error {
	_, err := obj.db.Exec("insert into game_records (roomid,round,result) values(?,?,?)",
		roomID, round, result)
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) GetRoundResult(roomID, round uint32) (result string, err error) {
	err = obj.db.QueryRow("select result from game_records where roomid=? and round=?",
		roomID, round).Scan(&result)
	return
}

func (obj *mysqlDB) GetUserNameAvatar(uid uint32) (name, avatar string, err error) {
	err = obj.db.QueryRow("select name,avatar from users where uid=?", uid).Scan(&name, &avatar)
	return
}
