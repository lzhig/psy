/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 10:13:22
 * @modify date 2018-01-19 10:13:22
 * @desc [description]
 */

package main

import (
	"runtime"

	"github.com/lzhig/rapidgo/base"
)

// App type
type App struct {
	base.App

	config  *Config
	network *NetworkEngine
}

// Init function
func (obj *App) Init() error {
	config := &Config{}
	if err := config.Load("config.json"); err != nil {
		return err
	}

	runtime.GOMAXPROCS(config.CPUNum)

	obj.App.Init()

	// init services
	obj.network = &NetworkEngine{app: obj}

	return nil
}

// Start function
func (obj *App) Start() {
	obj.network.Start(obj.config.Server.Addr, obj.config.Server.MaxUsers)

	obj.App.Start()
}
