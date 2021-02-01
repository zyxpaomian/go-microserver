package structs

type Agent struct {
	Id      int64  `json:"id"`
	AgentIp string `json:"agentip"`
}

type Version struct {
	AgentVersion      string  `json:"version"`
	UpdateTime string `json:"updatetime"`
}