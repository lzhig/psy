/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 11:25:01
 * @modify date 2018-01-19 11:25:01
 * @desc [description]
 */

package main

import (
	"fmt"

	"github.com/lzhig/rapidgo/net"
)

// NetworkEngine type
type NetworkEngine struct {
	app *App

	server   *net.TCPServer
	callback netCallback
}

// Start function
func (obj *NetworkEngine) Start(addr string, maxUsers uint32) {
	obj.server = net.CreateTCPServer()

	err := obj.server.Start(addr, maxUsers, obj.callback)

	fmt.Println(err)
}

// // Start function
// func (obj *NetworkEngine) Start() {
// 	ctx, _ := obj.app.CreateCancelContext()

// 	obj.app.GoRoutine(ctx,
// 		func(ctx context.Context, args ...interface{}) {
// 			fmt.Println(args)
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					fmt.Println("goroutine 1 exit.")
// 					return
// 				}
// 			}
// 		},
// 		1)
// }

type netCallback struct {
}

func (callback netCallback) Disconnected(conn *net.Connection, err error) {
	fmt.Println(conn.RemoteAddr().String(), "disconnected error: ", err)
}

func (callback netCallback) Connected(conn *net.Connection) {
	fmt.Println(conn.RemoteAddr().String(), "connected")
}

func (callback netCallback) Received(conn *net.Connection, packet net.Packet) {
	fmt.Println(conn.RemoteAddr().String(), "data received")
}
