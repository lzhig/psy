package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/lzhig/rapidgo/base"
)

// DateStatistic 统计
type DateStatistic struct {
}

// Init 初始化
func (obj *DateStatistic) Init() error {
	now := time.Now()
	today := base.GetTodayZeroClockTime(&now)
	yesterday := today.AddDate(0, 0, -1)
	timePerDay, err := time.Parse("15:4:5", gApp.config.Statistic.DateSheet)
	if err != nil {
		return err
	}
	h, m, s := timePerDay.Clock()
	s = h*3600 + m*60 + s
	t := today.Add(time.Duration(s) * time.Second)
	d := t.Sub(now)
	if d <= 0 {
		obj.generateDateSheet(yesterday)
	}

	nextTickTime := t.AddDate(0, 0, 1)

	var f func()
	f = func() {
		time.AfterFunc(time.Second*5, f)
		yesterday = yesterday.Add(time.Hour * 24)
		obj.generateDateSheet(yesterday)
	}
	time.AfterFunc(nextTickTime.Sub(now), f)

	return nil
}

// 计算前一天的统计数据
func (obj *DateStatistic) generateDateSheet(begin time.Time) {
	base.LogInfo("DataStatistic:", begin, ", timestamp:", begin.Unix())
	end := begin.AddDate(0, 0, 1)
	newUsers := db.getNewUsers(begin, end)
	activeUsers := db.getActiveUsers(begin, end)
	playedUsers := db.getPlayedUsers(begin, end)
	consumedUsers := db.getUsersConsumedDiamonds(begin, end)
	roomsCreated := db.getRoomsCreated(begin, end)
	roundsPlayed := db.getRoundsPlayed(begin, end)
	diamondsConsumed := db.getDiamondsConsumed(begin, end)
	diamondsProvided := db.getDiamondsProvided(begin, end)

	base.LogInfo("date:", begin.String())
	base.LogInfo("newUsers:", newUsers)
	base.LogInfo("activeUsers:", activeUsers)
	base.LogInfo("playedUsers:", playedUsers)
	base.LogInfo("consumedUsers:", consumedUsers)
	base.LogInfo("roomsCreated:", roomsCreated)
	base.LogInfo("roundsPlayed:", roundsPlayed)
	base.LogInfo("diamondsConsumed:", diamondsConsumed)
	base.LogInfo("diamondsProvided:", diamondsProvided)

	db.saveDateStatistic(newUsers, activeUsers, playedUsers, consumedUsers, roomsCreated, roundsPlayed, diamondsConsumed, diamondsProvided, begin)
}

type maxStatistic struct {
	Max int32 `json:"max"`
	now int32
	c   chan int32
}

func (obj *maxStatistic) reset() {
	obj.Max = 0
	obj.now = 0
}

// change 变化, increase为true增加，否则减少
func (obj *maxStatistic) change(increase bool) {
	if increase {
		obj.c <- 1
	} else {
		obj.c <- -1
	}
}

// update
func (obj *maxStatistic) update(num int32) {
	if obj.now < -num {
		base.LogError("Invalid value")
		obj.now = 0
	} else {
		obj.now += num
	}

	if obj.Max < obj.now {
		obj.Max = obj.now
	}
}

// OnlineStatistic 在线统计，取当日最高值
type OnlineStatistic struct {
	Online  *maxStatistic `json:"online"`  // 最高在线人数
	Playing *maxStatistic `json:"playing"` // 最高在玩人数
	Rooms   *maxStatistic `json:"rooms"`   // 最高在玩房间
	Date    string        `json:"date"`    // 当前时间
}

// Init 初始化
func (obj *OnlineStatistic) Init() error {
	obj.Online = &maxStatistic{c: make(chan int32, 16)}
	obj.Playing = &maxStatistic{c: make(chan int32, 16)}
	obj.Rooms = &maxStatistic{c: make(chan int32, 16)}

	if err := obj.load(); err != nil {
		base.LogWarn("Failed to load statistic file. error:", err)
	}

	go obj.loop()
	return nil
}

// Close 关闭
func (obj *OnlineStatistic) Close() {
	if err := obj.save(); err != nil {
		base.LogError("Failed to save statistic file. error:", err)
		return
	}
}

// load 服务器开始时，读取存档
func (obj *OnlineStatistic) load() error {
	f, err := os.Open(gApp.config.Statistic.OnlineFilename)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	// 判断当前时间是否是今天，否则清零
	t, err := time.ParseInLocation("2006-1-2 15:4:5", obj.Date, time.Local)
	if err != nil {
		return err
	}
	now := time.Now()

	if base.GetTodayZeroClockTime(&t).Unix() != base.GetTodayZeroClockTime(&now).Unix() {
		obj.Online.reset()
		obj.Playing.reset()
		obj.Rooms.reset()
	}
	return nil
}

// Save 服务器关闭时，进行保存
func (obj *OnlineStatistic) save() error {
	now := time.Now()
	today := base.GetTodayZeroClockTime(&now)
	obj.generateOnlineSheet(&today)
	obj.Date = time.Now().Format("2006-1-2 15:4:5")
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(gApp.config.Statistic.OnlineFilename, data, 0666)
}

// OnlinePersonsChange 在线人数变化, online为True,玩家登录, false为玩家离开
func (obj *OnlineStatistic) OnlinePersonsChange(online bool) {
	obj.Online.change(online)
}

// PlayingPersonsChange 在玩人数变化, playing为True,玩家进入房间, false为玩家离开房间
func (obj *OnlineStatistic) PlayingPersonsChange(playing bool) {
	obj.Playing.change(playing)
}

// RoomsChange 在线房间变化, create为true时，房间创建，否则为房间销毁
func (obj *OnlineStatistic) RoomsChange(create bool) {
	obj.Rooms.change(create)
}

func (obj *OnlineStatistic) loop() {
	now := time.Now()
	today := base.GetTodayZeroClockTime(&now)
	var ticker *time.Ticker
	if now.Unix() == today.Unix() {
		ticker = time.NewTicker(time.Hour * 24)
		obj.generateOnlineSheet(&today)
	} else {
		ticker = time.NewTicker(today.AddDate(0, 0, 1).Sub(now))
	}
	defer ticker.Stop()

	for {
		select {
		case num := <-obj.Online.c:
			obj.Online.update(num)

		case num := <-obj.Playing.c:
			obj.Playing.update(num)

		case num := <-obj.Rooms.c:
			obj.Rooms.update(num)

		case <-ticker.C:
			ticker = time.NewTicker(time.Hour * 24)
			obj.generateOnlineSheet(&today)
			today = today.AddDate(0, 0, 1)
		}
	}
}

func (obj *OnlineStatistic) generateOnlineSheet(date *time.Time) {
	db.SaveOnlineStatistic(date, obj.Online.Max, obj.Playing.Max, obj.Rooms.Max)
}
