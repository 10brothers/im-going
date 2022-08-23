package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/server"
	"im/libs/define"
	inet "im/libs/net"
	"im/libs/proto"
)

type PushRpc int

func InitLogicRpcServer() (err error) {
	log.Info("InitPushRpc")
	var (
		network, addr string
	)
	for _, bind := range Conf.RpcCometAddrs {
		if network, addr, err = inet.ParseNetwork(bind.Addr); err != nil {
			log.Panicf("InitLogicRpc ParseNetwork error : %s", err)
		}
		log.Infof("InitPushRpc addr %s", addr)
		go createServer(network, addr)
	}
	return
}

func createServer(network string, addr string) {
	flag.Parse()
	s := server.NewServer()

	//addRegistryPlugin(s, network, addr)

	err := s.RegisterName("comet", new(PushRpc), "")
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	log.Infof("createServer addr %s", addr)
	err = s.Serve(network, addr)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
}

func (rpc *PushRpc) PushSingleMsg(ctx context.Context, args *proto.PushMsgArg, SuccessReply *proto.SuccessReply) (err error) {
	var (
		bucket  *Bucket
		channel *Channel
	)

	log.Info("rpc PushMsg :%v ", args)
	if args == nil {
		log.Errorf("rpc PushRpc() error(%v)", err)
		return
	}
	bucket = DefaultServer.Bucket(args.Uid)
	if channel = bucket.Channel(args.Uid); channel != nil {
		err = channel.Push(&args.P)

		log.Infof("DefaultServer Channel err nil : %v", err)
		return
	}

	SuccessReply.Code = define.SUCCESS_REPLY
	SuccessReply.Msg = define.SUCCESS_REPLY_MSG
	log.Infof("SuccessReply v :%v", SuccessReply)
	return
}

func (rpc *PushRpc) PushRoomMsg(ctx context.Context, args *proto.RoomMsgArg, SuccessReply *proto.SuccessReply) (err error) {

	SuccessReply.Code = define.SUCCESS_REPLY
	SuccessReply.Msg = define.SUCCESS_REPLY_MSG
	log.Infof("PushRoomMsg msg %v", args)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(args)
		// room.next

	}
	return
}

// PushRoomCount 广播房间人数
func (rpc *PushRpc) PushRoomCount(ctx context.Context, args *proto.RoomMsgArg, SuccessReply *proto.SuccessReply) (err error) {
	SuccessReply.Code = define.SUCCESS_REPLY
	SuccessReply.Msg = define.SUCCESS_REPLY_MSG
	log.Infof("PushRoomCount count %v", args)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(args)
		// room.next
	}
	return
}

// PushRoomInfo 广播房间信息
func (rpc *PushRpc) PushRoomInfo(ctx context.Context, args *proto.RoomMsgArg, SuccessReply *proto.SuccessReply) (err error) {
	log.Infof("PushRoomInfo  %v", args)
	SuccessReply.Code = define.SUCCESS_REPLY
	SuccessReply.Msg = define.SUCCESS_REPLY_MSG

	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(args)
		// room.next
	}
	return
}
