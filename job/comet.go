package main

import (
	"context"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/client"
	"im/libs/define"
	"im/libs/proto"
)

// type CometRpc int

var (
	cometRpcClient client.XClient
)

func InitComets() (err error) {

	discovery, err := client.NewPeer2PeerDiscovery(Conf.CometRpc.Addr, "")
	if err != nil {
		log.Error(err)
		return nil
	}
	cometRpcClient = client.NewXClient("comet", client.Failtry, client.RandomSelect, discovery, client.DefaultOption)
	return
}

// PushSingleToComet 广播消息到单个用户
func PushSingleToComet(serverId int8, userId string, msg []byte) {
	log.Infof("PushSingleToComet Body %s", msg)
	pushMsgArg := &proto.PushMsgArg{Uid: userId, P: proto.Proto{Ver: 1, Operation: define.OP_SINGLE_SEND, Body: msg}}
	reply := &proto.SuccessReply{}
	err := cometRpcClient.Call(context.Background(), "PushSingleMsg", pushMsgArg, reply)
	if err != nil {
		log.Infof(" PushSingleToComet Call err %v", err)
	}
	log.Infof("reply %s", reply.Msg)
}

/*
*
广播消息到房间
*/
func broadcastRoomToComet(RoomId int32, msg []byte) {
	pushMsgArg := &proto.RoomMsgArg{
		RoomId: RoomId, P: proto.Proto{
			Ver:       1,
			Operation: define.OP_ROOM_SEND,
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	log.Infof("broadcastRoomToComet roomid %d", RoomId)
	cometRpcClient.Call(context.Background(), "PushRoomMsg", pushMsgArg, reply)
}

/*
*
广播在线人数到房间
*/
func broadcastRoomCountToComet(RoomId int32, count int) {

	var (
		body []byte
		err  error
	)
	msg := &proto.RedisRoomCountMsg{
		Count: count,
		Op:    define.OP_ROOM_COUNT_SEND,
	}

	if body, err = json.Marshal(msg); err != nil {
		log.Warnf("broadcastRoomCountToComet  json.Marshal err :%s", err)
		return
	}

	pushMsgArg := &proto.RoomMsgArg{
		RoomId: RoomId, P: proto.Proto{
			Ver:       1,
			Operation: define.OP_ROOM_SEND,
			Body:      body,
		},
	}

	reply := &proto.SuccessReply{}
	cometRpcClient.Call(context.Background(), "PushRoomCount", pushMsgArg, reply)
}

/*
*
广播房间信息到房间
*/
func broadcastRoomInfoToComet(RoomId int32, RoomUserInfo map[string]string) {

	var (
		body []byte
		err  error
	)
	msg := &proto.RedisRoomInfo{
		Count:        len(RoomUserInfo),
		Op:           define.OP_ROOM_COUNT_SEND,
		RoomUserInfo: RoomUserInfo,
		RoomId:       RoomId,
	}

	if body, err = json.Marshal(msg); err != nil {
		log.Warnf("broadcastRoomInfoToComet  json.Marshal err :%s", err)
		return
	}

	pushMsgArg := &proto.RoomMsgArg{
		RoomId: RoomId, P: proto.Proto{
			Ver:       1,
			Operation: define.OP_ROOM_SEND,
			Body:      body,
		},
	}
	reply := &proto.SuccessReply{}
	cometRpcClient.Call(context.Background(), "PushRoomInfo", pushMsgArg, reply)
}
