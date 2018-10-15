package worker

//www.89ip.cn/api/
func (spd *spider) Spider89ip() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		resp, err := http.Get("http://www.89ip.cn/apijk/?&tqsl=300&sxa=&sxb=&tta=&ports=&ktip=&cf=1")
		if err != nil {
			log.Println("[ERR]:[GET] http://www.89ip.cn/api/ : ", err)
		} else {
			bodyScanner := bufio.NewScanner(resp.Body)
			for bodyScanner.Scan() {
				line := bodyScanner.Text()
				if len(line) > 1000 {
					addrs := strings.Split(line, "<BR>")
					if index := strings.Index(addrs[len(addrs)-1], "<br>"); index > 0 {
						addrs[len(addrs)-1] = addrs[len(addrs)-1][:index]
					}
					for _, addr := range addrs {
						addr := strings.Split(addr, ":")
						spd.add(addr[0], addr[1])
					}
					break
				}
			}
			if err := bodyScanner.Err(); err != nil {
				log.Println("[ERR] reading body : ", err) //unexpected EOF
			}
			resp.Body.Close()
		}
		spiderWait.Done()
		time.Sleep(10 * time.Minute)
	}
}

//www.xicidaili.com
func (spd *spider) SpiderXicidaili() {
	for {
		exportWait.Wait()
		spiderWait.Add(1)
		req, _ := http.NewRequest("GET", "http://www.xicidaili.com/", nil)
		req.Header.Add("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36")
		resp, _ := http.DefaultClient.Do(req)
		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			log.Println(err)
		} else {
			doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
				td := s.Find("td")
				spd.add(td.Eq(1).Text(), td.Eq(2).Text())
			})
		}
		spiderWait.Done()
		time.Sleep(10 * time.Minute)
	}
}
