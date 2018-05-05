package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/lzhig/rapidgo/base"
)

type gameManager struct {
}

func (obj *gameManager) Start(addr string) {
	go func() {
		defer base.LogPanic()

		http.HandleFunc("/user", userCountHandler)
		//http.HandleFunc("/GmOperation", gmOperationHandler)
		http.HandleFunc("/exit", gmExit)
		http.HandleFunc("/updateVersion", gmUpdateVersion)
		http.HandleFunc("/flushLog", func(w http.ResponseWriter, r *http.Request) {
			base.LogInfo(roomManager.roomsNumber)
			base.LogFlush()
		})
		http.HandleFunc("/kick", kick)
		http.HandleFunc("/ban", ban)
		http.HandleFunc("/unban", unban)
		http.HandleFunc("/updateNotice", updateNotice)
		http.ListenAndServe(addr, nil)
	}()
}

func updateNotice(w http.ResponseWriter, r *http.Request) {
	if err := userManager.NotifyAllUsers(); err != nil {
		fmt.Fprintln(w, "Failed to notify notices. error:", err)
		return
	}
	fmt.Fprintln(w, "Done!")
}

func kick(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	uids := r.Form["uid"]
	if len(uids) == 0 {
		fmt.Fprintln(w, "Please specify the uid you want to kick.")
		return
	}

	for _, str := range uids {
		uid, err := strconv.Atoi(str)
		if err != nil || uid <= 0 {
			fmt.Fprintln(w, "uid:", str, " is invalid uid.")
			return
		}

		if err := userManager.KickUser(uint32(uid)); err != nil {
			fmt.Fprintln(w, "Failed to kick the uid:", uid, ". error:", err)
			return
		}
		fmt.Fprintln(w, "uid:", uid, " kicked.")
	}
}

func ban(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	uids := r.Form["uid"]
	if len(uids) == 0 {
		fmt.Fprintln(w, "Please specify the uid you want to ban.")
		return
	}

	for _, str := range uids {
		uid, err := strconv.Atoi(str)
		if err != nil || uid <= 0 {
			fmt.Fprintln(w, "uid:", str, " is invalid uid.")
			return
		}
		if err := db.ban(uint32(uid)); err != nil {
			fmt.Fprintln(w, "Failed to ban the uid:", uid, ". error:", err)
			return
		}
		fmt.Fprintln(w, "uid:", uid, " banned.")
	}
}

func unban(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	uids := r.Form["uid"]
	if len(uids) == 0 {
		fmt.Fprintln(w, "Please specify the uid you want to unban.")
		return
	}

	for _, str := range uids {
		uid, err := strconv.Atoi(str)
		if err != nil || uid <= 0 {
			fmt.Fprintln(w, "uid:", str, " is invalid uid.")
			return
		}
		if err := db.unban(uint32(uid)); err != nil {
			fmt.Fprintln(w, "Failed to unban the uid:", uid, ". error:", err)
			return
		}
		fmt.Fprintln(w, "uid:", uid, " unbanned.")
	}
}

func userCountHandler(w http.ResponseWriter, r *http.Request) {
	// s := fmt.Sprintf("user count: %d", GetApp().user.GetUserCount())
	// w.Write([]byte(s))
}

func gmExit(w http.ResponseWriter, r *http.Request) {
	// 1. 禁止用户进行登录
	loginService.EnableLogin(false)

	// 2. 将不在房间或者房间没有开始游戏的用户踢离，房间关闭
	userManager.KickUsersNotInRoom()

	// 3. 等待已经开始游戏的一局结束，再将房间内的用户踢离，房间关闭
	roomManager.Enable(false)

	base.LogFlush()
	gApp.Exit()
	fmt.Fprintln(w, "exit.")
}

func gmUpdateVersion(w http.ResponseWriter, r *http.Request) {
	versionService.UpdateVersionInfo()
	fmt.Fprintln(w, "updated version info.")
}

// func gmOperationHandler(w http.ResponseWriter, r *http.Request) {
// 	data, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		glog.Errorln(err)
// 		return
// 	}
// 	r.Body.Close()
// 	glog.Infoln(string(data))

// 	var req map[string]interface{}
// 	err = json.Unmarshal(data, &req)
// 	if err != nil {
// 		glog.Errorln(err)
// 		return
// 	}

// 	switch req["oper_type"] {
// 	case "GmNotice":
// 		// notice := req["Notice"].(map[string]interface{})
// 		// starttime := int64(notice["start_time"].(float64))
// 		// endtime := int64(notice["end_time"].(float64))
// 		// content, err := base64.StdEncoding.DecodeString(notice["content"].(string))
// 		// if err != nil {
// 		// 	io.WriteString(w, "Failed to decode")
// 		// 	return
// 		// }
// 		// glog.Infoln(string(content))
// 		//app.announcement.Set(string(content), time.Unix(starttime, 0), time.Unix(endtime, 0))

// 	case "StopServer":
// 		gApp.Exit()
// 		//GetApp().StopServer()
// 		fmt.Fprintln(w, "Stopped.")
// 		//GetApp().Quit()

// 	case "StartServer":

// 	case "StartAI":
// 		//app.room.StartAI()

// 	case "StopAI":
// 		//app.room.StopAI()
// 	}
// }
