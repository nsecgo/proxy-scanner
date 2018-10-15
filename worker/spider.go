package worker

import (
	"log"
	"net/http"
	"bufio"
	"strings"
	"net"
	"strconv"
	"github.com/PuerkitoBio/goquery"
	"time"
	"github.com/nsecgo/proxy-scanner/models"
	"encoding/json"
	"io/ioutil"
	"sync/atomic"
	"sync"
	"bytes"
	"fmt"
)

//www.proxyrotator.com/free-proxy-list/
func (spd *spider) SpiderProxyrotator() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		for i := 1; i <= 10; i++ {
			doc, err := goquery.NewDocument("http://www.proxyrotator.com/free-proxy-list/" + strconv.Itoa(i) + "/")
			if err != nil {
				log.Println(err)
			} else {
				doc.Find("tbody tr").Each(func(ii int, s *goquery.Selection) {
					td := s.Find("td")
					style, _ := s.Find("style").Html()
					var inline []string
					for _, v := range strings.Split(style, "\n.") {
						if i := strings.Index(v, "{display:inline"); i != -1 {
							inline = append(inline, v[:i])
						}
					}
					if len(inline) == 0 {
						log.Println("proxyrotator网页异常")
						return
					}
					var ip []string
					span, _ := td.Eq(1).Html()
					for _, v := range strings.Split(span, "</") {
						if strings.Contains(v, "none") || strings.Contains(v, ">.") {
							continue
						}
						if i := strings.Index(v, `inline">`); i != -1 {
							ip = append(ip, v[i+8:])
							continue
						}
						if i := strings.Index(v, `class="`); i != -1 {
							l := strings.LastIndex(v, `">`)
							class := v[i+7 : l]
							for _, in := range inline {
								if class == in {
									ip = append(ip, v[l+2:])
									break
								}
							}
						}
					}
					spd.add(strings.Join(ip, "."), "")
				})
			}
		}
		spiderWait.Done()
		time.Sleep(5 * time.Minute)
	}
}

//free-proxy-list.net
func (spd *spider) SpiderFreeProxyList() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		doc, err := goquery.NewDocument("https://free-proxy-list.net/")
		if err != nil {
			log.Println(err)
		} else {
			doc.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
				td := s.Find("td")
				spd.add(td.Eq(0).Text(), td.Eq(1).Text())
			})
		}
		spiderWait.Done()
		time.Sleep(12 * time.Minute)
	}
}

//www.socks-proxy.net
func (spd *spider) SpiderSocksProxyList() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		doc, err := goquery.NewDocument("https://www.socks-proxy.net/")
		if err != nil {
			log.Println(err)
		} else {
			doc.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
				td := s.Find("td")
				spd.add(td.Eq(0).Text(), td.Eq(1).Text())
			})
		}
		spiderWait.Done()
		time.Sleep(9 * time.Minute)
	}
}

//spys.me/proxy.txt
func (spd *spider) SpiderSpysme() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		resp, err := http.Get("http://spys.me/proxy.txt")
		if err != nil {
			log.Println("[ERR]:[GET] http://spys.me/proxy.txt : ", err)
		} else {
			bodyScanner := bufio.NewScanner(resp.Body)
			for bodyScanner.Scan() {
				line := bodyScanner.Text()
				if len(line) == 0 {
					continue
				}
				index := strings.Index(line, " ")
				if index == -1 {
					continue
				}
				addr := strings.Split(line[:index], ":")
				if len(addr) == 2 {
					spd.add(addr[0], addr[1])
				}
			}
			if err := bodyScanner.Err(); err != nil {
				log.Println("[ERR] reading body : ", err)
			}
			resp.Body.Close()
		}
		spiderWait.Done()
		time.Sleep(100 * time.Minute)
	}
}

//gimmeproxy.com/api/getProxy
func (spd *spider) SpiderGimmeproxy() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		resp, err := http.Get("https://gimmeproxy.com/api/getProxy")
		if err != nil {
			log.Println(err)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			} else {
				type proxy struct {
					Ip   string
					Port string
				}
				var p proxy
				if err = json.Unmarshal(body, &p); err != nil {
					log.Println(err)
					return
				}
				spd.add(p.Ip, p.Port)
			}
			resp.Body.Close()
		}
		spiderWait.Done()
		time.Sleep(5 * time.Minute) //Your request count is over the allowed limit of 240.
	}
}

//pubproxy.com/api/proxy?limit=20
func (spd *spider) SpiderPubproxy() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		resp, err := http.Get("http://pubproxy.com/api/proxy?limit=20&format=txt")
		if err != nil {
			log.Println(err)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			} else {
				for _, line := range bytes.Split(body, []byte("\n")) {
					if i := bytes.Split(line, []byte(":")); len(i) == 2 {
						spd.add(string(i[0]), string(i[1]))
					} else {
						fmt.Println(string(line))
					}
				}
			}
			resp.Body.Close()
		}
		spiderWait.Done()
		time.Sleep(30 * time.Second)
	}
}

//导出
func (spd *spider) SpiderExport() {
	for {
		for {
			time.Sleep(time.Minute)
			if atomic.LoadUint32(&spd.count) > 100 {
				break
			}
		}
		exportWait.Add(1)
		spiderWait.Wait()

		fName := models.Write2File4Scan(&spd.ips)

		var port string
		spd.ports.Range(func(key, value interface{}) bool {
			port += strconv.Itoa(int(key.(uint16))) + ","
			return true
		})
		port += "1080,8080,3128,80,8118,53281,42619,808,8081,9797,9999,3129,8888,8088,65103,443,55555,8000,9000,8123,8090" //添加自定义端口，可重复
		go Scanner([]string{"--rate", "5000", "--includefile", fName, "-p", port})
		spd.ips = sync.Map{}
		spd.ports = sync.Map{}
		spd.count = 0
		exportWait.Done()
	}
}

func (spd *spider) SpiderHistoryRst() {
	for {
		time.Sleep(4 * time.Hour)
		t := time.Now().Add(-3 * time.Hour)
		spd.history.Range(func(key, value interface{}) bool {
			if value.(time.Time).Before(t) {
				spd.history.Delete(key)
			}
			return true
		})
	}
}

func (spd *spider) add(ip, port string) {
	if len(ip) < 7 || net.ParseIP(ip) == nil {
		return
	}
	if _, loaded := spd.history.LoadOrStore(ip, time.Now()); !loaded {
		spd.ips.Store(ip, true)
		atomic.AddUint32(&spd.count, 1)
		if port, err := strconv.ParseUint(port, 10, 16); err == nil {
			spd.ports.Store(uint16(port), true)
		}
	}
}
