package dao

import (
	se "microserver/common/error"
	log "microserver/common/formatlog"
	"microserver/common/mysql"
	"microserver/structs"
)

type AgentDAO struct {
}

// 获取所有Agents信息
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

// 获取Agent最新版本
func (d *AgentDAO) GetAgentVersion() (*structs.Version, error) {
	result := &structs.Version{}
	tx := mysql.DB.GetTx()
	if tx == nil {
		return nil, se.New("tx is nil")
	}

	sql := `select AGENTVERSION, UPDATETIME  from VERSION `
	stmt, err := tx.Prepare(sql)
	if err != nil {
		log.Errorf("ListAgent版本错误, sql: %s ,错误信息: %s", sql, err.Error())
		tx.Rollback()
		return nil, err
	}
	rows, err := stmt.Query()
	if err != nil {
		log.Errorf("ListAgent版本错误, sql: %s ,错误信息: %s", sql, err.Error())
		stmt.Close()
		tx.Rollback()
		return nil, err
	}
	for rows.Next() {
		version := &structs.Version{}
		err := rows.Scan(&version.AgentVersion, &version.UpdateTime)
		if err != nil {
			log.Errorf("ListAgent版本错误, sql: %s ,错误信息: %s", sql, err.Error())
			rows.Close()
			stmt.Close()
			tx.Rollback()
			return nil, err
		} else {
			result = version
		}
	}
	rows.Close()
	stmt.Close()
	tx.Commit()

	return result, nil
}
