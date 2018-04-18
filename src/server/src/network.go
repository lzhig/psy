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

	"./msg"

	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/base"
	"github.com/lzhig/rapidgo/rapidnet"
)

// NetworkEngine type
type NetworkEngine struct {
	server *rapidnet.TCPServer

	eventChan <-chan *rapidnet.Event

	//protoHandler protocolHandler
}

func (obj *NetworkEngine) init() {
	//obj.protoHandler.init()
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
							base.LogInfo(event.Conn.RemoteAddr().String(), " connected")
							ctx, _ := context.WithCancel(ctx)
							gApp.GoRoutineArgs(ctx, obj.handleConnection, &userConnection{conn: event.Conn})

							// todo 连接成功后，一段时间后需要登录成功，否则将断线
						case rapidnet.EventDisconnected:
							base.LogInfo(event.Conn.RemoteAddr().String(), " disconnected. error:", event.Err)

						case rapidnet.EventSendFailed:
							base.LogInfo(event.Conn.RemoteAddr().String(), " Failed to send. error:", event.Err)
						}
					}
				}
			})
	}
	return err
}

func (obj *NetworkEngine) handleConnection(ctx context.Context, args ...interface{}) {
	defer debug("exit NetworkEngine handleConnection goroutine")
	if args == nil || len(args) == 0 {
		base.LogError("[NetworkEngine][handleConnection] invalid args")
		return
	}
	userconn := args[0].(*userConnection)
	defer func() {
		if userconn.user != nil {
			userManager.userDisconnect(userconn.user.uid, userconn)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-userconn.conn.ReceiveDataChan():
			if !ok {
				return
			}

			p := &msg.Protocol{}
			if err := proto.Unmarshal(data, p); err != nil {
				base.LogWarn("failed to unmarshal request. error:", err, ". address:", userconn.conn.RemoteAddr().String())
				userconn.Disconnect()
				return
			}

			// 如果没有登录，不处理其他协议
			if userconn.user == nil && p.Msgid != msg.MessageID_Login_Req {
				base.LogWarn("receive request while no login. address:", userconn.conn.RemoteAddr().String())
				userconn.Disconnect()
				return
			}

			obj.handle(p.Msgid, &ProtocolConnection{p: p, userconn: userconn})
		}
	}
}

type userConnection struct {
	//uid uint32
	//name string
	user *User
	conn *rapidnet.Connection
	//room *Room
}

func (obj *userConnection) Disconnect() {
	obj.conn.Disconnect()
}

func (obj *userConnection) sendProtocol(p *msg.Protocol) {
	sendProtocol(obj.conn, p)
}

// ProtocolConnection type
type ProtocolConnection struct {
	p        *msg.Protocol
	userconn *userConnection
}
