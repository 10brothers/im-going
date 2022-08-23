package main

import (
	"im/libs/proto"
)

type Operator interface {
	Connect(*proto.ConnArg) (string, error)
	Disconnect(*proto.DisconnArg) error
}

type DefaultOperator struct {
}

// Connect 通过logic服务来完成登录认证
func (operator *DefaultOperator) Connect(connArg *proto.ConnArg) (uid string, err error) {
	// var connReply *proto.ConnReply
	uid, err = connect(connArg)
	return
}

// Disconnect 注销登录
func (operator *DefaultOperator) Disconnect(disconnArg *proto.DisconnArg) (err error) {

	if err = disconnect(disconnArg); err != nil {
		return
	}

	return
}
