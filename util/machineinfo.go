package util

import (
	"net"
	"os"
)

//本地ip
func GetLocalIP() (ip string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		// fmt.Printf("%+v\n", a)
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ip + ipnet.IP.String() + ";"
			}
		}

	}
	return
}

//出口ip
func GetOutboundIP() (ip string, err error) {
	conn, dErr := net.Dial("udp", "8.8.8.8:80")
	if dErr != nil {
		err = dErr
		return
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

func GetHostName() (name string, err error) {
	return os.Hostname()
}

func GetHostLabel() (label string, err error) {
	ip, _ := GetLocalIP()
	n, _ := GetHostName()
	return ip + n, nil
}

// func main() {
// 	ips, _ := GetLocalIP()
// 	fmt.Println(ips)

// 	ops, _ := GetOutboundIP()
// 	fmt.Println(ops)

// 	n, _ := GetHostName()
// 	fmt.Println(n)
// }
