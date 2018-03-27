package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// FacebookUserCheckCallback 回调函数
type FacebookUserCheckCallback func(*FacebookUser, bool, string)

// FacebookUser facebook用户
type FacebookUser struct {
	Fbid  string
	Token string
	Name  string

	callback FacebookUserCheckCallback
}

// GetAvatarURL 获取头像url
func (obj *FacebookUser) GetAvatarURL() string {
	return fmt.Sprintf("http://graph.facebook.com/%s/picture?type=%s", obj.Fbid, gApp.config.User.FacebookAvatarType)
}

// GetUID 获取uid
func (obj *FacebookUser) GetUID() (uint32, error) {
	return db.getUIDFacebook(obj.Fbid)
}

func (obj *FacebookUser) SaveToDB(uid uint32) error {
	return db.AddFacebookUser(obj.Fbid, uid)
}

func (obj *FacebookUser) GetPlatformID() uint32 { return 0 }
func (obj *FacebookUser) GetName() string       { return obj.Name }

// FacebookUserCheck 验证facebook用户合法性
type FacebookUserCheck struct {
	list chan *FacebookUser
}

// Init 初始化
func (obj *FacebookUserCheck) Init() {
	obj.list = make(chan *FacebookUser)
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

// Check 验证
func (obj *FacebookUserCheck) Check(user *FacebookUser, callback FacebookUserCheckCallback) bool {
	user.callback = callback
	select {
	case obj.list <- user:
		return true
	default:
		return false
	}
}

type checkError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubCode int    `json:"error_subcode"`
	FbTraceID    string `json:"fbtrace_id"`
}

type checkResult struct {
	Name  string      `json:"name"`
	ID    string      `json:"id"`
	Error *checkError `json:"error"`
}

func (obj *FacebookUserCheck) loop(ctx context.Context) {
	defer debug("exit FacebookUserCheck goroutine")
	for {
		select {
		case <-ctx.Done():
			return

		case user := <-obj.list:
			go obj.workroutine(user)
		}
	}
}

func (obj *FacebookUserCheck) workroutine(user *FacebookUser) {
	// tr := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	// client := &http.Client{Transport: tr,
	// 	Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://graph.facebook.com/%s?access_token=%s", user.Fbid, user.Token)
	rsp, err := http.Get(url)

	if err != nil {
		fmt.Println("failed to access.")
		user.callback(user, false, err.Error())
		return
	}
	defer rsp.Body.Close()

	fmt.Println("access successful.")

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		user.callback(user, false, err.Error())
		return
	}

	result := &checkResult{}
	err = json.Unmarshal(body, result)
	if err != nil {
		user.callback(user, false, err.Error())
		return
	}
	if result.Error != nil {
		user.callback(user, false, result.Error.Message)
		return
	}
	user.Name = result.Name
	user.callback(user, true, "")
}
