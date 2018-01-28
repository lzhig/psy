/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 11:24:28
 * @modify date 2018-01-19 11:24:28
 * @desc [description]
 */

package main

// UserManager type
type UserManager struct {
	users map[int]*User
}

func (obj *UserManager) init() {
	obj.users = make(map[int]*User)
}

func (obj *UserManager) fbUserExists(fbID, name string) int {
	return db.getUIDFacebook(fbID, name)
}

func (obj *UserManager) fbUserCreate(fbID, name string) int {
	uid := db.createFacebookUser(fbID, name)
	if uid != 0 {
		obj.fbUserLoad(uid)
	}
	return uid
}

func (obj *UserManager) fbUserLoad(uid int) {

}

// LoginFacebookUser function
func (obj *UserManager) LoginFacebookUser(id, token, name string) {

	// request facebook api to check id and token

	// if failed, return error

	// if new user, save data to db

	// if old user, load data from db
}

// The User type represents a player
type User struct {
	uid      uint32
	name     string // 名字
	diamonds uint32 // 钻石
}
