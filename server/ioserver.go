package server

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	cfg "microserver/common/configparse"
	se "microserver/common/error"
	log "microserver/common/formatlog"
	"microserver/controller"
	"microserver/msg"
	"microserver/plugin/collector"
	"net"
	"strings"
	"sync"
	"time"
)

var Ioserver *IoServer

const (
	Waiting = iota
	Running
	Erroring
)

type IoServer struct {
	clientsLock *sync.RWMutex      // 客户端列表锁
	clients     map[string]*Client // 客户端列表
}

// Server初始化
func (s *IoServer) Init() {
	log.Infoln("[IOServer] 初始化Agent 对象....")
	Ioserver = &IoServer{
		clientsLock: &sync.RWMutex{},
		clients:     map[string]*Client{},
	}
}

func (s *IoServer) backgroundService() {
	// 启动agent存活检查
	go s.agentAliveCheck()

	select {}
}

func (s *IoServer) agentAliveCheck() {
	for {
		// 延迟启动，避免误报
		time.Sleep(time.Duration(5) * time.Second)

		curAgents := s.ListAliveAcgents()
		dbAgents, err := controller.Agentctrl.ListAgents()
		if err != nil {
			log.Errorf("获取数据库Agents 清单失败, 错误原因: %s", err.Error())
			continue
		}
		for _, dbAgent := range dbAgents {
			dbAgentIp := dbAgent.AgentIp
			agentState := Erroring
			for _, curAgent := range curAgents {
				if dbAgentIp == curAgent {
					agentState = Running
					break
				}
			}
			if agentState == Erroring {
				log.Errorf("[IOServer] Agent: %s 状态错误, 请检查", dbAgentIp)
			}
		}
		time.Sleep(time.Duration(20) * time.Second)
	}
}

// 启动服务端
func (s *IoServer) Run() {
	// 启动后台服务
	go s.backgroundService()

	l, err := net.Listen("tcp", cfg.GlobalConf.GetStr("common", "svraddr"))
	if err != nil {
		panic(err)
	}
	log.Infof("[IOServer] IO服务启动，监听地址: %s", cfg.GlobalConf.GetStr("common", "svraddr"))
	defer l.Close()

	for {
		conn, err := l.Accept()

		if err != nil {
			log.Errorf("[IOServer] IO服务accept异常: %s", err.Error())
			continue
		}
		log.Debugf("[IOServer] IO服务收到连接请求，对端 -> 本端信息: %s -> %s", conn.RemoteAddr(), conn.LocalAddr())

		// 每次都启动一个专门的协程用于检查请求
		go s.handleConnection(conn)
	}
}

// 获取连接对端的唯一ID
func (s *IoServer) getConnId(conn net.Conn) (string, error) {
	addr := conn.RemoteAddr().String()
	addrs := strings.Split(addr, ":")
	if len(addrs) != 2 {
		log.Errorf("[IOServer] 对端地址信息异常: %s", addr)
		return "", se.New(fmt.Sprintf("对端地址信息异常: %s", addr))
	}
	return strings.TrimSpace(addrs[0]), nil
}

// 处理连接请求
func (s *IoServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// 获取对端ID，一般就是对端的IP信息
	clientId, err := s.getConnId(conn)
	if err != nil {
		log.Errorf("[IOServer] 获取AgentID错误: %v", err.Error())
		return
	}

	// 判断conn是否是新的client，是的话就新增，否则结束连接，确保每个链接是唯一的
	s.clientsLock.Lock()
	if _, ok := s.clients[clientId]; ok {
		log.Warnf("[IOServer] 对端已经连接，因此当前连接被忽略: %s", conn.RemoteAddr())
		s.clientsLock.Unlock()
		return
	}
	client := NewClient(conn, clientId)
	s.clients[clientId] = client
	s.clientsLock.Unlock()

	// 交互
	// 当前协程会负责所有的从client的read的请求。
	// 对client的write的请求由http server或者其它模块产生的协程负责
	for {
		// 完整的读取一个msg
		log.Debugf("[IOServer] 开始从客户端 %s 读取消息", client.clientId)
		// 判断下心跳包的超时问题
		if !client.Valid() {
			client.state = Erroring
		}
		msg, err := client.GetMsg()
		if err != nil {
			log.Errorf("[IOServer] 从客户端 %s 获取消息失败，结束与该客户端的连接，报错内容: %s", client.clientId, err.Error())
			// 从server端移除client
			s.clientsLock.Lock()
			client.state = Erroring
			if _, ok := s.clients[client.clientId]; ok {
				log.Infof("[IOServer] 移除客户端: %s", client.clientId)
				delete(s.clients, client.clientId)
			}
			s.clientsLock.Unlock()
			break
		}
		// 处理消息
		go s.handleMsg(msg, client)
	}
}

func (s *IoServer) handleMsg(agentMsg *msg.Msg, client *Client) {
	switch agentMsg.Type {
	case msg.CLIENT_MSG_HEARTBEAT:
		s.handleClientHeartbeatMsg(agentMsg, client)
	case msg.CLIENT_MSG_COLLECT:
		collector.HandleCollectData(client.clientId, agentMsg)
	default:
		log.Errorf("[IOServer] 未知的消息类型, %s: %d", client.clientId, agentMsg.Type)
	}
}

func (s *IoServer) handleClientHeartbeatMsg(agentMsg *msg.Msg, client *Client) {
	heartbeatMsg := &msg.Heartbeat{}
	if err := proto.Unmarshal(agentMsg.RawDatas, heartbeatMsg); err != nil {
		log.Errorf("[IOServer] 解析心跳信息失败 %s, 失败原因 %s", client.clientId, err.Error())
		return
	}
	log.Debugf("[IOServer] 接收到 %s 的心跳请求，心跳包时间 %s，心跳包状态 %s", client.clientId, heartbeatMsg.HeartbeatTime, heartbeatMsg.Status)
	client.SetLastHeartbeatSyncTime(heartbeatMsg.HeartbeatTime)
	// 返回响应报文，确保客户端读取不要超时
	heartbeatResponseMsg := &msg.Msg{
		Type: msg.SERVER_MSG_HEARTBEAT_RESPONSE,
		Msg:  nil,
	}
	client.SendMsg(heartbeatResponseMsg)
}

func (s *IoServer) broadcast(msg *msg.Msg) {
	s.clientsLock.Lock()
	for _, c := range s.clients {
		go c.SendMsg(msg)
	}
	s.clientsLock.Unlock()
}

func (s *IoServer) ListAliveAcgents() []string {
	agents := []string{}
	s.clientsLock.Lock()
	for k, _ := range s.clients {
		agents = append(agents, k)
	}
	s.clientsLock.Unlock()
	return agents
}

func (s *IoServer) BroadcastUpdate() {
	log.Infoln("[IOServer] 服务端开始通知agent进行升级")
	updateMsg := &msg.Msg{
		Type: msg.SERVER_MSG_AGENT_UPDATE,
		Msg: &msg.UpdateMsg{
			Updateswitch: true,
		},
	}
	s.broadcast(updateMsg)
}