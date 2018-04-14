package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"github.com/lzhig/rapidgo/base"
)

type gameManager struct {
}

func (obj *gameManager) Start(addr string) {
	go func() {
		defer base.LogPanic()

		http.HandleFunc("/user", userCountHandler)
		http.HandleFunc("/GmOperation", gmOperationHandler)
		http.HandleFunc("/exit", gmExit)
		http.HandleFunc("/updateVersion", gmUpdateVersion)
		http.ListenAndServe(addr, nil)
	}()
}

func userCountHandler(w http.ResponseWriter, r *http.Request) {
	// s := fmt.Sprintf("user count: %d", GetApp().user.GetUserCount())
	// w.Write([]byte(s))
}

func gmExit(w http.ResponseWriter, r *http.Request) {
	gApp.Exit()
	io.WriteString(w, "exit.\r\n")
}

func gmUpdateVersion(w http.ResponseWriter, r *http.Request) {
	versionService.UpdateVersionInfo()
	io.WriteString(w, "updated version info.\r\n")
}

func gmOperationHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorln(err)
		return
	}
	r.Body.Close()
	glog.Infoln(string(data))

	var req map[string]interface{}
	err = json.Unmarshal(data, &req)
	if err != nil {
		glog.Errorln(err)
		return
	}

	switch req["oper_type"] {
	case "GmNotice":
		// notice := req["Notice"].(map[string]interface{})
		// starttime := int64(notice["start_time"].(float64))
		// endtime := int64(notice["end_time"].(float64))
		// content, err := base64.StdEncoding.DecodeString(notice["content"].(string))
		// if err != nil {
		// 	io.WriteString(w, "Failed to decode")
		// 	return
		// }
		// glog.Infoln(string(content))
		//app.announcement.Set(string(content), time.Unix(starttime, 0), time.Unix(endtime, 0))

	case "StopServer":
		gApp.Exit()
		//GetApp().StopServer()
		io.WriteString(w, "Stopped.\r\n")
		//GetApp().Quit()

	case "StartServer":

	case "StartAI":
		//app.room.StartAI()

	case "StopAI":
		//app.room.StopAI()
	}
}
