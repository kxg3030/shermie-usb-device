package service

import (
	"bytes"
	"fmt"
	"github.com/gentlemanautomaton/volmgmt/volume"
	"github.com/gookit/goutil/arrutil"
	"golang.org/x/sys/windows"
	"strings"
	"unsafe"
)

// 进程快照
func GetProcessSnapshot() error {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, uint32(0))
	if err != nil {
		return fmt.Errorf("获取进程快照句柄错误：%w", err)
	}
	defer func() {
		_ = windows.CloseHandle(handle)
	}()
	processEntry := windows.ProcessEntry32{
		Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{})),
	}
	err = windows.Process32First(handle, &processEntry)
	if err != nil {
		return fmt.Errorf("获取首个进程快照错误：%w", err)
	}
	for {
		err = windows.Process32Next(handle, &processEntry)
		fmt.Println(processEntry.ProcessID)

		if err != nil {
			break
		}
	}
	return nil
}

// 枚举窗口
func EnumWindows() error {
	dll, err := windows.LoadDLL("user32.dll")
	if err != nil {
		return fmt.Errorf("加载user32错误：%w", err)
	}
	proc, err := dll.FindProc("GetWindowTextW")
	if err != nil {
		return fmt.Errorf("加载函数错误：%w", err)
	}

	callback := func(handle windows.HWND, lParam uintptr) uintptr {
		var preId uint32 = 0
		windowText := make([]uint16, 256)
		processId, _ := windows.GetWindowThreadProcessId(handle, &preId)
		_, _, _ = proc.Call(uintptr(handle), uintptr(unsafe.Pointer(&windowText[0])), uintptr(len(windowText)))
		fmt.Println(processId, windows.UTF16ToString(windowText))
		return uintptr(1)
	}
	err = windows.EnumWindows(windows.NewCallback(callback), unsafe.Pointer(nil))
	if err != nil {
		return err
	}
	return nil
}

// 获取驱动
func GetDeviceList() ([]string, error) {
	device := make([]uint16, 1024)
	num, err := windows.GetLogicalDriveStrings(uint32(len(device)), &device[0])
	if err != nil {
		return nil, fmt.Errorf("获取驱动列表错误：%w", err)
	}
	device = device[:num]
	deviceByte := arrutil.Map[uint16, byte](device, func(value uint16) (val byte, find bool) {
		return byte(value), true
	})
	deviceByteArr := bytes.Split(deviceByte, []byte{0})
	// 判断是不是u盘
	deviceVolume := arrutil.Map[[]byte, string](deviceByteArr, func(obj []byte) (val string, find bool) {
		if len(obj) <= 0 {
			return "", false
		}
		return "\\\\.\\" + strings.Replace(string(obj), "\\", "", -1), true
	})
	return deviceVolume, nil
}

// 获取u盘
func GetUsbDevice(deviceList []string) ([]string, error) {
	removeDevice := make([]string, 0)
	for i := 0; i < len(deviceList); i++ {
		// UTF-8字符串转UTF-16
		utf16Byte, err := windows.UTF16FromString(deviceList[i])
		if err != nil {
			return removeDevice, fmt.Errorf("转换设备%s错误：%w", deviceList[i], err)
		}
		handle, err := windows.CreateFile(&utf16Byte[0], windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.WinNullSid, windows.Handle(0))
		if handle == windows.InvalidHandle || err != nil {
			_ = windows.CloseHandle(handle)
			return removeDevice, fmt.Errorf("打开设备%s错误：%w", deviceList[i], err)
		}
		volumeInfo, err := volume.New(deviceList[i])
		if err != nil {
			return removeDevice, fmt.Errorf("获取设备%s错误：%w", deviceList[i], err)
		}
		if volumeInfo.RemovableMedia() {
			removeDevice = append(removeDevice, deviceList[i])
		}
	}
	return removeDevice, nil
}

// 移除u盘
func RemoveDevice(device string) (bool, error) {
	device = "\\\\.\\" + device
	volumeInfo, err := volume.New(device)
	if err != nil {
		return false, fmt.Errorf("读取设备%s错误：%w", device, err)
	}
	if !volumeInfo.RemovableMedia() {
		return false, fmt.Errorf("设备%s不是U盘", device)
	}
	deviceByte, err := volumeInfo.DeviceID()
	if err != nil {
		return false, fmt.Errorf("读取设备%sID错误：%w", device, err)
	}
	err = windows.SetupDiDestroyDeviceInfoList(windows.DevInfo(deviceByte[0]))
	if err != nil {
		return false, err
	}
	return false, nil
}
