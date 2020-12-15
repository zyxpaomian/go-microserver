package controller

import (
	"microserver/dao"
	"microserver/structs"
)

var Agentctrl *AgentCtrl

type AgentCtrl struct {
	agentDao *dao.AgentDAO
}

func init() {
	Agentctrl = &AgentCtrl{
		agentDao: &dao.AgentDAO{},
	}
}

func (a *AgentCtrl) ListAgents() ([]*structs.Agent, error) {
	return a.agentDao.ListAgents()
}
