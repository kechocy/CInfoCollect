package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func startServer(port, interval int) {
	log.Println("【Server】", "启动中 ...")
	err := initDataBase()
	if err != nil {
		log.Fatalln("【Server】", err)
	}
	http.HandleFunc("/report", handleReport)

	addr := fmt.Sprintf(":%d", port)
	log.Println("【Server】", "服务监听端口:", addr[1:])
	// 并发启动
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalln("【Server】", "服务启动失败:", err)
		}
	}()
	go startServerGUI(interval)

}

// 对接收到的数据处理
func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}
	var data ClientInfo
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("【Server】", "JSON 解析错误:", err)
		http.Error(w, "无效 JSON", http.StatusBadRequest)
		return
	}
	// 保存到数据库
	err = saveToDB(data)
	if err != nil {
		log.Println("【Server】", "保存数据失败:", err)
	} else {
		log.Printf("【Server】 收到一条来自 %v 的数据\n", data.Hostname)
	}
	// 即使数据库保存失败也要正确返回
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
