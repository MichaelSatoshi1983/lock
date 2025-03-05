package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

const (
	bluetoothAddress = "80:04:5F:73:B2:90" // 替换为实际MAC地址
)

var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

func main() {
	logInfo("开始查找蓝牙设备...")
	devicePath := getBluetoothDevicePath()
	if devicePath == "" {
		logError("蓝牙设备未找到，请确认MAC地址是否正确且设备已配对")
		return
	}

	logSuccess("开始监控设备 %s (%s)", bluetoothAddress, devicePath)
	watchBluetoothConnection(devicePath)
}

func getBluetoothDevicePath() string {
	logDebug("执行 bluetoothctl devices 命令")
	cmd := exec.Command("bluetoothctl", "devices")
	output, err := cmd.Output()
	if err != nil {
		logError("执行bluetoothctl失败: %v", err)
		return ""
	}

	logInfo("蓝牙设备列表:")
	logInfo(string(output))

	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, bluetoothAddress) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				mac := strings.ReplaceAll(bluetoothAddress, ":", "_")
				path := fmt.Sprintf("/org/bluez/hci0/dev_%s", mac)
				logSuccess("找到设备路径: %s", path)
				return path
			}
		}
	}
	return ""
}

func watchBluetoothConnection(devicePath string) {
	logInfo("启动dbus-monitor监听...")
	cmd := exec.Command("dbus-monitor", "--system",
		fmt.Sprintf("type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path=%s", devicePath))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logError("创建stdout管道失败: %v", err)
		return
	}
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	var inConnectedEntry bool

	for scanner.Scan() {
		line := scanner.Text()
		logDebug("收到DBus信号: %s", line)

		switch {
		case strings.Contains(line, "string \"Connected\""):
			inConnectedEntry = true
		case inConnectedEntry && strings.Contains(line, "variant"):
			if strings.Contains(line, "boolean false") {
				fmt.Printf(colorRed + "蓝牙断开！" + colorReset + "\n")
				logCritical("检测到蓝牙断开，准备锁定屏幕...")
				lockScreen()
			}
			inConnectedEntry = false
		default:
			inConnectedEntry = false
		}
	}
}

func lockScreen() {
	logCritical("尝试以下锁屏方法:")

	cmds := []struct {
		name string
		args []string
	}{
		{"dbus-send", []string{"--session", "--dest=org.gnome.ScreenSaver",
			"/org/gnome/ScreenSaver", "org.gnome.ScreenSaver.Lock"}}, // <button class="citation-flag" data-index="1"><button class="citation-flag" data-index="3"><button class="citation-flag" data-index="6">
		{"loginctl", []string{"lock-sessions"}},                          // <button class="citation-flag" data-index="5">
		{"dm-tool", []string{"lock"}},                                    // <button class="citation-flag" data-index="7">
		{"swayidle", []string{"timeout", "1", "loginctl lock-sessions"}}, // <button class="citation-flag" data-index="2">
	}

	for _, cmd := range cmds {
		logInfo("尝试执行: %s %s", cmd.name, strings.Join(cmd.args, " "))
		err := exec.Command(cmd.name, cmd.args...).Run()
		if err == nil {
			logSuccess("成功执行: %s", cmd.name)
			return
		}
		logError("命令失败: %v", err)
	}
	logCritical("所有锁屏方法均失败，请检查系统配置")
}

func logInfo(format string, a ...interface{}) {
	fmt.Printf(colorGreen+"[INFO] "+colorReset+format+"\n", a...)
}

func logDebug(format string, a ...interface{}) {
	fmt.Printf(colorBlue+"[DEBUG] "+colorReset+format+"\n", a...)
}

func logWarning(format string, a ...interface{}) {
	fmt.Printf(colorYellow+"[WARNING] "+colorReset+format+"\n", a...)
}

func logError(format string, a ...interface{}) {
	fmt.Printf(colorRed+"[ERROR] "+colorReset+format+"\n", a...)
}

func logCritical(format string, a ...interface{}) {
	fmt.Printf(colorPurple+"[CRITICAL] "+colorReset+format+"\n", a...)
}

func logSuccess(format string, a ...interface{}) {
	fmt.Printf(colorCyan+"[SUCCESS] "+colorReset+format+"\n", a...)
}
