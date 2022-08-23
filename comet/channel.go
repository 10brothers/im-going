package main

import (
	"github.com/gorilla/websocket"
	"im/libs/proto"
)

// Channel 表示每一个连接的双向链表
type Channel struct {
	Room      *Room             // 所属的房间信息
	broadcast chan *proto.Proto // 接收Proto指针类型的通道 chan
	uid       string
	conn      *websocket.Conn // 对应的websocket连接
	Next      *Channel        // 如果在房间的话
	Prev      *Channel
}

// NewChannel 有新的连接上来，创建一个Channel代表新连接
func NewChannel(svr int) *Channel {
	c := new(Channel)
	c.broadcast = make(chan *proto.Proto, svr)
	c.Next = nil
	c.Prev = nil
	return c
}

// Push 将要发送的消息，写入到对应Channel的chan通道中，然后通过websocket发送给客服端
func (ch *Channel) Push(p *proto.Proto) (err error) {
	select {
	case ch.broadcast <- p:
	default:
	}
	return
}
