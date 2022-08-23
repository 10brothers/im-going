package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/client"
	"im/libs/proto"
)

var (
	logicRpcClient client.XClient
)

func InitLogicRpcClient() (err error) {
	discovery, err := client.NewPeer2PeerDiscovery(Conf.RpcLogicAddrs.Addr, "")
	if err != nil {
		return err
	}

	logicRpcClient = client.NewXClient(
		"logic",
		client.Failtry,
		client.RandomSelect,
		discovery,
		client.DefaultOption)
	return
}

func connect(connArg *proto.ConnArg) (uid string, err error) {
	reply := &proto.ConnReply{}
	err = logicRpcClient.Call(context.Background(), "Connect", connArg, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	uid = reply.Uid
	log.Infof("comet logic uid :%s", reply.Uid)

	return
}

func disconnect(disconnArg *proto.DisconnArg) (err error) {

	reply := &proto.DisconnReply{}
	if err = logicRpcClient.Call(context.Background(), "Disconnect", disconnArg, reply); err != nil {
		log.Fatalf("failed to call: %v", err)
	}
	return
}
