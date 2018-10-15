package worker

import (
	"bufio"
	"bytes"
	"github.com/nsecgo/proxy-scanner/models"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func inspector() {
	for {
		addr := <-waitCheckch
		atomic.AddUint32(&WaitCheckCount, ^uint32(0))
		inspectorTaskch <- struct{}{}
		go func() {
			proxyCheck(addr)
			<-inspectorTaskch
		}()
	}
}
func regularCheck() {
	for {
		time.Sleep(1 * time.Hour)
		models.SendToCheck(waitCheckch, &WaitCheckCount)
	}
}
func proxyCheck(addr string) {
	i := strings.Index(addr, "://")
	var tp string
	if i != -1 {
		tp = addr[:i]
		addr = addr[i+3:]
	} else {
		tp = ""
	}

	var ip string
	var port uint16
	if i := strings.Index(addr, ":"); i != -1 {
		p, err := strconv.ParseUint(addr[i+1:], 10, 16)
		if err != nil {
			log.Println(err, addr)
			return
		} else {
			ip = addr[:i]
			port = uint16(p)
		}
	} else {
		log.Println(addr)
		return
	}

	if tp == "" || tp == "http" {
		var httpProxy models.HttpProxy
		httpProxy.Mode = httpMode(addr)
		httpProxy.Connect = https(addr)
		httpProxy.Ip = ip
		httpProxy.Port = port
		//更新到数据库
		httpProxy.InsertOrUpdate(tp == "http")
	}
	if tp == "" || tp == "socks" {
		var socksProxy models.SocksProxy
		socksProxy.Socks4 = socks4(addr)
		socksProxy.Socks4a = socks4a(addr)
		socksProxy.Socks5 = socks5(addr)
		socksProxy.Ip = ip
		socksProxy.Port = port
		//更新到数据库
		socksProxy.InsertOrUpdate(tp == "socks")
	}
}
func httpMode(addr string) models.HttpMode {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return models.None
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write([]byte("GET http://" + ServerAddr + ":" + ServerPort + "/header HTTP/1.0\r\nHost: " + ServerAddr + ":" + ServerPort + "\r\nProxy-Connection: close\r\nConnection: close\r\n\r\n"))
	if err != nil {
		return models.None
	}
	reader := bufio.NewReader(conn)
	for i := 0; i < 6; i++ {
		line, err := reader.ReadString('\n')
		if i := strings.Index(line, "begin#"); i != -1 {
			if l := strings.Index(line, "#end"); l != -1 {
				if m, err := strconv.Atoi(line[i+6 : l]); err == nil {
					return models.HttpMode(m)
				}
			}
			log.Println("return header addr:", addr, line)
			break
		} else if err != nil {
			break
		}
	}
	return models.None
}
func https(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write([]byte("CONNECT www.baidu.com:443 HTTP/1.1\r\nHost: www.baidu.com:443\r\nProxy-Connection: close\r\nConnection: close\r\n\r\n"))
	if err != nil {
		return false
	}
	reader := bufio.NewReader(conn)
	for i := 0; i < 6; i++ {
		line, err := reader.ReadString('\n')
		line = strings.ToLower(line)
		if strings.Contains(line, "established") {
			return true
		} else if err != nil {
			break
		}
	}
	return false
}
func socks4(addr string) bool {
	//conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	//if err != nil {
	//	return false
	//}
	//defer conn.Close()
	//conn.SetDeadline(time.Now().Add(10 * time.Second))
	//conn.Write([]byte{0x4,0x1,})
	//fmt.Println(n, err)
	//buf := make([]byte, 1000)
	//n, err = conn.Read(buf)
	//fmt.Println(11111, buf[:n], n, err)
	//
	//hello = []byte("GET / HTTP/1.1\r\nHost: nsec.ml\r\nConnection: close\r\n\r\n")
	//
	//conn.Write(hello)
	//n, err = conn.Read(buf)
	//fmt.Println(22222, buf[:n], n, err)
	return false
}
func socks4a(addr string) bool {
	return false
}
func socks5(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write([]byte{0x5, 0x1, 0x0})
	if err != nil {
		return false
	}
	buf := make([]byte, 3)
	n, _ := conn.Read(buf)
	if bytes.Equal(buf[:n], []byte{0x5, 0x0}) {
		return true
	}
	return false
}
