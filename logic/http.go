package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"im/libs/define"
	inet "im/libs/net"
	"im/libs/proto"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

type retData struct {
	Code int
	Msg  string
}

func InitHTTP() (err error) {
	// ServrMux 本质上是一个 HTTP 请求路由器
	var network, addr string

	for i := 0; i < len(Conf.Base.HttpAddrs); i++ {

		httpServeMux := http.NewServeMux()
		httpServeMux.HandleFunc("/api/v1/push", Push)
		httpServeMux.HandleFunc("/api/v1/pushRoom", PushRoom)
		httpServeMux.HandleFunc("/api/v1/count", Count)
		httpServeMux.HandleFunc("/api/v1/getRoomInfo", GetRoomInfo)
		httpServeMux.HandleFunc("/api/v1/coming", Login)
		httpServeMux.HandleFunc("/api/v1/friends", GetFriends)

		if network, addr, err = inet.ParseNetwork(Conf.Base.HttpAddrs[i]); err != nil {
			log.Errorf("inet.ParseNetwork() error(%v)", err)
			return
		}

		log.Infof("start http listen:\"%s\"", Conf.Base.HttpAddrs[i])

		go httpListen(httpServeMux, network, addr)
		select {}

	}
	return
}
func httpListen(mux *http.ServeMux, network, addr string) {

	// ServeTLS
	httpServer := &http.Server{Handler: mux, ReadTimeout: Conf.Base.HTTPReadTimeout, WriteTimeout: Conf.Base.HTTPWriteTimeout}
	httpServer.SetKeepAlivesEnabled(true)

	l, err := net.Listen(network, addr)
	if err != nil {
		log.Errorf("net.Listen(\"%s\", \"%s\") error(%v)", network, addr, err)
		panic(err)
	}
	if err := httpServer.Serve(l); err != nil {
		log.Errorf("server.Serve() error(%v)", err)
		panic(err)
	}
}

func InitHTTPS() (err error) {
	// ServrMux 本质上是一个 HTTP 请求路由器
	var network, addr string

	for i := 0; i < len(Conf.Base.HttpAddrs); i++ {

		httpServeMux := http.NewServeMux()
		httpServeMux.HandleFunc("/api/v1/push", Push)
		httpServeMux.HandleFunc("/api/v1/pushRoom", PushRoom)
		httpServeMux.HandleFunc("/api/v1/count", Count)
		httpServeMux.HandleFunc("/api/v1/getRoomInfo", GetRoomInfo)
		httpServeMux.HandleFunc("/api/v1/coming", Login)

		if network, addr, err = inet.ParseNetwork(Conf.Base.HttpAddrs[i]); err != nil {
			log.Errorf("inet.ParseNetwork() error(%v)", err)
			return
		}

		log.Infof("start http listen:\"%s\"", Conf.Base.HttpAddrs[i])

		go httpsListen(httpServeMux, network, addr)
		select {}

	}
	return
}

func httpsListen(mux *http.ServeMux, network, addr string) {

	// ServeTLS
	httpServer := &http.Server{Handler: mux, ReadTimeout: Conf.Base.HTTPReadTimeout, WriteTimeout: Conf.Base.HTTPWriteTimeout}
	httpServer.SetKeepAlivesEnabled(true)

	l, err := net.Listen(network, addr)
	if err != nil {
		log.Errorf("net.Listen(\"%s\", \"%s\") error(%v)", network, addr, err)
		panic(err)
	}
	if err := httpServer.ServeTLS(l, Conf.Base.CertPath, Conf.Base.KeyPath); err != nil {

		log.Errorf("Please check the certPath and keyPath of wss or other, error: %v", err)

		panic(err)
	}
}

func PushRoom(w http.ResponseWriter, r *http.Request) {
	// if r.Method != "POST" {
	// 	http.Error(w, "Method Not Allowed", 405)
	// }

	var (
		auth         = r.URL.Query().Get("auth")
		err          error
		bodyBytes    []byte
		body         string
		sendData     *proto.Send
		formUserInfo *proto.Router
		res          = map[string]interface{}{"code": define.SEND_ERR, "msg": define.SEND_ERR_MSG}
	)

	// get roomId
	rid, err := strconv.ParseInt(r.URL.Query().Get("rid"), 10, 32)
	if err != nil {
		log.Errorf("rid invalid : %s", rid)
		return

	}

	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		res["code"] = define.NETWORK_ERR
		res["msg"] = define.NETWORK_ERR_MSG
		log.Errorf("get router error : %s", err)
		return
	}

	if formUserInfo, err = getRouter(auth); err != nil {
		log.Errorf("get router error : %s", err)
		return
	}

	defer retPWrite(w, r, res, &body, time.Now())
	body = string(bodyBytes)
	log.Infof("PushRoom get bodyBytes : %s", body)
	json.Unmarshal(bodyBytes, &sendData)
	sendData.FormUserName = formUserInfo.UserName
	sendData.FormUserId = formUserInfo.UserId
	sendData.Op = define.OP_ROOM_SEND

	if bodyBytes, err = json.Marshal(sendData); err != nil {
		log.Errorf("redis Publish room err: %s", err)
		return
	}

	if err := RedisPublishRoom(int32(rid), bodyBytes); err != nil {
		log.Errorf("redis Publish room err: %s", err)
	}

	res["code"] = define.SUCCESS_REPLY
	res["msg"] = define.SUCCESS_REPLY_MSG
	return

}

// Push 接收用户的消息发送请求，写入redis的订阅组中
func Push(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
	}

	var (
		auth         = r.URL.Query().Get("auth")
		acceptUserId = r.URL.Query().Get("userId") // 消息接收人
		err          error
		bodyBytes    []byte
		body         string
		formUserInfo *proto.Router
		res          = map[string]interface{}{"code": define.SUCCESS_REPLY, "msg": define.SUCCESS_REPLY_MSG}
		sendData     *proto.Send
	)

	if formUserInfo, err = getRouter(auth); err != nil {
		log.Errorf("get router error : %s", err)
		return
	}

	log.Infof("push round userId %s", formUserInfo.UserId)

	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		res["code"] = define.NETWORK_ERR
		res["msg"] = define.NETWORK_ERR_MSG
		log.Errorf("get router error : %s", err)
		return
	}

	serverId := RedisCli.Get(getKey(acceptUserId)).Val()
	sid, err := strconv.ParseInt(serverId, 10, 8)
	if err != nil {
		res["code"] = define.SEND_ERR
		res["msg"] = define.SEND_ERR_MSG
		log.Errorf("router err %v", err)
		return
	}

	defer retPWrite(w, r, res, &body, time.Now())

	err = json.Unmarshal(bodyBytes, &sendData)
	if err != nil {
		return
	}
	sendData.FormUserName = formUserInfo.UserName
	sendData.FormUserId = formUserInfo.UserId
	sendData.Op = define.OP_SINGLE_SEND
	if bodyBytes, err = json.Marshal(sendData); err != nil {
		log.Errorf("redis Publish err: %s", err)
	}
	body = string(bodyBytes)

	if err := RedisPublishCh(int8(sid), acceptUserId, bodyBytes); err != nil {
		log.Errorf("redis Publish err: %s", err)

	}

}

/**
获取在线人数
*/

func Count(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
	}
	var (
		ridStr = r.URL.Query().Get("rid")
		count  int
		err    error
		rid    int
		res    = map[string]interface{}{"code": define.SEND_ERR, "msg": define.SEND_ERR_MSG}
	)

	if count, err = RedisCli.Get(getKey(ridStr)).Int(); err != nil {
		log.Warnf("Count redis get rid:%d, count err: %s", err)
		return
	}

	if rid, err = strconv.Atoi(ridStr); err != nil {
		log.Warnf("Count redis Count rid:%d, count err: %s", rid, err)
		return
	}

	if err = RedisPublishRoomCount(int32(rid), count); err != nil {
		log.Warnf("Count redis RedisPublishRoomCount err: %s", err)
		return
	}

	res["code"] = define.SUCCESS_REPLY
	res["msg"] = define.SUCCESS_REPLY_MSG
	defer retWrite(w, r, res, time.Now())
	return
}

/**
获取房间信息
*/

func GetRoomInfo(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
	}
	var (
		ridStr = r.URL.Query().Get("rid")
		rid    int
		err    error
		res    = map[string]interface{}{"code": define.SEND_ERR, "msg": define.SEND_ERR_MSG}
	)

	roomUserKey := getRoomUserKey(ridStr)
	roomUserInfo, err := RedisCli.HGetAll(roomUserKey).Result()
	if err != nil {
		log.Warnf("RedisCli HGetAll roomUserInfo key:%s, err: %s", roomUserKey, err)
	}
	if rid, err = strconv.Atoi(ridStr); err != nil {
		log.Warnf("Count redis Count rid:%d, count err: %s", rid, err)
		return
	}
	if err = RedisPublishRoomInfo(int32(rid), len(roomUserInfo), roomUserInfo); err != nil {
		log.Warnf("Count redis RedisPublishRoomCount err: %s", err)
		return
	}
	res["code"] = define.SUCCESS_REPLY
	res["msg"] = define.SUCCESS_REPLY_MSG
	defer retWrite(w, r, res, time.Now())
	return
}

func retPWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, body *string, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := string(data)
	log.Infof("dataStr %s", dataStr)

	w.Header().Set("Access-Control-Allow-Origin", "*")             // 允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") // header的类型
	w.Header().Set("content-type", "application/json")
	// 返回数据格式是json
	if _, err := w.Write([]byte(dataStr)); err != nil {
		log.Errorf("w.Write(\"%s\") error(%v)", dataStr, err)
	}

	log.Infof("req: \"%s\", post: \"%s\", res:\"%s\", ip:\"%s\", time:\"%fs\"", r.URL.String(), *body, dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}

// retWrite marshal the result and write to client(get).
func retWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		log.Error("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := string(data)
	w.Header().Set("Access-Control-Allow-Origin", "*")             // 允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") // header的类型
	w.Header().Set("content-type", "application/json")
	if _, err := w.Write([]byte(dataStr)); err != nil {
		log.Error("w.Write(\"%s\") error(%v)", dataStr, err)
	}

	log.Info("req: \"%s\", get: res:\"%s\", ip:\"%s\", time:\"%fs\"", r.URL.String(), dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}

type UP struct {
	U888 string `json:"u888"` // 如果除了首字母大小写差异外，可以不用显示的映射json字段
	M666 string `json:"m666"`
}

func Login(resp http.ResponseWriter, req *http.Request) {
	var res = map[string]interface{}{"code": define.SEND_ERR, "msg": define.SEND_ERR_MSG}

	defer retWrite(resp, req, res, time.Now())
	var (
		bodyBytes []byte
		err       error
	)
	if bodyBytes, err = ioutil.ReadAll(req.Body); err != nil {
		res["msg"] = "Error"
		return
	}
	log.Infof("PushRoom get bodyBytes : %s", bodyBytes)
	var up *UP
	errJson := json.Unmarshal(bodyBytes, &up)
	if errJson != nil { // nil表示zero value 并不是错误的意思 也不表示null
		res["msg"] = "反序列化异常"
		return
	}

	//
	var key = define.REDIS_ACCOUNT_PREFIX + up.U888 + up.M666
	val := RedisCli.Get(key).Val()
	log.Infof("key %s 的值 %s", key, val)
	if val == "" {
		res["msg"] = "滚滚滚!!!"
		return
	}

	RedisCli.HSet(define.REDIS_PREFIX+val, "UserId", up.U888)
	RedisCli.HSet(define.REDIS_PREFIX+val, "UserName", "anonymous|"+up.U888)
	RedisCli.Set(define.REDIS_PREFIX+up.U888, 1, 24*time.Hour)
	res["code"] = define.SUCCESS_REPLY
	res["msg"] = val
	return
}

type Friend struct {
	UserId   string
	NickName string
}

func GetFriends(w http.ResponseWriter, r *http.Request) {
	var res = map[string]interface{}{"code": define.SEND_ERR, "msg": define.SEND_ERR_MSG}

	defer retWrite(w, r, res, time.Now())
	var (
		token  = r.URL.Query().Get("auth")
		userId = r.URL.Query().Get("userId")
		list   []Friend
	)

	storeUid := RedisCli.HGet(define.REDIS_PREFIX+token, "UserId").Val()
	if userId != storeUid {
		res["msg"] = "Auth Error"
		return
	}
	friends := RedisCli.HGetAll(define.REDIS_FRIENDS + userId).Val()
	for userId, name := range friends {
		f := new(Friend)
		f.UserId = userId
		f.NickName = name
		list = append(list, *f)
	}
	res["friend"] = list
	res["code"] = 0
	res["msg"] = "success"
	return
}
