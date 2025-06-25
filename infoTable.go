package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
)

type ClientInfoTable struct {
	ID           int
	HostID       string
	Hostname     string
	Username     string
	OS           string
	CPU          string
	Memory       string
	Disk         string
	IPAddresses  []string
	MACAddresses []string
	Programs     []string
	Updated      string
	Checked      bool
	Online       bool
}
type ClientInfoModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*ClientInfoTable
	colNames   []string
	page       int
	pageSize   int
	totalCount int
	interval   int
}

func NewClientInfoModel(interval int) *ClientInfoModel {
	m := new(ClientInfoModel)
	m.interval = interval
	m.pageSize = 50 // 初始页面大小为 50
	m.page = 1
	m.colNames = []string{"ID", "HostID", "Hostname", "Username", "OS", "CPU", "Memory", "Disk", "Online"}
	m.items = make([]*ClientInfoTable, 0)
	total, err := queryClientInfoTotal()
	m.totalCount = total
	if err != nil {
		log.Println("【Server】", err)
	}

	if err := m.loadDataByPage(m.pageSize, 0); err != nil {
		log.Println("【Server】", err)
	}

	return m
}

func (m *ClientInfoModel) loadDataByPage(limit, offset int) error {
	total := m.totalCount
	if total == 0 {
		m.PublishRowsReset()
		return fmt.Errorf("数据总数为 0")
	}

	clients, err := queryClientInfoByPage(limit, offset)

	if err != nil {
		return err
	}

	if len(clients) == 0 {
		return fmt.Errorf("limit %v offset %v 时数据记录为空", limit, offset)
	}

	for i := range len(clients) {
		lastReport, err := time.Parse(time.RFC3339, clients[i].Updated)
		if err != nil {
			return fmt.Errorf("更新时间解析失败: %v", err)
		}
		var interval int = 1
		if m.interval >= 1 {
			interval = m.interval
		}
		online := time.Since(lastReport) <= time.Duration(interval)*time.Minute
		m.items = append(m.items, &ClientInfoTable{ //append 会自动扩容
			ID:           i + 1,
			HostID:       clients[i].HostID,
			Hostname:     clients[i].Hostname,
			Username:     clients[i].Username,
			OS:           clients[i].OS,
			CPU:          clients[i].CPU,
			Memory:       clients[i].Memory,
			Disk:         clients[i].Disk,
			IPAddresses:  clients[i].IPAddresses,
			MACAddresses: clients[i].MACAddresses,
			Programs:     clients[i].Programs,
			Updated:      clients[i].Updated,
			Online:       online,
		})
	}
	m.PublishRowsReset()
	return nil

}

func (m *ClientInfoModel) ColumnName(col int) string {

	if col >= 0 && col < len(m.colNames) {
		return m.colNames[col]
	}
	return ""
}
func (m *ClientInfoModel) ColumnCount() int {
	return len(m.colNames)
}

// Called by the TableView from SetModel and every time the model publishes a RowsReset event.
func (m *ClientInfoModel) RowCount() int {
	return len(m.items)
}

// Called by the TableView when it needs the text to display for a given cell.
func (m *ClientInfoModel) Value(row, col int) any {
	item := m.items[row]

	switch col {
	case 0:
		return item.ID
	case 1:
		return item.HostID
	case 2:
		return item.Hostname
	case 3:
		return item.Username
	case 4:
		return item.OS
	case 5:
		return item.CPU
	case 6:
		return item.Memory
	case 7:
		return item.Disk
	case 8:
		if item.Online {
			return "On"
		} else {
			return "Off"
		}
	}

	panic("unexpected col")
}

// Called by the TableView to retrieve if a given row is checked.
func (m *ClientInfoModel) Checked(row int) bool {
	return m.items[row].Checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *ClientInfoModel) SetChecked(index int, checked bool) error {
	m.items[index].Checked = checked
	if checked {
		selectedCount++
	} else {
		selectedCount--
	}
	if selectedCount == m.RowCount() && m.RowCount() > 0 {
		allSelected = true
		allSelectedBtn.SetText("取消全选")

	} else {
		allSelected = false
		allSelectedBtn.SetText("全选")
	}
	return nil
}

// Called by the TableView to sort the model.
func (m *ClientInfoModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]

		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}

			return !ls
		}

		switch m.sortColumn {
		case 0:
			return c(a.ID < b.ID)

		case 1:
			return c(a.HostID < b.HostID)

		case 2:
			return c(a.Hostname < b.Hostname)
		case 3:
			return c(a.Username < b.Username)
		case 4:
			return c(a.OS < b.OS)
		case 5:
			return c(a.CPU < b.CPU)
		case 6:
			return c(parseSizeToGB(a.Memory) < parseSizeToGB(b.Memory))
		case 7:
			return c(parseSizeToGB(a.Disk) < parseSizeToGB(b.Disk))
		case 8:
			return c(a.Online && !b.Online)
		}

		panic("unreachable")
	})

	return m.SorterBase.Sort(col, order)
}

// 容量单位转化为 GB 方便排序
func parseSizeToGB(s string) float64 {
	parts := strings.Fields(s) // "8.25 GB" -> ["8.25", "GB"]
	if len(parts) != 2 {
		return 0
	}
	val, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(parts[1]) {
	case "TB":
		return val * 1024
	case "GB":
		return val
	case "MB":
		return val / 1024
	case "KB":
		return val / 1024 / 1024
	}
	return val
}
