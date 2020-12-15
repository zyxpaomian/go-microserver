package dao

import (
	se "microserver/common/error"
	log "microserver/common/formatlog"
	"microserver/common/mysql"
	"microserver/structs"
)

type AgentDAO struct {
}

func (d *AgentDAO) ListAgents() ([]*structs.Agent, error) {
	result := []*structs.Agent{}

	tx := mysql.DB.GetTx()
	if tx == nil {
		return nil, se.New("tx is nil")
	}

	sql := `SELECT id, AGENTIP
			FROM AGENT
			ORDER BY id DESC`
	stmt, err := tx.Prepare(sql)
	if err != nil {
		log.Errorf("ListAgents错误, sql: %s ,错误信息: %s", sql, err.Error())
		tx.Rollback()
		return nil, err
	}
	rows, err := stmt.Query()
	if err != nil {
		log.Errorf("ListAgents错误, sql: %s ,错误信息: %s", sql, err.Error())
		stmt.Close()
		tx.Rollback()
		return nil, err
	}
	for rows.Next() {
		agent := &structs.Agent{}
		err := rows.Scan(&agent.Id, &agent.AgentIp)
		if err != nil {
			log.Errorf("ListAgents错误, sql: %s ,错误信息: %s", sql, err.Error())
			rows.Close()
			stmt.Close()
			tx.Rollback()
			return nil, err
		} else {
			result = append(result, agent)
		}
	}
	rows.Close()
	stmt.Close()
	tx.Commit()

	return result, nil
}
