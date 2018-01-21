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
	uid       uint32
	username  string // 用户名
	aliasname string // 标注名
	diamonds  uint32 // 钻石
}
