package handle

import (
	"microserver/http"
	//go_http "net/http"
)

func InitHandle(r *http.WWWMux) {
	// api相关的接口
	initAPIMapping(r)
}

func initAPIMapping(r *http.WWWMux) {
	// 测试
	r.RegistURLMapping("/v1/api/test", "POST", apiTestapi)
	// 获取所有Agents
	r.RegistURLMapping("/v1/api/listagents", "GET", apiGetAllAgents)
	// 获取所有Agents数量，用于Agent选举Server
	r.RegistURLMapping("/v1/api/listagentsnum", "GET", apiGetAllAgentsNum)

}
