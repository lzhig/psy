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
	"time"

	"./msg"

	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/base"
	"github.com/lzhig/rapidgo/rapidnet"
)

// NetworkEngine type
type NetworkEngine struct {
	server *rapidnet.TCPServer

	eventChan <-chan *rapidnet.Event
}

func (obj *NetworkEngine) init() {
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
							gApp.GoRoutineArgs(ctx, obj.handleConnection, &Connection{conn: event.Conn})

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
	//defer debug("exit NetworkEngine handleConnection goroutine")
	if args == nil || len(args) == 0 {
		base.LogError("[NetworkEngine][handleConnection] invalid args")
		return
	}
	userconn := args[0].(*Connection)
	defer func() {
		if userconn.user != nil {
			onlineStatistic.OnlinePersonsChange(false)
			base.LogInfo("disconnected. uid:", userconn.user.uid)
			userconn.user.Disconnect(userconn)
			//userconn.user = nil
		}
	}()
	ticker := time.NewTicker(time.Second * 5)
	lastTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-userconn.conn.ReceiveDataChan():
			// 连接被关闭
			if !ok || (userconn.user != nil && userconn.user.conn != userconn) {
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
			} else if p.Msgid == msg.MessageID_Login_Req && userconn.user != nil {
				base.LogError("Already login.", userconn.conn.RemoteAddr().String())
				userconn.Disconnect()
				return
			}

			lastTime = time.Now()

			obj.handle(p.Msgid, &ProtocolConnection{p: p, userconn: userconn, user: userconn.user})

		case <-ticker.C:
			// 登录超时
			if userconn.user == nil && time.Now().Sub(lastTime) > time.Duration(gApp.config.Server.LoginTimeout)*time.Second {
				base.LogInfo("login timeout.")
				userconn.Disconnect()
				return
			}

			// idle超时
			if time.Now().Sub(lastTime) > time.Duration(gApp.config.Server.IdleTime)*time.Second {
				base.LogInfo("disconnect idle connection.")
				userconn.Disconnect()
				return
			}
		}
	}
}

type Connection struct {
	user *User
	conn *rapidnet.Connection
}

func (obj *Connection) Disconnect() {
	obj.conn.Disconnect()
}

func (obj *Connection) sendProtocol(p *msg.Protocol) {
	sendProtocol(obj.conn, p)
}

// ProtocolConnection type
type ProtocolConnection struct {
	p        *msg.Protocol
	userconn *Connection
	user     *User
}
