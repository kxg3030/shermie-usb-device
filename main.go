package main

import (
	"fmt"
	"github.com/kxg3030/shermie-driver-proxy/service"
)

func main() {
	x, _ := service.GetDeviceList()
	device, err := service.GetUsbDevice(x)
	fmt.Println(device, err)
}
