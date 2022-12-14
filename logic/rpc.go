package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/server"
	"im/libs/define"
	inet "im/libs/net"
	"im/libs/proto"
	"strconv"
	"time"
)

type LogicRpc int

func InitRPC() (err error) {

	var (
		network, addr string
	)
	// 单台开多个端口的情况
	for _, bind := range Conf.Base.RPCAddrs {
		log.Infof("RPCAddrs :%v", bind)
		if network, addr, err = inet.ParseNetwork(bind); err != nil {
			log.Panicf("InitLogicRpc ParseNetwork error : %s", err)
		}

		go createServer(network, addr)
	}

	return
}

func createServer(network string, addr string) {

	s := server.NewServer()
	//addRegistryPlugin(s, network, addr)
	// serverId must be unique
	err := s.RegisterName("logic", new(LogicRpc), "")
	if err != nil {
		return
	}
	err = s.Serve(network, addr)
	if err != nil {
		return
	}
}

// Connect 通过comet建立websocket连接时登录认证接口
func (rpc *LogicRpc) Connect(ctx context.Context, args *proto.ConnArg, reply *proto.ConnReply) (err error) {

	if args == nil {
		log.Errorf("Connect() error(%v)", err)
		return
	}

	key := getKey(args.Auth)
	log.Infof("logic rpc key:%s", key)
	userInfo, err := RedisCli.HGetAll(key).Result()
	if err != nil {
		log.Infof("RedisCli HGetAll key :%s , err:%s", key, err)
	}

	reply.Uid = userInfo["UserId"]
	roomUserKey := getRoomUserKey(strconv.Itoa(int(args.RoomId)))
	// UserId
	if reply.Uid == "" {
		reply.Uid = define.NO_AUTH
	} else {
		userKey := getKey(reply.Uid)
		log.Infof("logic redis set uid serverId:%s, serverId : %d", userKey, args.ServerId)
		validTime := define.REDIS_BASE_VALID_TIME * time.Second
		err = RedisCli.Set(userKey, args.ServerId, validTime).Err()
		if err != nil {
			log.Warnf("logic set err:%s", err)
		}
		// write redis roomUserList
		RedisCli.HSet(roomUserKey, reply.Uid, userInfo["UserName"])
	}

	// 增加房间人数
	RedisCli.Incr(getKey(strconv.FormatInt(int64(args.RoomId), 10)))

	log.Infof("logic rpc uid:%s", reply.Uid)

	return
}

func (rpc *LogicRpc) Disconnect(ctx context.Context, args *proto.DisconnArg, reply *proto.DisconnReply) (err error) {

	roomUserKey := getRoomUserKey(strconv.Itoa(int(args.RoomId)))

	// 房间总人数减少
	_, err = RedisCli.Decr(getKey(strconv.FormatInt(int64(args.RoomId), 10))).Result()
	if err != nil {
		return err
	}

	// 房间登录人数减少
	if args.Uid != define.NO_AUTH {
		err = RedisCli.HDel(roomUserKey, args.Uid).Err()
		if err != nil {
			log.Warnf("HDel getRoomUserKey err : %s", err)
		}

	}

	roomUserInfo, err := RedisCli.HGetAll(roomUserKey).Result()
	if err != nil {
		log.Warnf("RedisCli HGetAll roomUserInfo key:%s, err: %s", roomUserKey, err)
	}

	if err = RedisPublishRoomInfo(args.RoomId, len(roomUserInfo), roomUserInfo); err != nil {
		log.Warnf("Count redis RedisPublishRoomCount err: %s", err)
		return
	}
	return
}
