package server

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"microserver/common"
	cfg "microserver/common/configparse"
	se "microserver/common/error"
	log "microserver/common/formatlog"
	"microserver/msg"
	"net"
	"sync"
	"time"
)

type Client struct {
	clientId              string      //客户端ID
	state                 int         // agent 状态字段
	conn                  net.Conn    // agent的Conn
	sendLock              *sync.Mutex // 发送锁
	readBuf               []byte      // 读取的缓存
	readMsgPayloadLth     uint64      // 读取的当前消息的长度
	readTotalBytesLth     uint64      // 读取的总的消息长度
	lastHeartbeatSyncTime string      // 最近一次心跳同步时间
	readTimeout           int         // 读超时
	writeTimeout          int         // 写超时
	agentHeartbeatTimeout int         // heartbeat超时时间，单位分钟
}

// Client初始化
func NewClient(conn net.Conn, clientId string) *Client {
	client := Client{
		conn:                  conn,
		clientId:              clientId,
		readBuf:               []byte{},
		readMsgPayloadLth:     0,
		state:                 Waiting,
		lastHeartbeatSyncTime: "1970-01-01 00:00:00",
		sendLock:              &sync.Mutex{},
		readTotalBytesLth:     0,
		readTimeout:           cfg.GlobalConf.GetInt("common", "readtimeout"),
		writeTimeout:          cfg.GlobalConf.GetInt("common", "writeimeout"),
		agentHeartbeatTimeout: cfg.GlobalConf.GetInt("common", "agentHeartbeatTimeout"),
	}
	return &client
}

// 设置最近一次心跳同步时间
func (c *Client) SetLastHeartbeatSyncTime(t string) {
	log.Debugf("[IOServer]  %s 的心跳时间更新为 %s", c.clientId, t)
	c.lastHeartbeatSyncTime = t
}

// 客户端是否有效
func (c *Client) Valid() bool {
	if c.state == Erroring {
		return false
	}

	// 如果最近一次心跳同步时间为默认值，等待下次检查
	if c.lastHeartbeatSyncTime == "1970-01-01 00:00:00" {
		return true
	}

	now := time.Now()
	h, _ := time.ParseDuration(fmt.Sprintf("-%dm", c.agentHeartbeatTimeout))
	targetDate := now.Add(h)
	targetDateStr := targetDate.Format(common.TIME_FORMAT)
	if targetDateStr > c.lastHeartbeatSyncTime {
		log.Errorf("[IOServer] 客户端 %s 的心跳时间 %s 超时，允许间隔 %d 分钟", c.clientId, targetDateStr, c.lastHeartbeatSyncTime)
		return false
	}

	return true
}

// 读取一个消息
func (c *Client) GetMsg() (*msg.Msg, error) {
	/*
		消息格式: 8字节protobuf报文长度 + 4字节类型 + protobuf报文
		读取逻辑:
		- 读取报文，将报文和client内部的buf相加，如果长度小于8则继续读取
		- 如果长度大于8则根据计算得到的长度读取所有数据
		- 读取后解析protobuf生成msg，并消耗相关的buf
	*/

	for {
		if c.state == Erroring {
			log.Warnf("[IOServer] %s 无效，退出读消息循环", c.clientId)
			return nil, se.New(fmt.Sprintf("%s 无效，退出读取循环", c.clientId))
		}

		// 设置读的超时时间
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.readTimeout) * time.Second))

		if len(c.readBuf) < 8 {
			// 长度信息还没有读取到
			log.Debugf("[IOServer] 准备从 %s 读取报文，当前报文长度小于8", c.clientId)
			readBuf := make([]byte, 512)
			n, err := c.conn.Read(readBuf)
			log.Debugf("[IOServer] 从 %s 读取到报文", c.clientId)
			if err != nil {
				log.Errorf("[IOServer] 从 %s 读取数据时报错，报错内容: %s", c.clientId, err.Error())
				return nil, err
			}
			c.conn.SetReadDeadline(time.Time{})
			log.Debugf("[IOServer] 此次读取到了 %d 字节的数据 %s", n, c.clientId)
			log.Debugf("[IOServer] 读取到 %s 报文，内容: %v", c.clientId, readBuf[:n])
			c.readBuf = append(c.readBuf, readBuf[:n]...)

			// 判断长度，如果达到8了则生成payload长度
			if len(c.readBuf) >= 8 {
				c.readMsgPayloadLth = common.GenIntFromLength(c.readBuf[0:8])
				log.Debugf("[IOServer] 读取到足够长度的报文，解析得到的报文长度为 %d, 报文长度元数据: %v, 客户端 %s, buf内容: %v", c.readMsgPayloadLth, c.readBuf[0:8], c.clientId, c.readBuf)
			}
		} else if c.readMsgPayloadLth+12 > uint64(len(c.readBuf)) {
			log.Debugf("[IOServer] 报文长度信息已经都读取完成，但是整个报文还没有全部传输 %s。报文长度期望为 %d, 当前buf中的长度为 %d, buf内容: %v", c.clientId, c.readMsgPayloadLth, (len(c.readBuf) - 12), c.readBuf)
			// 长度信息读取到了，但是payload没有读取完成
			readBuf := make([]byte, 256)
			n, err := c.conn.Read(readBuf)
			if err != nil {
				log.Errorf("[IOServer] 从 %s 读取数据时报错，报错内容: %s", c.clientId, err.Error())
				return nil, err
			}
			c.conn.SetReadDeadline(time.Time{})
			log.Debugf("[IOServer] 此次读取到了 %d 字节的数据 %s", n, c.clientId)
			c.readBuf = append(c.readBuf, readBuf[:n]...)
		} else {
			log.Debugf("[IOServer] 报文已经都读取完成 %s", c.clientId)
			c.conn.SetReadDeadline(time.Time{})
			// 整个消息都读取到了，此时c.readBuf中可能包含一个或一个以上的消息内容
			msgLength := 12 + c.readMsgPayloadLth
			msgTypeBytes := c.readBuf[8:12]
			log.Debugf("[IOServer] 报文总长度 %d, 消息类型 %d, 消息体长度 %d - %s", msgLength, common.GenIntFromType(msgTypeBytes), c.readMsgPayloadLth, c.clientId)

			if c.readMsgPayloadLth == 0 {
				// 不存在消息体
				if uint64(len(c.readBuf)) == msgLength {
					log.Debugf("[IOServer] msgLength %d", msgLength)
					log.Debugf("[IOServer] A. 消息接受完成，老的buf内容为: %v", c.readBuf)
					c.readBuf = []byte{}
					log.Debugf("[IOServer] A. 消息接收完成，新的buf内容为: %v", c.readBuf)
				} else {
					log.Debugf("[IOServer] msgLength %d", msgLength)
					log.Debugf("[IOServer] B. 消息接受完成，老的buf内容为: %v", c.readBuf)
					c.readBuf = c.readBuf[msgLength:len(c.readBuf)]
					log.Debugf("[IOServer] B. 消息接收完成，新的buf内容为: %v", c.readBuf)
				}
				// 生成消息
				msg := &msg.Msg{
					Type:     common.GenIntFromType(msgTypeBytes),
					RawDatas: []byte{},
					Msg:      nil,
				}

				c.readTotalBytesLth += msgLength
				c.readMsgPayloadLth = 0
				if len(c.readBuf) >= 8 {
					c.readMsgPayloadLth = common.GenIntFromLength(c.readBuf[0:8])
					log.Debugf("[IOServer] 读取到足够长度的报文，解析得到的报文长度为 %d, 报文长度元数据: %v, 客户端 %s, buf内容: %v", c.readMsgPayloadLth, c.readBuf[0:8], c.clientId, c.readBuf)
				}
				log.Debugf("[IOServer] 当前读取的总长度: %d", c.readTotalBytesLth)
				return msg, nil
			} else {
				// 存在消息体
				msgPayloadBytes := c.readBuf[12:msgLength]
				log.Debugf("[IOServer] msgLength %d", msgLength)
				log.Debugf("[IOServer] C. 消息接收完成，老的buf内容为: %v", c.readBuf)
				c.readBuf = c.readBuf[msgLength:len(c.readBuf)]
				log.Debugf("[IOServer] C. 消息接收完成，新的buf内容为: %v", c.readBuf)

				// 生成消息
				msg := &msg.Msg{
					Type:     common.GenIntFromType(msgTypeBytes),
					RawDatas: msgPayloadBytes,
				}

				c.readTotalBytesLth += msgLength
				c.readMsgPayloadLth = 0
				if len(c.readBuf) >= 8 {
					c.readMsgPayloadLth = common.GenIntFromLength(c.readBuf[0:8])
					log.Debugf("[IOServer] 读取到足够长度的报文，解析得到的报文长度为 %d, 报文长度元数据: %v, 客户端 %s, buf内容: %v", c.readMsgPayloadLth, c.readBuf[0:8], c.clientId, c.readBuf)
				}
				log.Debugf("[IOServer] 当前读取的总长度: %d", c.readTotalBytesLth)
				return msg, nil
			}
		}
	}
}

// 向客户端发送消息，不关心响应
func (c *Client) SendMsg(msg *msg.Msg) {
	if c.state == Erroring {
		log.Warnf("[IOServer] %s 无效，退出发消息循环", c.clientId)
		return
	}

	// 生成MSG
	protobufMsg := []byte{}
	var err error
	if msg.Msg != nil {
		protobufMsg, err = proto.Marshal(msg.Msg)
		if err != nil {
			log.Errorf("[IOServer] protobuf消息生成失败: %s", err.Error())
			return
		}
	}
	// 计算长度等信息
	msgSize := len(protobufMsg)
	msgType := msg.Type
	// 生成结果报文
	packetBuf := &bytes.Buffer{}
	lengthBytes := common.GenLengthFromInt(msgSize)
	packetBuf.Write(lengthBytes[:])
	typeBytes := common.GenTypeFromInt(int(msgType))
	packetBuf.Write(typeBytes[:])
	packetBuf.Write(protobufMsg)
	packet := packetBuf.Bytes()

	log.Debugf("[IOServer] 发送报文，报文长度 %d，类型 %d，消息体长度 %d", msgSize+12, msgType, msgSize)
	log.Debugf("[IOServer] 报文内容: %v", packet)

	c.sendLock.Lock()
	// 设置写入超时
	c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.writeTimeout) * time.Second))
	_, err = c.conn.Write(packet)
	if err != nil {
		log.Errorf("[IOServer] sendMsg失败: %s", err.Error())
		c.state = Erroring
	}
	c.sendLock.Unlock()
}
