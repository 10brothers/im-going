package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"im/libs/proto"
	"net/http"
	"time"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path[1:])
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func InitWebsocket() (err error) {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(DefaultServer, w, r)
	}) // 用于处理websocket过来的请求，建立websocket连接，

	err = http.ListenAndServe(Conf.Websocket.Bind, nil) // 在给定端口上启动http服务器
	return err
}

// InitWebsocketWss 接入TLS的websocket
func InitWebsocketWss() (err error) {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(DefaultServer, w, r)
	})

	err = http.ListenAndServeTLS(Conf.Websocket.Bind, Conf.Base.CertPath, Conf.Base.KeyPath, nil)
	return err
}

// serveWs handles websocket requests from the peer.
func serveWs(server *Server, w http.ResponseWriter, r *http.Request) {

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  DefaultServer.Options.ReadBufferSize,
		WriteBufferSize: DefaultServer.Options.WriteBufferSize,
	}
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// 将协议升级到websocket
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Error(err)
		return
	}
	var ch *Channel
	// 写入配置
	ch = NewChannel(server.Options.BroadcastSize)
	ch.conn = conn

	go server.writePump(ch)
	go server.readPump(ch)
}

func (s *Server) readPump(ch *Channel) {
	defer func() { //   用于异常时做一些处理操作，等同于catch 或者finally
		disconnArg := new(proto.DisconnArg)

		disconnArg.RoomId = ch.Room.Id
		if ch.uid != "" {
			disconnArg.Uid = ch.uid
		}

		s.Bucket(ch.uid).delCh(ch)
		if err := s.operator.Disconnect(disconnArg); err != nil {
			log.Warnf("Disconnect err :%s", err)
		}
		ch.conn.Close()
	}()

	ch.conn.SetReadLimit(s.Options.MaxMessageSize)
	ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
	ch.conn.SetPongHandler(func(string) error {
		ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
		return nil
	})

	for {
		_, message, err := ch.conn.ReadMessage() // 这里会阻塞，直到有信息发过来
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("readPump ReadMessage err:%v", err)
				return
			}
		}
		if message == nil {
			return
		}
		var (
			connArg *proto.ConnArg
		)

		log.Infof("message :%s", message)
		if err := json.Unmarshal([]byte(message), &connArg); err != nil {
			log.Errorf("message struct %b", connArg)
		}
		connArg.ServerId = Conf.Base.ServerId

		uid, err := s.operator.Connect(connArg)
		log.Infof("websocket uid:%s", uid)
		if err != nil {
			log.Errorf("s.operator.Connect error %s", err)
			return
		}
		if uid == "" {
			log.Error("Invalid Auth ,uid empty")
			return
		}

		b := s.Bucket(uid)

		// rpc 操作获取uid 存入ch 存入Server 未写
		// b.broadcast <- message
		// 如果已经连接上了，可以在进入不同的房间时，不断更新房间信息
		log.Infof("connArg roomId : %d", connArg.RoomId)
		err = b.Put(uid, connArg.RoomId, ch)
		if err != nil {
			log.Errorf("conn close err: %s", err)
			ch.conn.Close()
		}
		log.Infof("message  333 :%s", message)
		// ch.broadcast <- message

	}
}

func (s *Server) writePump(ch *Channel) {
	ticker := time.NewTicker(s.Options.PingPeriod)
	log.Printf("ticker :%v", ticker)

	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case message, ok := <-ch.broadcast:
			ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			if !ok {
				// The hub closed the channel.
				log.Warn("SetWriteDeadline not ok ")
				ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := ch.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Warn(" ch.conn.NextWriter err :%s  ", err)
				return
			}
			log.Infof("message write body:%s", message.Body)
			_, err = w.Write(message.Body)
			if err != nil {
				return
			}

			// Add queued chat messages to the current websocket message.
			// n := len(ch.broadcast)
			// for i := 0; i < n; i++ {
			// 	w.Write(newline)
			// 	w.Write(<-ch.broadcast)
			// }

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			err := ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			if err != nil {
				return
			}
			log.Printf("websocket.PingMessage :%v", websocket.PingMessage)
			if err := ch.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
