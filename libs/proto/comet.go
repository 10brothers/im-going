package proto

type PushMsgArg struct {
	Uid string
	P   Proto
}

// RoomMsgArg 发送到聊天室的消息
type RoomMsgArg struct {
	RoomId int32
	P      Proto
}

type RoomCountArg struct {
	RoomId int32
	Count  int
}
