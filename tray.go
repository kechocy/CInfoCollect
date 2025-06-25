package main

import (
	_ "embed"
	"fmt"
	"log"
	"syscall"

	"github.com/getlantern/systray"
)

//go:embed icon.ico
var iconData []byte

func startServerWithTray(port, interval int) {
	go startServer(port, interval)
	systray.Run(onServerReady, onServerExit)

}
func startClientWithTray(port int, ip string, interval int) {
	go startClient(port, ip, interval)
	systray.Run(onClientReady, onClientExit)
}

func onClientReady() {
	onReady("Client", "计算机信息收集中")
}
func onServerReady() {
	onReady("Server", "计算机信息收集服务运行中")
}
func onClientExit() {
	onExit("Client")
}
func onServerExit() {
	if db != nil {
		db.Close()
	}
	onExit("Server")

}
func onReady(tp string, title string) {
	// iconData, err := os.ReadFile("icon.ico")
	// if err != nil {
	// 	log.Printf("【%v】 图标加载失败: %v\n", tp, err)
	// } else {
	// 	systray.SetIcon(iconData)
	// }
	systray.SetIcon(iconData)
	systray.SetTitle(fmt.Sprintf("Computer Information Collect %v", tp))
	systray.SetTooltip(title)

	var mShow *systray.MenuItem

	if tp == "Server" {
		mShow = systray.AddMenuItem("显示窗口", "显示窗口")
	}
	mQuit := systray.AddMenuItem("退出", "退出程序")
	// 等待菜单点击
	go func() {
		if tp == "Server" {
			for {
				select {
				case <-mShow.ClickedCh:
					if serverWin != nil {
						// serverWin.Show()
						// serverWin.SetFocus()
						// serverWin.Activate()
						restoreAndActivate(syscall.Handle(serverWin.Handle()))
					}
				case <-mQuit.ClickedCh:
					if serverWin != nil {
						serverWin.Dispose()
					}
					systray.Quit()
					return
				}
			}
		}
		if tp == "Client" {
			for range mQuit.ClickedCh {
				if serverWin != nil {
					serverWin.Dispose()
				}
				systray.Quit()
			}
		}

	}()
}

func onExit(tp string) {
	log.Printf("【%v】 程序退出\n", tp)
}
