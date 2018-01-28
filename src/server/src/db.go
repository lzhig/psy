package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
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
	obj.db.Close()
}

// 查找fbid，如果存在更新name
func (obj *mysqlDB) getUIDFacebook(fbID, name string) int {
	var uid int
	err := obj.db.QueryRow("select uid from facebook_users where fbid=?", fbID).Scan(&uid)
	switch {
	case err == sql.ErrNoRows:
		return 0
	case err != nil:
		logError("[mysql][getUIDFacebook] query uid, error:", err, "fbID:", fbID, ", name:", name)
		return 0
	default:
		_, err := obj.db.Exec("update facebook_users set name=? where fbid=?;", name, fbID)
		if err != nil {
			logWarn("[mysql][getUIDFacebook] update name, error:", err, "fbID:", fbID, ", name:", name)
		}

		return uid
	}
}

func (obj *mysqlDB) createFacebookUser(fbID, name string) int {
	stmt, err := obj.db.Prepare("CALL create_facebook_user(?,?,@uid);")
	if err != nil {
		logError("[mysql][createFacebookUser] failed to prepare sp, error:", err, "fbID:", fbID, ", name:", name)
		return 0
	}
	defer stmt.Close()

	_, err = stmt.Exec(fbID, name)
	if err != nil {
		logError("[mysql][createFacebookUser] failed to exec, error:", err, "fbID:", fbID, ", name:", name)
		return 0
	}

	stmt1, err := obj.db.Prepare("select @uid as uid")
	if err != nil {
		logError("[mysql][createFacebookUser] failed to prepare select, error:", err, "fbID:", fbID, ", name:", name)
		return 0
	}
	defer stmt1.Close()

	var uid int
	err = stmt1.QueryRow().Scan(&uid)
	if err != nil {
		logError("[mysql][createFacebookUser] failed to scan, error:", err, "fbID:", fbID, ", name:", name)
		return 0
	}

	return uid
}
