package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"./msg"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lzhig/rapidgo/base"
)

type mysqlDB struct {
	db *sql.DB
}

func (obj *mysqlDB) open(addr, username, password, dbname string, maxConns int) error {
	var err error
	obj.db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true", username, password, addr, dbname))
	if err != nil {
		return err
	}
	obj.db.SetMaxIdleConns(maxConns)
	obj.db.SetMaxOpenConns(maxConns)
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
		base.LogError("[mysql][getUIDFacebook] query uid, error:", err, ", fbID:", fbID)
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
	result, err := obj.db.Exec("insert into users (`name`,avatar,diamonds,platform,regtime,logintime,status) values(?,?,?,?,UNIX_TIMESTAMP(NOW()),UNIX_TIMESTAMP(NOW()),0)",
		name, avatar, diamonds, platformID)
	if err != nil {
		base.LogError("failed to create user. error:", err)
		return 0, err
	}

	id, _ := result.LastInsertId()
	return uint32(id), nil
}

// func (obj *mysqlDB) createFacebookUser(fbID, name string, diamonds uint32) (uint32, error) {
// 	stmt, err := obj.db.Prepare("CALL create_facebook_user(?,?,?,@uid)")
// 	if err != nil {
// 		base.LogError("[mysql][createFacebookUser] failed to prepare sp, error:", err, "fbID:", fbID, ", name:", name)
// 		return 0, err
// 	}
// 	defer stmt.Close()

// 	_, err = stmt.Exec(fbID, name, diamonds)
// 	if err != nil {
// 		base.LogError("[mysql][createFacebookUser] failed to exec, error:", err, "fbID:", fbID, ", name:", name)
// 		return 0, err
// 	}

// 	stmt1, err := obj.db.Prepare("select @uid as uid")
// 	if err != nil {
// 		base.LogError("[mysql][createFacebookUser] failed to prepare select, error:", err, "fbID:", fbID, ", name:", name)
// 		return 0, err
// 	}
// 	defer stmt1.Close()

// 	var uid uint32
// 	err = stmt1.QueryRow().Scan(&uid)
// 	if err != nil {
// 		base.LogError("[mysql][createFacebookUser] failed to scan, error:", err, "fbID:", fbID, ", name:", name)
// 		return 0, err
// 	}

// 	return uid, nil
// }

// func (obj *mysqlDB) saveUserDiamonds(uid, diamonds uint32) error {
// 	_, err := obj.db.Exec("update users set diamonds=? where uid=?", diamonds, uid)
// 	return err
// }

func (obj *mysqlDB) GiveFreeDiamonds(uid, diamonds uint32) error {
	// 今天0点
	now := time.Now()
	today := base.GetTodayZeroClockTime(&now).Unix()
	success := false
	tx, err := obj.db.Begin()
	if err != nil {
		base.LogError("GiveFreeDiamonds: failed to start a transaction, error:", err)
		return err
	}
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	var lastTime int64
	err = tx.QueryRow("select time from free_diamonds where uid=?", uid).Scan(&lastTime)
	switch {
	case err == sql.ErrNoRows:
		// 用户还没有领取过
	case err != nil:
		base.LogError("GiveFreeDiamonds: query uid, error:", err, ", uid:", uid)
		return err
	default:
		if lastTime >= today {
			// 今天已经领取过了
			return nil
		}
	}

	var userDiamonds uint32
	err = tx.QueryRow("select diamonds from users where uid=?", uid).Scan(&userDiamonds)
	switch {
	case err == sql.ErrNoRows:
		base.LogError("Cannot find the user. uid:", uid)
		return nil
	case err != nil:
		base.LogError("error:", err)
		return err
	}

	result, err := tx.Exec("update users set diamonds=? where uid=? and diamonds<?", diamonds, uid, diamonds)
	if err != nil {
		base.LogError("GiveFreeDiamonds: failed to exec, error:", err, ", uid:", uid, ", diamonds:", diamonds)
		return err
	}
	n, _ := result.RowsAffected()
	if n == 1 {
		_, err := tx.Exec("REPLACE INTO free_diamonds (`time`,uid) values(?,?)", time.Now().Unix(), uid)
		if err != nil {
			base.LogError("GiveFreeDiamonds: failed to exec, error:", err, ", uid:", uid, ", today:", today)
			return err
		}

		_, err = tx.Exec("insert into diamond_records (`timestamp`,`from`,`to`,diamonds) values(?,?,?,?)", time.Now().Unix(), 0, uid, userDiamonds)
		if err != nil {
			base.LogError("failed to update db. error:", err)
			return err
		}

		success = true
		tx.Commit()
		return nil
	}

	return nil
}

// ErrorNotEnoughDiamonds 错误
var ErrorNotEnoughDiamonds = errors.New("diamonds are not enough")

func (obj *mysqlDB) ConsumeDiamonds(uids []uint32, diamonds uint32) ([]uint32, error) {
	success := false
	//usersDiamonds := make([]uint32, len(uids))
	tx, err := obj.db.Begin()
	if err != nil {
		base.LogError("PayDiamonds: failed to start a transaction, error:", err)
		return nil, err
	}
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	nomoneyUids := make([]uint32, 0, len(uids))
	allUsersHaveEnoughDiamonds := true
	for _, uid := range uids {
		result, err := tx.Exec("update users set diamonds=diamonds-? where uid=? and diamonds >= ?", diamonds, uid, diamonds)
		if err != nil {
			return nil, err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows != 1 {
			allUsersHaveEnoughDiamonds = false
			nomoneyUids = append(nomoneyUids, uid)
			//return nil, ErrorNotEnoughDiamonds
		}

		// _, err = tx.Exec("update users set diamonds=diamonds+? where uid=?", diamonds, to)
		// if err != nil {
		// 	base.LogError("failed to update db. error:", err)
		// 	return nil, err
		// }

		if allUsersHaveEnoughDiamonds {
			_, err = tx.Exec("insert into diamond_records (`timestamp`,`from`,`to`,diamonds) values(?,?,?,?)", time.Now().Unix(), uid, 0, diamonds)
			if err != nil {
				base.LogError("failed to update db. error:", err)
				return nil, err
			}
		}

		// var newDiamonds uint32
		// err = tx.QueryRow("select diamonds from users where uid=?", uid).Scan(&newDiamonds)
		// if err != nil {
		// 	base.LogError("failed to query diamonds. error:", err)
		// 	return nil, err
		// }

		// usersDiamonds[ndx] = newDiamonds
	}

	if !allUsersHaveEnoughDiamonds {
		return nomoneyUids, ErrorNotEnoughDiamonds
	}

	success = true
	tx.Commit()

	return nil, err
}

func (obj *mysqlDB) PayDiamonds(from, to, diamonds uint32, diamondsFee uint32, keep uint32) (uint32, error) {
	success := false
	tx, err := obj.db.Begin()
	if err != nil {
		base.LogError("PayDiamonds: failed to start a transaction, error:", err)
		return 0, err
	}
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	result, err := tx.Exec("update users set diamonds=diamonds-? where uid=? and diamonds >= ?", diamonds+diamondsFee, from, diamonds+keep+diamondsFee)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows != 1 {
		return 0, ErrorNotEnoughDiamonds
	}

	_, err = tx.Exec("update users set diamonds=diamonds+? where uid=?", diamonds, to)
	if err != nil {
		base.LogError("failed to update db. error:", err)
		return 0, err
	}

	_, err = tx.Exec("insert into diamond_records (`timestamp`,`from`,`to`,diamonds) values(?,?,?,?)", time.Now().Unix(), from, to, diamonds)
	if err != nil {
		base.LogError("failed to update db. error:", err)
		return 0, err
	}

	var newDiamonds uint32
	err = tx.QueryRow("select diamonds from users where uid=?", from).Scan(&newDiamonds)
	if err != nil {
		base.LogError("failed to query diamonds. error:", err)
		return 0, err
	}

	success = true
	tx.Commit()

	return newDiamonds, err
}

func (obj *mysqlDB) createRoom(num int, name string, uid, hands, minBet, maxBet, creditPoints uint32, isShare bool, createTime int64) (uint32, error) {
	result, err := obj.db.Exec("insert into room_records (name,number,owner_uid,hands,played_hands,is_share,min_bet,max_bet,credit_points,create_time,close_time,closed) values(?,?,?,?,0,?,?,?,?,?,0,false)",
		name, num, uid, hands, isShare, minBet, maxBet, creditPoints, createTime)
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

func (obj *mysqlDB) getRoomsCountCreatedByUser(uid uint32) (count uint32, err error) {
	err = obj.db.QueryRow("select count(*) from room_records where owner_uid=? and closed=false", uid).Scan(&count)
	return
}

func (obj *mysqlDB) GetRoomID(num uint32) (roomID uint32, err error) {
	err = obj.db.QueryRow("select room_id from room_records where number=? and closed=false", num).Scan(&roomID)
	return
}

func (obj *mysqlDB) loadRoom(num uint32) (name string, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints uint32, isShare bool, createTime int64, err error) {
	err = obj.db.QueryRow("select room_id,name,owner_uid,hands,played_hands,is_share,min_bet,max_bet,credit_points,create_time from room_records where number=? and closed=false",
		num).Scan(&roomID, &name, &uid, &hands, &playedHands, &isShare, &minBet, &maxBet, &creditPoints, &createTime)
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

func (obj *mysqlDB) SaveRoundResult(roomID, round uint32, uids []uint32, result string) error {
	success := false
	tx, err := obj.db.Begin()
	if err != nil {
		base.LogError("SaveRoundResult: failed to start a transaction, error:", err)
		return err
	}
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	r, err := tx.Exec("insert into game_records (roomid,round,result,timestamp) values(?,?,?,UNIX_TIMESTAMP(NOW()))",
		roomID, round, result)
	if err != nil {
		base.LogError("SaveRoundResult: failed to insert into game_records, error:", err)
		return err
	}
	roundID, err := r.LastInsertId()
	if err != nil {
		base.LogError("SaveRoundResult: failed to get last insert id, error:", err)
		return err
	}
	sql := "insert into round_players (round_id,uid) values "
	i := 0
	for _, uid := range uids {
		if i == 0 {
			sql = sql + fmt.Sprintf("(%d,%d)", roundID, uid)
		} else {
			sql = sql + fmt.Sprintf(",(%d,%d)", roundID, uid)
		}
		i++
	}
	//base.LogInfo(sql)
	_, err = tx.Exec(sql)
	if err != nil {
		base.LogError("SaveRoundResult: failed to insert into round_players, error:", err)
		return err
	}

	_, err = tx.Exec("update room_records set played_hands=? where room_id=?",
		round+1, roomID)
	if err != nil {
		base.LogError("SaveRoundResult: failed to update room_records, error:", err)
		return err
	}
	success = true
	tx.Commit()

	return nil
}

func (obj *mysqlDB) GetRoundResult(roomID, round uint32) (result string, err error) {
	err = obj.db.QueryRow("select result from game_records where roomid=? and round=?",
		roomID, round).Scan(&result)
	return
}

func (obj *mysqlDB) GetUserProfile(uid uint32) (name, signture, avatar string, diamonds uint32, status uint32, err error) {
	err = obj.db.QueryRow("select name,signture,avatar,diamonds,status from users where uid=?", uid).Scan(&name, &signture, &avatar, &diamonds, &status)
	if err != nil {
		base.LogError("GetUserProfile(", uid, "), error:", err)
	}
	return
}

func (obj *mysqlDB) ExistUser(uid uint32) (bool, error) {
	err := obj.db.QueryRow("select uid from users where uid=?", uid).Scan(&uid)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		base.LogError("ExistUser(", uid, "), error:", err)
		return false, err
	}
	return true, nil
}

func (obj *mysqlDB) GetAllOpenRooms() ([]*roomCreateTime, error) {
	rows, err := obj.db.Query("select room_id,create_time from room_records where closed=false order by create_time")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]*roomCreateTime, 0, 256)
	for rows.Next() {
		var roomid uint32
		var createTime int64
		if err := rows.Scan(&roomid, &createTime); err != nil {
			return nil, err
		}
		r = append(r, &roomCreateTime{roomid: roomid, createTime: createTime})
	}
	return r, nil
}

func (obj *mysqlDB) CloseRoom(roomID uint32, closeTime int64) error {
	_, err := obj.db.Exec("update room_records set closed=true,close_time=? where room_id=?", closeTime, roomID)
	if err != nil {
		return err
	}
	return nil
}

func (obj *mysqlDB) GetRoomsListJoined(uid uint32) ([]*msg.ListRoomItem, error) {
	rows, err := obj.db.Query("SELECT room_id, NAME, number, owner_uid,hands,played_hands FROM room_records WHERE room_id IN (SELECT b.`room_id` FROM scoreboard AS a, room_records AS b WHERE a.`uid`=? AND a.`roomid`=b.`room_id`  AND b.`closed`=FALSE) OR owner_uid=? AND closed=FALSE order by create_time", uid, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]*msg.ListRoomItem, 0, 10)
	for rows.Next() {
		var roomid uint32
		var name string
		var number int
		var ownerUID uint32
		var hands uint32
		var playedHands uint32
		if err := rows.Scan(&roomid, &name, &number, &ownerUID, &hands, &playedHands); err != nil {
			return nil, err
		}
		r = append(r, &msg.ListRoomItem{
			RoomId:      roomid,
			RoomName:    name,
			RoomNumber:  roomNumberGenerator.decode(number),
			OwnerUid:    ownerUID,
			PlayedHands: playedHands,
			Hands:       hands,
		})
	}
	return r, nil
}

func (obj *mysqlDB) GetDiamondRecords(uid uint32, begin, end int64) ([]*msg.DiamondsRecordsItem, error) {
	rows, err := obj.db.Query("SELECT `timestamp`,`from`,`to`,diamonds FROM diamond_records WHERE `timestamp`>=? and `timestamp`<? and ((`from`=? and `to`<>0) or (`from`<>0 and `to`=?))  order by `timestamp` desc", begin, end, uid, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]*msg.DiamondsRecordsItem, 0, 16)
	for rows.Next() {
		var time uint32
		var from uint32
		var to uint32
		var diamonds int32
		if err := rows.Scan(&time, &from, &to, &diamonds); err != nil {
			return nil, err
		}

		if to == uid {
			r = append(r, &msg.DiamondsRecordsItem{
				Time:     time,
				Uid:      from,
				Diamonds: diamonds,
			})
		} else {
			r = append(r, &msg.DiamondsRecordsItem{
				Time:     time,
				Uid:      to,
				Diamonds: -diamonds,
			})
		}
	}
	return r, nil
}

func (obj *mysqlDB) GetUsersNameAvatar(uids []uint32) ([]*msg.UserNameAvatar, error) {
	sql := "SELECT uid,name,avatar FROM users WHERE "
	l := len(uids)
	if l == 0 {
		return []*msg.UserNameAvatar{}, nil
	} else if l == 1 {
		sql += fmt.Sprintf("uid=%d", uids[0])
	} else {
		sql += fmt.Sprintf("uid in (%d", uids[0])
		for i := 1; i < l; i++ {
			sql += fmt.Sprintf(",%d", uids[i])
		}
		sql += ")"
	}
	rows, err := obj.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]*msg.UserNameAvatar, 0, 16)
	for rows.Next() {
		var uid uint32
		var name, avatar string
		if err := rows.Scan(&uid, &name, &avatar); err != nil {
			return nil, err
		}

		r = append(r, &msg.UserNameAvatar{
			Uid:    uid,
			Name:   name,
			Avatar: avatar,
		})

	}
	return r, nil
}

func (obj *mysqlDB) GetCareerWinLoseData(uid uint32, start, end int64) ([]int, error) {
	rows, err := obj.db.Query("SELECT b.`score` FROM room_records AS a, scoreboard AS b WHERE a.`room_id`=b.`roomid` AND a.create_time >= ? AND a.create_time < ? AND b.uid=?", start, end, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]int, 0, 8)
	for rows.Next() {
		var score int
		if err := rows.Scan(&score); err != nil {
			return nil, err
		}
		r = append(r, score)
	}
	return r, nil
}

func (obj *mysqlDB) GetCareerRooms(uid uint32, start, end int64, pos uint32, num uint32) ([]*msg.CareerRoomRecord, error) {
	rows, err := obj.db.Query("SELECT a.`room_id`,a.`name`,a.`hands`,a.`played_hands`,a.`is_share`,a.`min_bet`,a.`max_bet`,a.`create_time`,a.`close_time` FROM room_records AS a, scoreboard AS b WHERE a.`room_id`=b.`roomid` AND a.create_time >= ? AND a.create_time < ? AND b.uid=? order by a.create_time desc limit ?,?",
		start, end, uid, pos, num)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	r := make([]*msg.CareerRoomRecord, 0, 8)
	for rows.Next() {
		var roomID, hands, playedHands, minBet, maxBet, createTime, closeTime uint32
		var name string
		var isShare bool
		if err := rows.Scan(&roomID, &name, &hands, &playedHands, &isShare, &minBet, &maxBet, &createTime, &closeTime); err != nil {
			return nil, err
		}
		r = append(r, &msg.CareerRoomRecord{
			RoomId:      roomID,
			Name:        name,
			Hands:       hands,
			PlayedHands: playedHands,
			IsShare:     isShare,
			MinBet:      minBet,
			MaxBet:      maxBet,
			BeginTime:   createTime,
			EndTime:     closeTime,
		})
	}
	return r, nil
}

func (obj *mysqlDB) SaveOnlineStatistic(date *time.Time, online, playing, rooms int32) error {
	_, err := obj.db.Exec(
		"REPLACE INTO online_statistics (`timestamp`,max_online_users,max_playing_users,max_playing_rooms) VALUES(?,?,?,?)",
		date.Unix(), online, playing, rooms)

	return err
}

func (obj *mysqlDB) getNewUsers(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE regtime >= ? AND regtime < ?",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getNewUsers. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getActiveUsers(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT COUNT(*) FROM (SELECT COUNT(*) c,uid FROM round_players WHERE round_id IN (SELECT round_id FROM game_records WHERE `timestamp`>=? AND `timestamp`<?) GROUP BY uid) a WHERE a.c>=5",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getActiveUsers. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getPlayedUsers(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT COUNT(*) FROM (SELECT COUNT(*) c,uid FROM round_players WHERE round_id IN (SELECT round_id FROM game_records WHERE `timestamp`>=? AND `timestamp`<?) GROUP BY uid) a",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getActiveUsers. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getUsersConsumedDiamonds(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT COUNT(DISTINCT(`from`)) FROM diamond_records WHERE `timestamp`>=? AND `timestamp`<? AND `to`=0",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getUsersConsumedDiamonds. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getRoomsCreated(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT COUNT(*) FROM room_records WHERE `create_time`>=? AND `create_time`<?",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getRoomsCreated. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getRoundsPlayed(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT COUNT(*) FROM game_records WHERE `timestamp`>=? AND `timestamp`<?",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getRoundsPlayed. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getDiamondsConsumed(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT IFNULL(SUM(diamonds),0) FROM diamond_records WHERE `timestamp`>=? AND `timestamp`<? AND `to`=0",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getDiamondsConsumed. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) getDiamondsProvided(begin, end time.Time) uint32 {
	var count uint32
	err := obj.db.QueryRow(
		"SELECT IFNULL(SUM(diamonds),0) FROM diamond_records WHERE `timestamp`>=? AND `timestamp`<? AND `from`=0",
		begin.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		base.LogError("Failed to getDiamondsProvided. begin:", begin, ", end:", end, ", error:", err)
	}
	return count
}

func (obj *mysqlDB) saveDateStatistic(newUsers, activeUsers, playedUsers, consumedUsers, roomsCreated, roundsPlayed, diamondsConsumed, diamondsProvided uint32, date time.Time) {
	_, err := obj.db.Exec(
		`INSERT INTO game_statistics (timestamp, new_users,active_users,play_users,cost_diamonds_users,create_rooms,played_rounds,cost_diamonds,offer_diamonds) VALUES(?,?,?,?,?,?,?,?,?) 
		ON DUPLICATE KEY UPDATE new_users=?,active_users=?,play_users=?,cost_diamonds_users=?,create_rooms=?,played_rounds=?,cost_diamonds=?,offer_diamonds=?`,
		date.Unix(), newUsers, activeUsers, playedUsers, consumedUsers, roomsCreated, roundsPlayed, diamondsConsumed, diamondsProvided,
		newUsers, activeUsers, playedUsers, consumedUsers, roomsCreated, roundsPlayed, diamondsConsumed, diamondsProvided)
	if err != nil {
		base.LogError("Failed to saveDateStatistic. error: ", err)
	}
}

func (obj *mysqlDB) ban(uid uint32) error {
	result, err := obj.db.Exec("update users set status=1 where uid=?", uid)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if rows != 1 {
		return fmt.Errorf("the uid:%d has been banned already or not exist", uid)
	}
	return err
}

func (obj *mysqlDB) unban(uid uint32) error {
	result, err := obj.db.Exec("update users set status=0 where uid=?", uid)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if rows != 1 {
		return fmt.Errorf("the uid:%d has not been banned yet or not exist", uid)
	}
	return err
}
