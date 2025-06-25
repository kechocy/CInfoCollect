// Copyright 2013 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/getlantern/systray"
	"github.com/lxn/walk"
	"github.com/xuri/excelize/v2"

	d "github.com/lxn/walk/declarative"
)

var serverWin *walk.MainWindow
var allSelected bool = false
var selectedCount int = 0
var allSelectedBtn *walk.PushButton

func startServerGUI(interval int) {
	var resetBtn *walk.PushButton
	var tv *walk.TableView
	var pageEdit *walk.LineEdit
	var sizeEdit *walk.LineEdit
	var pageCountLabel *walk.Label  // 当前页码
	var totalCountLabel *walk.Label // 记录条数
	var prePage *walk.PushButton
	var nextPage *walk.PushButton
	var detailView *walk.TextEdit
	model := NewClientInfoModel(interval)

	// 从 rsrc.syso 中加载图标
	icon, err := walk.NewIconFromResourceId(2)
	if err != nil {
		log.Fatalln("图标加载失败:", err)
	}

	// 创建窗口
	if err := (d.MainWindow{
		AssignTo: &serverWin,
		Title:    "Computer Information Collect",
		Icon:     icon,
		MinSize:  d.Size{Width: 200, Height: 120},
		MenuItems: []d.MenuItem{
			d.Menu{
				Text: "&Help",
				Items: []d.MenuItem{
					d.Action{
						Text: "About",
						OnTriggered: func() {

							message := `【Client】
							%v （默认，启动带托盘的客户端）
							%v -b （静默启动客户端，不显示托盘）
							%v -t （客户端定时上报间隔，单位：分钟）
							%v -p 7890 （指定连接服务端的端口号）
							%v -p 7890 -ip "10.10.10.10" （指定服务端 IP 和端口号）
							【Server】
							%v -s （启动服务端，默认监听 9870 端口）
							%v -s -p 7890 （启动服务端并指定端口号）`
							walk.MsgBox(serverWin, "提示", strings.ReplaceAll(message, "%v", getExecutableName()), walk.MsgBoxIconInformation)
						},
					},
					d.Action{
						Text: "Exit",
						OnTriggered: func() {
							if serverWin != nil {
								serverWin.Dispose()
							}
							systray.Quit()
						},
					},
				},
			},
		},
		Layout: d.VBox{},
		Children: []d.Widget{
			d.Composite{
				Layout: d.HBox{MarginsZero: true},
				Children: []d.Widget{
					d.PushButton{
						AssignTo: &allSelectedBtn,
						Text:     "全选",
						MinSize:  d.Size{Width: 80, Height: 40},
						MaxSize:  d.Size{Width: 80, Height: 40},

						OnClicked: func() {
							if model.RowCount() > 0 {
								allSelected = !allSelected

								// 选中所有行
								if allSelected {
									// indexes := make([]int, rowCount)
									// for i := range rowCount {
									// 	indexes[i] = i
									// }
									// tv.SetSelectedIndexes(indexes)
									selectedCount = model.RowCount()
									allSelectedBtn.SetText("取消全选")

								} else {
									// tv.SetSelectedIndexes(nil)
									selectedCount = 0
									allSelectedBtn.SetText("全选")
								}

								// 勾选复选框
								for i := range model.RowCount() {
									model.items[i].Checked = allSelected
								}

								model.PublishRowsReset()
								tv.Invalidate()
							}
						},
					},
					// d.Label{Text: "", MinSize: d.Size{Width: 40, Height: 0}},
					d.PushButton{
						Text:    "导出",
						MinSize: d.Size{Width: 80, Height: 40},
						MaxSize: d.Size{Width: 80, Height: 40},

						OnClicked: func() {
							f := excelize.NewFile()
							sheet := "Sheet1"
							f.SetSheetName("Sheet1", sheet)

							// 设置表格标题
							for col := range model.ColumnCount() {
								title := model.ColumnName(col)
								cell := fmt.Sprintf("%s1", columnLetter(col))
								f.SetCellValue(sheet, cell, title)
							}
							// 勾选行内容导出
							outputRow := 2 // 表格内容起始行
							for row := range model.RowCount() {
								if checked := model.items[row].Checked; checked {
									for col := range model.ColumnCount() {
										value := model.Value(row, col)
										f.SetCellValue(sheet, fmt.Sprintf("%s%d", columnLetter(col), outputRow), value)
									}
									outputRow++
								}
							}
							if outputRow == 2 {
								walk.MsgBox(serverWin, "提示", "未勾选任何行", walk.MsgBoxIconWarning)
							} else {
								if err := f.SaveAs("导出.xlsx"); err != nil {
									walk.MsgBox(serverWin, "错误", "保存失败: "+err.Error(), walk.MsgBoxIconError)
								} else {
									walk.MsgBox(serverWin, "成功", "导出成功", walk.MsgBoxIconInformation)
								}
							}

						},
					},
					d.HSpacer{}, // 把剩余空间推到右边
					d.PushButton{
						Text:     "重置刷新",
						AssignTo: &resetBtn,
						MinSize:  d.Size{Width: 80, Height: 40},
						MaxSize:  d.Size{Width: 80, Height: 40},

						OnClicked: func() {
							resetBtn.SetEnabled(false)
							// 更新记录总数
							total, err := queryClientInfoTotal()
							model.totalCount = total
							if err != nil {
								log.Println("【Server】", err)
							}

							//更新全局变量
							allSelected = false
							selectedCount = 0
							// 更新页码和页面大小
							model.page = 1
							model.pageSize = 50

							// 更新表格数据
							model.items = nil
							if err := model.loadDataByPage(model.pageSize, 0); err != nil {
								log.Println("【Server】", err)
							}
							// 更新 UI
							prePage.SetEnabled(model.isEnablePrePage())
							nextPage.SetEnabled(model.isEnableNextPage())
							pageEdit.SetText("1")
							sizeEdit.SetText("50")
							pageCountLabel.SetText(fmt.Sprint(getPageCount(model.totalCount, model.pageSize)))
							totalCountLabel.SetText(fmt.Sprint(model.totalCount))
							allSelectedBtn.SetText("全选")
							resetDetailView(detailView)
							model.Sort(0, walk.SortAscending) // 重置排序
							// 强制刷新页面（不能省略，否则全选后再重置，会导致只有鼠标经过第一行时，其选中状态才会取消）
							tv.Invalidate()
							resetBtn.SetEnabled(true)

						},
					},
				},
			},

			d.Composite{
				Layout: d.HBox{MarginsZero: true},
				Children: []d.Widget{
					d.TableView{
						StretchFactor:    3,
						AssignTo:         &tv,
						AlternatingRowBG: true,
						CheckBoxes:       true,
						ColumnsOrderable: true,
						MultiSelection:   true,
						Columns: []d.TableViewColumn{
							{Title: model.ColumnName(0), Width: 50},
							{Title: model.ColumnName(1), Width: 70},
							{Title: model.ColumnName(2), Width: 70},
							{Title: model.ColumnName(3), Width: 70},
							{Title: model.ColumnName(4), Width: 70},
							{Title: model.ColumnName(5), Width: 70},
							{Title: model.ColumnName(6), Width: 60},
							{Title: model.ColumnName(7), Width: 60},
							{Title: model.ColumnName(8), Width: 50},
						},
						StyleCell: func(style *walk.CellStyle) {
							if style.Row() < 0 || style.Row() >= len(model.items) {
								return
							}
							item := model.items[style.Row()]

							if item.Checked {
								if style.Row()%2 == 0 {
									style.BackgroundColor = walk.RGB(159, 215, 255)
								} else {
									style.BackgroundColor = walk.RGB(143, 199, 239)
								}
							}
						},
						Model: model,
					},
					d.TextEdit{
						AssignTo:      &detailView,
						StretchFactor: 2,
						ReadOnly:      true,
						OnMouseMove: func(x, y int, button walk.MouseButton) {
							detailView.SetCursor(walk.CursorArrow()) // 更改鼠标样式
						},
						Text:          "\r\nComputer Information Collect\r\n\r\nVersion 1.0.0\r\n\r\n©2025 Powerd By Kecho\r\n",
						VScroll:       true,
						HScroll:       true,
						Font:          d.Font{Family: "Consolas", PointSize: 14},
						TextAlignment: d.Alignment1D(walk.AlignCenter),
					},
				},
			},
			// 分页组件
			d.Composite{
				Layout: d.HBox{MarginsZero: true},
				Children: []d.Widget{
					d.PushButton{
						Text:     "上一页",
						AssignTo: &prePage,
						Enabled:  model.isEnablePrePage(),
						MinSize:  d.Size{Width: 80, Height: 40},
						MaxSize:  d.Size{Width: 80, Height: 40},
						OnClicked: func() {
							input := model.page
							input--
							// 超出范围修正
							if input < 1 {
								input = 1
							}

							// 更新全局变量
							allSelected = false
							selectedCount = 0
							model.page = input // 更新页码

							// 更新表格数据
							model.items = nil
							if err := model.loadDataByPage(model.pageSize, (model.page-1)*model.pageSize); err != nil {
								log.Println("【Server】", err)
							}
							// 更新 UI（放最后）
							prePage.SetEnabled(model.isEnablePrePage())
							nextPage.SetEnabled(model.isEnableNextPage())
							pageEdit.SetText(strconv.Itoa(input))
							allSelectedBtn.SetText("全选")
							tv.Invalidate()
						},
					},
					d.Label{Text: "第"},
					d.LineEdit{
						AssignTo:      &pageEdit,
						Text:          strconv.Itoa(model.page),
						MinSize:       d.Size{Width: 30, Height: 40},
						MaxSize:       d.Size{Width: 30, Height: 40},
						TextAlignment: d.AlignCenter,
						OnEditingFinished: func() {
							text := strings.TrimSpace(pageEdit.Text())
							input, err := strconv.Atoi(text)
							if err != nil || input < 1 {
								input = 1
							}
							maxPage := getPageCount(model.totalCount, model.pageSize)
							// 超出范围修正
							if input > maxPage {
								input = maxPage
							}

							if input != model.page {
								// 更新全局变量
								allSelected = false
								selectedCount = 0
								model.page = input // 更新 model 中页码

								// 更新表格数据
								model.items = nil
								if err := model.loadDataByPage(model.pageSize, (model.page-1)*model.pageSize); err != nil {
									log.Println("【Server】", err)
								}

								// 更新 UI（放最后）
								pageEdit.SetText(strconv.Itoa(input))
								prePage.SetEnabled(model.isEnablePrePage())
								nextPage.SetEnabled(model.isEnableNextPage())
								allSelectedBtn.SetText("全选")
								tv.Invalidate()
							}

						},
					},
					d.Label{Text: "页 / 共"},
					d.Label{
						AssignTo: &pageCountLabel,
						Text:     fmt.Sprint(getPageCount(model.totalCount, model.pageSize)),
					},
					d.Label{Text: "页"},
					d.Label{Text: "每页"},
					d.LineEdit{
						AssignTo:      &sizeEdit,
						Text:          strconv.Itoa(model.pageSize),
						TextAlignment: d.AlignCenter,
						MinSize:       d.Size{Width: 30, Height: 40},
						MaxSize:       d.Size{Width: 30, Height: 40},
						OnEditingFinished: func() {
							text := strings.TrimSpace(sizeEdit.Text())
							input, err := strconv.Atoi(text)
							if err != nil || input < 1 {
								input = 1
							}
							// 超出范围修正
							if input > 1000 {
								input = 1000
							}

							if input != model.pageSize {
								// 更新全局变量
								allSelected = false
								selectedCount = 0
								model.page = 1         // 更改页大小回到第一页
								model.pageSize = input // 更新 model 中页面大小

								// 更新表格数据
								model.items = nil
								if err := model.loadDataByPage(model.pageSize, (model.page-1)*model.pageSize); err != nil {
									log.Println("【Server】", err)
								}

								// 更新 UI（放最后）
								sizeEdit.SetText(strconv.Itoa(input))
								pageEdit.SetText("1")
								pageCountLabel.SetText(fmt.Sprintf("%v", getPageCount(model.totalCount, model.pageSize)))
								prePage.SetEnabled(false)
								nextPage.SetEnabled(model.isEnableNextPage())
								allSelectedBtn.SetText("全选")
								tv.Invalidate() // 不能省略

							}
						},
					},
					d.Label{Text: "条【共"},
					d.Label{
						AssignTo: &totalCountLabel,
						Text:     fmt.Sprintf("%d", model.totalCount),
					},
					d.Label{Text: "条】"},
					d.PushButton{
						Text:     "下一页",
						AssignTo: &nextPage,
						Enabled:  model.isEnableNextPage(),
						MinSize:  d.Size{Width: 80, Height: 40},
						MaxSize:  d.Size{Width: 80, Height: 40},
						OnClicked: func() {

							input := model.page
							input++
							maxPage := getPageCount(model.totalCount, model.pageSize)
							// 超出范围修正
							if input > maxPage {
								input = maxPage
							}

							// 更新全局变量
							allSelected = false
							selectedCount = 0
							model.page = input // 更新页码

							// 更新表格数据
							model.items = nil
							if err := model.loadDataByPage(model.pageSize, (model.page-1)*model.pageSize); err != nil {
								log.Println("【Server】", err)
							}

							// 更新 UI (放最后)
							prePage.SetEnabled(model.isEnablePrePage())
							nextPage.SetEnabled(model.isEnableNextPage())
							pageEdit.SetText(strconv.Itoa(input))
							allSelectedBtn.SetText("全选")
							tv.Invalidate()
						},
					},
					d.HSpacer{},
				},
			},
			d.Label{
				Text: "©2025 Powerd By Kecho",
			},
		},
	}).Create(); err != nil {
		log.Fatalln("【Server】", "创建窗口失败:", err)
	}

	// 关闭窗口改为隐藏窗口
	serverWin.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		serverWin.Hide()
	})
	// 单机某行自动勾选
	tv.CurrentIndexChanged().Attach(func() {
		row := tv.CurrentIndex()
		if row >= 0 && row < model.RowCount() {
			isChecked := model.items[row].Checked

			model.items[row].Checked = !isChecked
			if isChecked {
				selectedCount--
			} else {
				selectedCount++
			}
			model.PublishRowChanged(row)
		}
		if selectedCount == model.RowCount() && model.RowCount() > 0 {
			allSelected = true
			allSelectedBtn.SetText("取消全选")

		} else {
			allSelected = false
			allSelectedBtn.SetText("全选")
		}
	})
	// 双击某行显示详情
	tv.ItemActivated().Attach(func() {
		row := tv.CurrentIndex()
		if row >= 0 && row < len(model.items) {
			detailView.SetTextAlignment(walk.AlignDefault)
			font, err := walk.NewFont("Consolas", 10, 0)
			if err != nil {
				fmt.Println("【Server】", "详情页面设置字体错误:", err)
			}
			detailView.SetFont(font)

			item := model.items[row]

			var b strings.Builder
			val := reflect.ValueOf(item)
			typ := reflect.TypeOf(item)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
				typ = typ.Elem()
			}
			if val.Kind() != reflect.Struct {
				return
			}
			width := 12
			for i := range val.NumField() {
				field := typ.Field(i)
				if field.PkgPath != "" {
					continue // 跳过私有字段
				}
				if field.Name == "ID" || field.Name == "Checked" || field.Name == "Online" {
					continue
				}
				if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
					// 是 [] string 类型
					slice := val.Field(i).Interface().([]string)
					if len(slice) == 0 {
						fmt.Fprintf(&b, "%-*s: \r\n", width, field.Name)
					} else {
						fmt.Fprintf(&b, "%-*s: %s\r\n", width, field.Name, slice[0]) // 第一行
						indent := strings.Repeat(" ", width+2)                       // 冒号后面空格数，第二行之后行的前缀
						for _, v := range slice[1:] {
							b.WriteString(indent)
							b.WriteString(v)
							b.WriteString("\r\n")
						}
					}
				} else { // string 类型
					v := val.Field(i).Interface()
					fmt.Fprintf(&b, "%-*s: %s\r\n", width, field.Name, v)
				}
			}
			detailView.SetText(b.String())
			// fmt.Sprintf("Hostname: %v\n Username: %v\n OS: %v\n CPU: %v\n Memory: %v\n IP: %v\n Mac: %v\n Program: %v\n", item.Hostname, item.Username, item.OS, item.CPU, item.Memory, item.IPAddresses, item.MACAddresses, item.InstalledPrograms)
		}
	})

	serverWin.Run()
}

// 获取当前应用名称
func getExecutableName() string {
	exePath, err := os.Executable()
	if err != nil {
		return "InfoCollectGUI.exe"
	}
	return filepath.Base(exePath)

}

// 转换为 xlsx 单元格格式，如将 0 转 A, ... , 26 转 AA
func columnLetter(col int) string {
	result := ""
	for col >= 0 {
		result = string(rune('A'+(col%26))) + result
		col = col/26 - 1
	}
	return result
}

// 计算总页数
func getPageCount(total, pageSize int) int {
	if total == 0 {
		return 1
	}
	return (total-1)/pageSize + 1
}

// 上一页按钮状态
func (m *ClientInfoModel) isEnablePrePage() bool {
	if m.page == 1 {
		return false
	} else {
		return true
	}
}

// 下一页按钮状态
func (m *ClientInfoModel) isEnableNextPage() bool {
	maxPage := getPageCount(m.totalCount, m.pageSize)
	if maxPage == 1 {
		return false
	} else {
		if m.page == maxPage {
			return false
		} else {
			return true
		}
	}
}

// 重置详情控件
func resetDetailView(view *walk.TextEdit) {
	font, err := walk.NewFont("Consolas", 14, 0)
	if err != nil {
		fmt.Println("【Server】", "详情页面设置字体错误:", err)
	}
	view.SetFont(font)

	view.SetTextAlignment(walk.AlignCenter)
	view.SetText("\r\nComputer Information Collect\r\n\r\nVersion 1.0.0\r\n\r\n©2025 Powerd By Kecho\r\n")
}

// func loadIconFromEmbed() (*walk.Icon, error) {
// 	tmpFile, err := os.CreateTemp("", "icon-*.ico")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer tmpFile.Close()
// 	if _, err := tmpFile.Write(iconData); err != nil {
// 		return nil, err
// 	}
// 	return walk.NewIconFromFile(tmpFile.Name())
// }
