package main

import (
	log "github.com/sirupsen/logrus"
	"im/libs/hash/cityhash"
	"time"
)

type ServerOptions struct {
	WriteWait       time.Duration
	PongWait        time.Duration
	PingPeriod      time.Duration
	MaxMessageSize  int64
	ReadBufferSize  int
	WriteBufferSize int
	BroadcastSize   int
}

type Server struct {
	Buckets   []*Bucket // subkey bucket
	Options   ServerOptions
	bucketIdx uint32
	operator  Operator
}

// NewServer returns a new Server.
func NewServer(b []*Bucket, o Operator, options ServerOptions) *Server {
	s := new(Server)
	s.Buckets = b
	s.Options = options
	s.bucketIdx = uint32(len(b))
	s.operator = o
	return s
}

// Bucket 结构体server上的方法，用于根据key来获取一个Bucket
func (server *Server) Bucket(subKey string) *Bucket {
	idx := cityhash.CityHash32([]byte(subKey), uint32(len(subKey))) % server.bucketIdx
	log.Printf("\"%s\" hit channel bucket index: %d use cityhash", subKey, idx)
	return server.Buckets[idx]
}
