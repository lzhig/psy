/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 11:25:01
 * @modify date 2018-01-19 11:25:01
 * @desc [description]
 */

package main

import (
	"context"
	"fmt"

	"./msg"

	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/rapidnet"
)

// NetworkEngine type
type NetworkEngine struct {
	server *rapidnet.TCPServer

	eventChan <-chan *rapidnet.Event

	protoHandler protocolHandler
}

func (obj *NetworkEngine) init() {
	obj.protoHandler.init()
}

// Start function
func (obj *NetworkEngine) Start(addr string, maxUsers uint32) error {
	obj.server = rapidnet.CreateTCPServer()

	var err error
	obj.eventChan, err = obj.server.Start(addr, maxUsers)

	if err == nil {
		ctx, _ := gApp.CreateCancelContext()

		gApp.GoRoutineArgs(ctx,
			func(ctx context.Context, args ...interface{}) {
				defer debug("exit NetworkEngine Event goroutine")
				for {
					select {
					case <-ctx.Done():
						return

					case event := <-obj.eventChan:
						switch event.Type {
						case rapidnet.EventConnected:
							fmt.Println(event.Conn.RemoteAddr().String(), "connected")
							ctx, _ := context.WithCancel(ctx)
							gApp.GoRoutineArgs(ctx, obj.handleConnection, event.Conn)
						case rapidnet.EventDisconnected:
							fmt.Println(event.Conn.RemoteAddr().String(), "disconnected", event.Err)

						case rapidnet.EventSendFailed:
							fmt.Println(event.Conn.RemoteAddr().String(), "Failed to send", event.Err)
						}
					}
				}
			})
	}
	return err
}

func (obj *NetworkEngine) handleConnection(ctx context.Context, args ...interface{}) {
	defer debug("exit NetworkEngine handleConnection goroutine")
	conn := args[0].(*rapidnet.Connection)
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-conn.ReceiveDataChan():
			if data == nil {
				return
			}
			fmt.Println("Recieve data. size:", len(data))

			p := &msg.Protocol{}
			if err := proto.Unmarshal(data, p); err != nil {
				fmt.Println(err)
				conn.Disconnect()
				return
			}

			obj.protoHandler.getProtoChan() <- &ProtocolConnection{p: p, conn: conn}
		}
	}
}

// ProtocolConnection type
type ProtocolConnection struct {
	p    *msg.Protocol
	conn *rapidnet.Connection
}
