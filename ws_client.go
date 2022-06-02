package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"sync"
	"time"
)

type websocketClientManager struct {
	conn        *websocket.Conn
	addr        *string
	path        string
	sendMsgChan chan string
	recvMsgChan chan string
	isAlive     bool
	timeout     int
}

// 构造函数
func NewWsClientManager(addrIp, addrPort, path string, timeout int) *websocketClientManager {
	addrString := addrIp + ":" + addrPort
	var sendChan = make(chan string, 10)
	var recvChan = make(chan string, 10)
	var conn *websocket.Conn
	return &websocketClientManager{
		addr:        &addrString,
		path:        path,
		conn:        conn,
		sendMsgChan: sendChan,
		recvMsgChan: recvChan,
		isAlive:     false,
		timeout:     timeout,
	}
}

// 链接服务端
func (wsc *websocketClientManager) dail() {
	var err error
	u := url.URL{Scheme: "ws", Host: *wsc.addr, Path: wsc.path}
	log.Printf("connecting to %s", u.String())
	wsc.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("[dail] error: ", err)
		return
	}
	wsc.isAlive = true
	log.Printf("connecting to %s 链接成功！！！", u.String())

}

// 发送消息
func (wsc *websocketClientManager) sendMsgThread() {
	go func() {
		for {
			msg := <-wsc.sendMsgChan
			switch msg {
			case "Ping":
				{
					err := wsc.conn.WriteMessage(websocket.PingMessage, []byte(msg))
					if err != nil {
						wsc.isAlive = false
						log.Println("write ping error:", err)
						continue
					}
				}
			case "Pong":
				{
					err := wsc.conn.WriteMessage(websocket.PongMessage, []byte(msg))
					if err != nil {
						wsc.isAlive = false
						log.Println("write pong error:", err)
						continue
					}
				}
			case "Close":
				{
					err := wsc.conn.WriteMessage(websocket.CloseMessage, []byte(msg))
					if err != nil {
						wsc.isAlive = false
						log.Println("write close error:", err)
						continue
					}
				}
			default:
				log.Println("write text:", msg)
				err := wsc.conn.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					wsc.isAlive = false
					log.Println("write text error:", err)
					continue
				}
			}
		}
	}()
}

// 读取消息
func (wsc *websocketClientManager) readMsgThread() {
	go func() {
		for {
			if wsc.conn != nil {
				if typ, _, err := wsc.conn.NextReader(); err != nil {
					log.Println("read failed:", err)
					log.Println("[NextReader] typ:", typ)
					wsc.conn.Close()
					break

				}
				typ, message, err := wsc.conn.ReadMessage()
				log.Println("[readMsgThread] typ:", typ)

				if err != nil {
					log.Println("read failed:", err)
					//wsc.isAlive = false
					// 出现错误，退出读取，尝试重连
					break
				}

				switch typ {
				case websocket.TextMessage:
					{
						log.Println("receive text: ", string(message))
					}
				case websocket.BinaryMessage:
					{
						log.Println("receive binary: ", message)

					}
				default:
					log.Println("unknown message: ", message)
				}
				// 需要读取数据，不然会阻塞
				wsc.recvMsgChan <- string(message)
			}

		}
	}()
}

// 开启服务并重连
func (wsc *websocketClientManager) start() {
	for {
		log.Println("wsc.isAlive: ", wsc.isAlive)
		if wsc.isAlive == false {
			wsc.dail()
			if wsc.conn != nil {
				wsc.sendMsgThread()
				wsc.readMsgThread()
			}
		}
		time.Sleep(time.Second * time.Duration(wsc.timeout))
	}
}

func (wsc *websocketClientManager) heartBeat() {
	pingTicker := time.NewTicker(time.Second * 3)
	defer func() {
		pingTicker.Stop()
		if wsc.conn != nil {
			log.Printf("conn %v\n", wsc.conn)
			log.Printf("close connection to %s\n", wsc.conn.RemoteAddr().String())
			wsc.conn.Close()
		}
	}()

	for {
		select {
		case <-pingTicker.C:
			if wsc.conn != nil {
				err := wsc.conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
				if err != nil {
					log.Println("ping error:", err.Error())
				}
				wsc.conn.SetPongHandler(func(appData string) error {
					log.Println("pong appData:", appData)
					return nil
				})
				wsc.conn.SetCloseHandler(func(code int, text string) error {
					log.Println("disconnecting...")
					log.Printf("Close code: [%v], peer close with text: %s ", code, text)
					err := wsc.conn.Close()
					if err != nil {
						log.Printf("disconnect error: %s", err)
						return err
					}
					log.Println("disconnected")
					return nil
				})
				wsc.sendMsgChan <- "Ping"
			}

			//case wsc.conn.SetWriteDeadline(time.Now().Add(time.Second * 2)){
			//	fmt.Println("--->send ping")
			//	if err := wsc.conn.WriteMessage(websocket.PingMessage, []byte{});
			//		err != nil {
			//		fmt.Println("send ping err:", err)
			//		return
			//	}
			//}

		}
	}
}

func main() {
	wsc := NewWsClientManager("10.10.10.114", "12345", "/login/数联宝/111", 10)
	//wsc := NewWsClientManager("levitas.quakeai.tech", "31081", "/login/store_id/topic/111", 10)

	wsc.sendMsgChan <- "hello websocket"
	go wsc.start()
	wsc.heartBeat()

	var w1 sync.WaitGroup
	w1.Add(1)
	w1.Wait()
}
