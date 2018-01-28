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
	loginService.init()

	obj.network = &NetworkEngine{}
	obj.network.init()

	// gm
	obj.gm = &gameManager{}

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
