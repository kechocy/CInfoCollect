package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"golang.org/x/sys/windows/registry"
)

func getIPAddresses() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println("【Client】", "获取网络接口失败:", err)
		ips = append(ips, "unknown")
		return ips
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ip4 := ipNet.IP.To4(); ip4 != nil {
					ipStr := ip4.String()
					// 过滤 169.254.x.x
					if strings.HasPrefix(ipStr, "169.254.") {
						continue
					}
					ips = append(ips, ipStr)
				}
			}
		}
	}
	if len(ips) == 0 {
		ips = append(ips, "unknown") // 防止数据库非空字段错误
	}
	return ips
}

func getMACAddresses() []string {
	var macs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println("【Client】", "获取网络接口失败:", err)
		macs = append(macs, "unknown")
		return macs
	}
	for _, iface := range ifaces {
		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // 回环
		}
		if iface.Flags&net.FlagUp == 0 {
			continue // 未连接
		}
		macs = append(macs, mac)
	}
	if len(macs) == 0 {
		macs = append(macs, "unknown") // 防止数据库非空字段错误
	}
	return macs
}

func getPrograms() []string {
	locations := []registry.Key{
		registry.LOCAL_MACHINE,
		registry.CURRENT_USER,
	}
	subPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	var programs []string
	for _, root := range locations {
		for _, path := range subPaths {
			k, err := registry.OpenKey(root, path, registry.READ)
			if err != nil {
				continue
			}
			defer k.Close()

			names, err := k.ReadSubKeyNames(-1)
			if err != nil {
				continue
			}

			for _, name := range names {
				subKey, err := registry.OpenKey(k, name, registry.READ)
				if err != nil {
					continue
				}
				defer subKey.Close()

				displayName, _, err := subKey.GetStringValue("DisplayName")
				if err == nil && displayName != "" {
					programs = append(programs, displayName)
				}
			}
		}
	}
	if len(programs) == 0 {
		programs = append(programs, "unknown") // 防止数据库非空字段错误
	} else {
		sort.Strings(programs) // 按字母顺序排序
	}
	return programs
}

// 收集客户端信息
func collectClientInfo() *ClientInfo {
	var hostId string = "unknown"
	var hostname string = "unknown"
	var osVersion string = "unknown"
	var arch string = ""
	var cpuModel string = "unknown"
	var memV string = "unknown"
	var diskSize string = "unknown"
	// 计算机名、操作系统
	hostInfo, err := host.Info()
	if err != nil {
		fmt.Println("【Client】", "获取客户端信息出错:", err)
	} else {
		hostId = hostInfo.HostID
		hostname = hostInfo.Hostname
		osVersion = hostInfo.Platform
		arch = hostInfo.KernelArch
	}

	// 用户名
	currentUser, err := user.Current()
	if err != nil {
		currentUser = &user.User{Username: "unknown"}
		fmt.Println("【Client】", "获取客户端信息出错:", err)
	}

	// cpu 型号
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	} else {
		fmt.Println("【Client】", "获取客户端信息出错:", err)
	}

	// 内存
	vmem, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("【Client】", "获取客户端信息出错:", err)

	} else {
		memV = fmt.Sprintf("%.2f GB", float64(vmem.Total)/(1<<30))
	}

	// 获取磁盘
	partitions, err := disk.Partitions(false)
	var diskTotal uint64
	if err != nil {
		fmt.Println("【Client】", "获取客户端信息出错:", err)
	} else {
		for _, p := range partitions {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}
			diskTotal += usage.Total
		}
		if diskTotal >= (1 << 40) {
			diskSize = fmt.Sprintf("%.2f TB", float64(diskTotal)/(1<<40))
		} else {
			diskSize = fmt.Sprintf("%.2f GB", float64(diskTotal)/(1<<30))
		}
	}

	client := &ClientInfo{
		HostID:       hostId,
		Hostname:     hostname,
		Username:     currentUser.Username,
		OS:           fmt.Sprintf("%v %v", osVersion, arch),
		CPU:          cpuModel,
		Memory:       memV,
		Disk:         diskSize,
		IPAddresses:  getIPAddresses(),
		MACAddresses: getMACAddresses(),
		Programs:     getPrograms(),
		Updated:      time.Now().Format(time.RFC3339),
	}

	return client
}

// 发送数据
func sendToServer(info *ClientInfo, serverURL string) error {

	if !testServer(serverURL) {
		return fmt.Errorf("无法连接到服务端: %v", serverURL)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("JSON 解析失败: %v", err)
	}

	resp, err := http.Post(serverURL+"/report", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("发送 Post 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送成功，但服务端返回: %v", resp.Status)
	}
	return nil
}

// 测试连接
func testServer(url string) bool {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Get(url)
	if err == nil {
		resp.Body.Close()
	}
	return err == nil
}

// 启动客户端
func startClient(port int, ip string, interval int) {
	log.Println("【Client】", "启动中 ...")
	serverURL := fmt.Sprintf("http://%s:%d", ip, port)

	for {
		// 收集系统信息
		info := collectClientInfo()

		// 发送数据失败并不终止程序
		err := sendToServer(info, serverURL)
		if err != nil {
			log.Println("【Client】", err)
		} else {
			log.Println("【Client】", "发送数据成功")
		}

		// 只执行一次
		if interval == 0 {
			log.Println("【Client】", "执行一次成功，退出程序")
			os.Exit(1)
		}

		// 每 interval 分钟发送一次
		time.Sleep(time.Duration(interval) * time.Minute)

	}
}
