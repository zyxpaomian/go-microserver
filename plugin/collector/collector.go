package collector

import (
	"github.com/golang/protobuf/proto"
	log "microserver/common/formatlog"
	"microserver/msg"
)

func HandleCollectData(clientId string, agentMsg *msg.Msg) {
	collectMsg := &msg.Collect{}
	if err := proto.Unmarshal(agentMsg.RawDatas, collectMsg); err != nil {
		log.Errorf("[Collect] 解析收集项信息失败 %s, 失败原因 %s", clientId, err.Error())
		return
	}
	log.Debugf("[Collect] 接收到 %s 的收集项，开机时间: %s，cpu架构: %s，Cpu数量: %d, 总内存: %s, 收集时间: %s", clientId, collectMsg.Uptime, collectMsg.Cpuarch, collectMsg.Cpunum, collectMsg.Memtotal, collectMsg.ColTime)
}
