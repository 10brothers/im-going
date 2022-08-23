package main

import (
	"im/libs/define"
	"im/libs/proto"
	"sync"
	"sync/atomic"
)

type BucketOptions struct {
	ChannelSize   int
	RoomSize      int
	RoutineAmount uint64
	RoutineSize   int
}

// Bucket 用来管理连接以及消息推送的
type Bucket struct {
	cLock    sync.RWMutex        // protect the channels for chs
	chs      map[string]*Channel // map sub key to a channel 单聊场景下的用户id和Channel的映射
	boptions BucketOptions
	// room
	rooms       map[int32]*Room          // bucket room channels 房间号和房间的对应关系
	routines    []chan *proto.RoomMsgArg //
	routinesNum uint64
	broadcast   chan []byte // 单聊场景下的消息通道
}

func NewBucket(boptions BucketOptions) (b *Bucket) {
	b = new(Bucket)
	b.chs = make(map[string]*Channel, boptions.ChannelSize)

	b.boptions = boptions
	b.routines = make([]chan *proto.RoomMsgArg, boptions.RoutineAmount)
	b.rooms = make(map[int32]*Room, boptions.RoomSize)
	for i := uint64(0); i < b.boptions.RoutineAmount; i++ {
		c := make(chan *proto.RoomMsgArg, boptions.RoutineSize)
		b.routines[i] = c
		go b.PushRoom(c) // 创建一个协程用于处理群聊消息推送
	}
	return
}

// Put 将连接上来的用户对应的websocket 放到当前的Bucket来管理
func (b *Bucket) Put(uid string, rid int32, ch *Channel) (err error) {
	var (
		room *Room
		ok   bool
	)
	b.cLock.Lock()

	if rid != define.NO_ROOM { // 存在房间号的情况下，表示进入房间
		if room, ok = b.rooms[rid]; !ok {
			room = NewRoom(rid)
			b.rooms[rid] = room
		}
		ch.Room = room
	}
	ch.uid = uid
	b.chs[uid] = ch
	b.cLock.Unlock()

	if room != nil {
		err = room.Put(ch)
	}
	return
}

// Channel 根据key，从bucket获取一个Channel
func (b *Bucket) Channel(key string) (ch *Channel) {
	// 读操作的锁定和解锁
	b.cLock.RLock()
	ch = b.chs[key]
	b.cLock.RUnlock()
	return
}

// PushRoom 用于处理群聊的消息通道
func (b *Bucket) PushRoom(c chan *proto.RoomMsgArg) {
	for {
		var (
			arg  *proto.RoomMsgArg
			room *Room
		)
		arg = <-c

		if room = b.Room(arg.RoomId); room != nil {
			room.Push(&arg.P)
		}

	}

}

func (b *Bucket) delCh(ch *Channel) {
	var (
		ok   bool
		room *Room
	)
	b.cLock.RLock()

	if ch, ok = b.chs[ch.uid]; ok {
		room = b.chs[ch.uid].Room
		delete(b.chs, ch.uid)

	}
	if room != nil && room.Del(ch) {
		// if room empty delete
		room.Del(ch)
	}

	b.cLock.RUnlock()

}

// Room get a room by roomid.
func (b *Bucket) Room(rid int32) (room *Room) {
	b.cLock.RLock()
	room, _ = b.rooms[rid]
	b.cLock.RUnlock()
	return
}

func (b *Bucket) BroadcastRoom(arg *proto.RoomMsgArg) {
	// 广播消息递增id
	num := atomic.AddUint64(&b.routinesNum, 1) % b.boptions.RoutineAmount
	// log.Infof("BroadcastRoom RoomMsgArg :%s", arg)
	// log.Infof("bucket routinesNum :%d", b.routinesNum)
	b.routines[num] <- arg

}
