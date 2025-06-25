package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDataBase() error {
	var err error = nil
	err = openDatabase()
	if err != nil {
		return fmt.Errorf("数据库连接失败: %v", err)
	}
	err = createTable()
	if err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}
	return err
}

func openDatabase() error {
	var err error = nil
	db, err = sql.Open("sqlite3", "data.db")
	return err
}

func createTable() error {
	sql := `
		CREATE TABLE IF NOT EXISTS client_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id TEXT NOT NULL,
			hostname TEXT NOT NULL,
			username TEXT NOT NULL,
			os TEXT NOT NULL,
			cpu TEXT NOT NULL,
			memory TEXT NOT NULL,
			disk TEXT NOT NULL,
			ip_addresses TEXT NOT NULL,
			mac_addresses TEXT NOT NULL,
			programs TEXT NOT NULL,
			updated TEXT NOT NULL
		);`
	_, err := db.Exec(sql)
	return err
}

// 新增
func insertToDB(data ClientInfo) error {
	stmt, err := db.Prepare(
		`INSERT INTO client_info
			(host_id, hostname, username, os, cpu, memory, disk, ip_addresses, mac_addresses, programs, updated) 
		VALUES 
			(?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	ipJson, _ := json.Marshal(data.IPAddresses)
	macJson, _ := json.Marshal(data.MACAddresses)
	progJson, _ := json.Marshal(data.Programs)

	_, err = stmt.Exec(
		data.HostID,
		data.Hostname,
		data.Username,
		data.OS,
		data.CPU,
		data.Memory,
		data.Disk,
		string(ipJson),
		string(macJson),
		string(progJson),
		data.Updated,
	)

	return err
}

// 更新
func updateToDB(data ClientInfo) error {
	stmt, err := db.Prepare(
		`UPDATE client_info SET
		hostname = ?, username = ?, os = ?, cpu = ?, memory = ?, disk = ?, ip_addresses = ?, mac_addresses = ?, programs = ?, updated = ?
		WHERE host_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	ipJson, _ := json.Marshal(data.IPAddresses)
	macJson, _ := json.Marshal(data.MACAddresses)
	progJson, _ := json.Marshal(data.Programs)

	_, err = stmt.Exec(
		data.Hostname,
		data.Username,
		data.OS,
		data.CPU,
		data.Memory,
		data.Disk,
		string(ipJson),
		string(macJson),
		string(progJson),
		data.Updated,
		data.HostID,
	)

	return err
}

func saveToDB(data ClientInfo) error {
	var exist bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM client_info WHERE host_id = ?)", data.HostID).Scan(&exist)
	if err != nil {
		return err
	}
	if exist {
		return updateToDB(data)
	} else {
		return insertToDB(data)
	}
}

// 分页查询
func queryClientInfoByPage(limit, offset int) ([]ClientInfo, error) {
	query := `SELECT host_id, hostname, username, os, cpu, memory, disk, ip_addresses, mac_addresses, programs, updated
			FROM client_info
			WHERE (updated) IN (
				SELECT MAX(updated)
				FROM client_info
				GROUP BY host_id
			)
			ORDER BY updated DESC
			LIMIT ? OFFSET ?`
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("分页查询失败: %v", err)
	}

	defer rows.Close()

	var clients []ClientInfo

	for rows.Next() {
		var c ClientInfo
		var ip, mac, prog string
		if err := rows.Scan(&c.HostID, &c.Hostname, &c.Username, &c.OS, &c.CPU, &c.Memory, &c.Disk, &ip, &mac, &prog, &c.Updated); err != nil {
			return nil, fmt.Errorf("分页查询解析错误: %v", err)
		}
		// 忽略 json 解析失败错误
		json.Unmarshal([]byte(ip), &c.IPAddresses)
		json.Unmarshal([]byte(mac), &c.MACAddresses)
		json.Unmarshal([]byte(prog), &c.Programs)

		clients = append(clients, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("分页查询遍历错误: %v", err)
	}
	return clients, nil
}

// 查询记录总数
func queryClientInfoTotal() (int, error) {
	var total int = 0
	row := db.QueryRow("SELECT COUNT(DISTINCT host_id) FROM client_info")
	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("查询记录总数解析失败: %v", err)
	}
	return total, nil
}
