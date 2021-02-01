package handle

import (
	"microserver/http"
	//go_http "net/http"
)

func InitHandle(r *http.WWWMux) {
	// api相关的接口
	initAPIMapping(r)
	initPackageMapping(r)
}

func initAPIMapping(r *http.WWWMux) {
	// 测试
	r.RegistURLMapping("/v1/api/test", "POST", apiTestapi)
	// 获取所有Agents
	r.RegistURLMapping("/v1/api/listagents", "GET", apiGetAllAgents)
	// 获取所有Agents数量，用于Agent选举Server
	r.RegistURLMapping("/v1/api/listagentsnum", "GET", apiGetAllAgentsNum)
	// 获取当前agent的最新版本，用于自动更新
	r.RegistURLMapping("/v1/api/agentlastversion", "GET", apiGetAgentLastestVersion)	
	// 发送广播报文，让Agent 开启更新自检
	r.RegistURLMapping("/v1/api/updatebroadcast", "POST", apiBroadCastUpdate)	
}

func initPackageMapping(r *http.WWWMux) {
	// 获取当前新版本的agent
	r.RegistURLMapping("/v1/package/newrinckagent", "GET", packageNewAgent)
}
