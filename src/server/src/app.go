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

	logInfo("version:", obj.config.Version)

	runtime.GOMAXPROCS(obj.config.CPUNum)

	obj.App.Init()

	mysqlCfg := obj.config.Mysql
	if err := db.open(mysqlCfg.Addr, mysqlCfg.Username, mysqlCfg.Password, mysqlCfg.Db); err != nil {
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
	roomManager.init()
	userManager.init()
	loginService.init()

	obj.network = &NetworkEngine{}
	obj.network.init()

	// gm
	obj.gm = &gameManager{}

	logInfo("init done.")

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
