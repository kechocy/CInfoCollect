package main

import (
	"flag"
)

func main() {
	logFile := initLogger()
	defer logFile.Close()

	isServer := flag.Bool("s", false, "启动服务端")
	isBackground := flag.Bool("b", false, "后台静默启动")
	port := flag.Int("p", 9870, "监听端口")
	serverIP := flag.String("ip", "collect.example.com", "服务端IP")
	interval := flag.Int("t", 2, "定时上报间隔（分钟）0 表示只执行一次")
	flag.Parse()

	if *isServer {
		startServerWithTray(*port, *interval)
	} else {
		if *isBackground {
			startClient(*port, *serverIP, *interval)
		} else {
			startClientWithTray(*port, *serverIP, *interval)
		}
	}

}
