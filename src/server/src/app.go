/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 10:13:22
 * @modify date 2018-01-19 10:13:22
 * @desc [description]
 */

package main

import (
	"fmt"
	"runtime"

	"github.com/lzhig/rapidgo/base"
)

var db = &mysqlDB{}
var userManager = &UserManager{}
var loginService = &LoginService{}
var debug func(a ...interface{}) (int, error)
var roomManager = &RoomManager{}
var roomNumberGenerator = &RoomNumberGenerator{}
var dealer = &Dealer{}
var diamondsCenter = &DiamondsCenter{}
var careerCenter = &CareerCenter{}
var versionService = &VersionService{}

// App type
type App struct {
	base.App

	config  *Config
	network *NetworkEngine

	gm *gameManager
}

// Init function
func (obj *App) Init() error {
	obj.config = &Config{}
	if err := obj.config.Load("config.json"); err != nil {
		return err
	}

	if obj.config.Debug {
		debug = fmt.Println
	} else {
		debug = func(a ...interface{}) (int, error) { return 0, nil }
	}

	base.LogInfo("version:", obj.config.Version)

	runtime.GOMAXPROCS(obj.config.CPUNum)

	obj.App.Init()

	versionService.Start(obj.config.VersionService.Addr, obj.config.VersionService.File)

	mysqlCfg := obj.config.Mysql
	if err := db.open(mysqlCfg.Addr, mysqlCfg.Username, mysqlCfg.Password, mysqlCfg.Db, mysqlCfg.MaxConns); err != nil {
		return err
	}

	// init services
	usedNum, err := db.getRoomNumberUsed()
	if err != nil {
		return err
	}
	if err := roomNumberGenerator.init(4, []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}, usedNum); err != nil {
		return err
	}
	//debug(roomNumberGenerator.encode("5428"))

	dealer.init()
	if err := roomManager.init(); err != nil {
		return err
	}
	careerCenter.Init()
	diamondsCenter.init()
	userManager.init()
	loginService.init()

	obj.network = &NetworkEngine{}
	obj.network.init()

	// gm
	obj.gm = &gameManager{}

	base.LogInfo("init done.")

	// t, err := time.ParseInLocation("2006-1-2 15:4:5", "0001-1-1 0:0:0", time.Local)
	// fmt.Println(t.Unix(), err)
	// fmt.Println(t.Date())
	// y, m, d := time.Now().AddDate(0, 0, 0).Date()
	// t = t.AddDate(y-1, int(m)-1, d-1)
	// fmt.Println(t.Format("2006-1-2 15:4:5"))
	// fmt.Println(t.Unix())

	// a := `{"error":{"message":"Unsupported get request. Object with ID '1637239499921854' does not exist, cannot be loaded due to missing permissions, or does not support this operation. Please read the Graph API documentation at https:\/\/developers.facebook.com\/docs\/graph-api","type":"GraphMethodException","code":100,"error_subcode":33,"fbtrace_id":"BlfdHAICYcb"}}`
	// b := checkResult{}
	// err = json.Unmarshal([]byte(a), &b)
	// fmt.Println(err, b)

	// for i := uint32(0); i < 12; i++ {
	// 	buf, err := db.GetRoundResult(7, i)
	// 	if err != nil {
	// 		fmt.Println("i:", i, "error:", err)
	// 		continue
	// 	}
	// 	r := &msg.DBResults{}
	// 	err = proto.Unmarshal([]byte(buf), r)
	// 	if err != nil {
	// 		fmt.Println("i:", i, "error:", err)
	// 		continue
	// 	}
	// 	fmt.Println(r)
	// }

	return nil
}

// Start function
func (obj *App) Start() {
	obj.network.Start(obj.config.Server.Addr, obj.config.Server.MaxUsers)
	obj.gm.Start(obj.config.Server.Gm)

	obj.App.Start()
}

// End function
func (obj *App) End() {
	db.close()
}
