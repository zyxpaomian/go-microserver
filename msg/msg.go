package msg

import "github.com/golang/protobuf/proto"

const CLIENT_MSG_HEARTBEAT = 1
const CLIENT_MSG_COLLECT = 2
const SERVER_MSG_HEARTBEAT_RESPONSE = 3
const CLIENT_MSG_RPMS = 4
const SERVER_MSG_AGENT_UPDATE = 5

// Msg ...
// 消息
type Msg struct {
	Type     uint64
	RawDatas []byte
	Msg      proto.Message
}
