package handle

import (
	"encoding/json"
	"io/ioutil"
	"microserver/common"
	log "microserver/common/formatlog"
	"microserver/controller"
	"net/http"
)

// 测试API
func apiTestapi(res http.ResponseWriter, req *http.Request) {
	type Request struct {
		A string `json:"a"`
		B string `json:"b"`
	}

	reqContent, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		log.Errorf("[http] 请求报文解析失败")
		common.ReqBodyInvalid(res)
		return
	}

	request := &Request{}
	if err := common.ParseJsonStr(string(reqContent), request); err != nil {
		log.Errorln("[http] 解析模板JSON失败")
		common.ResMsg(res, 400, err.Error())
		return
	}

	type Response struct {
		Ab string `json:"ab"`
	}

	response := &Response{Ab: "test"}
	result, err := json.Marshal(response)
	if err != nil {
		log.Errorf("[http] testapi JSON生成失败, %v", err.Error())
		common.ResMsg(res, 500, err.Error())
		return
	}
	common.ResMsg(res, 200, string(result))
}

// 获取所有agents
func apiGetAllAgents(res http.ResponseWriter, req *http.Request) {
	agents, err := controller.Agentctrl.ListAgents()
	if err != nil {
		log.Errorf("[http] apiGetAllAgents 数据处理失败, %v", err.Error())
		common.ResMsg(res, 500, err.Error())
		return
	}

	b, err := json.Marshal(agents)
	if err != nil {
		log.Errorf("[http] apiGetAllAgents JSON生成失败, %v", err.Error())
		common.ResMsg(res, 400, err.Error())
		return
	}
	common.ResMsg(res, 200, string(b))
}

// 获取所有agents数量
func apiGetAllAgentsNum(res http.ResponseWriter, req *http.Request) {
	agents, err := controller.Agentctrl.ListAgents()
	if err != nil {
		log.Errorf("[http] apiGetAllAgents 数据处理失败, %v", err.Error())
		common.ResMsg(res, 500, err.Error())
		return
	}

	a := make(map[string]int)
	a["agentsnum"] = len(agents)

	b, err := json.Marshal(a)
	if err != nil {
		log.Errorf("[http] apiGetAllAgents JSON生成失败, %v", err.Error())
		common.ResMsg(res, 400, err.Error())
		return
	}
	common.ResMsg(res, 200, string(b))
}
