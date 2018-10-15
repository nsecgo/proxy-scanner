package main

import (
	_ "net/http/pprof"
	"github.com/nsecgo/proxy-scanner/worker"
	"net/http"
	"strings"
	"encoding/json"
	"github.com/nsecgo/proxy-scanner/models"
	"strconv"
	"html/template"
	"log"
	"flag"
)

type status struct {
	ScannerTaskStat map[string][]interface{}
	WaitCheckCount  uint32
}
func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	serverAddr := flag.String("l", "", "The External `Address`")
	serverPort := flag.String("p", "80", "The Listen `Port`")
	crawl := flag.Bool("c", false, "Crawl free proxy?")
	dsn := flag.String("dsn", "root@unix(/var/run/mysqld/mysqld.sock)/proxy?parseTime=true&loc=Asia%2FShanghai", "[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]")

	flag.Parse()
	worker.ServerAddr = *serverAddr
	worker.ServerPort = *serverPort
	models.Init(*dsn)

	if worker.ServerAddr == "" {
		log.Fatal("The External Address should not be empty!")
	}
	worker.Run(*crawl)

	http.HandleFunc("/header", func(w http.ResponseWriter, r *http.Request) {
		conn, bufrw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		var proxyMode models.HttpMode
		forward := r.Header.Get("X-Forwarded-For")
		via := r.Header.Get("Via")
		if len(forward) == 0 && len(via) == 0 {
			proxyMode = models.EliteHttp
		} else if strings.Contains(forward, worker.ServerAddr) || strings.Contains(via, worker.ServerAddr) {
			proxyMode = models.TransparentHttp
		} else if strings.Contains(forward, r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]) {
			proxyMode = models.AnonymousHttp
		} else {
			proxyMode = models.DistortingHttp
		}
		j, _ := json.Marshal(r.Header)
		//Server: begin#ProxyMode#end
		mode := strconv.Itoa(int(proxyMode))
		bufrw.Write(append([]byte("HTTP/1.0 200 OK\r\n"+
			"Warning: begin#"+ mode+ "#end\r\n"+
			"Set-Cookie: m=begin#"+ mode+ "#end\r\n\r\n"), j...))
		bufrw.Flush()
	})
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.ParseFiles("views/status.html")
		var s = make(map[string][]interface{})
		worker.ScannerTaskStat.Range(func(key, value interface{}) bool {
			s[key.(string)] = value.([]interface{})
			return true
		})
		err := t.Execute(w, status{ScannerTaskStat: s, WaitCheckCount: worker.WaitCheckCount})
		if err != nil {
			log.Println(err)
		}
	})
	http.HandleFunc("/addtoscan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			go worker.Scanner([]string{r.PostFormValue("ips"), "-p", r.PostFormValue("ports"), "--rate", "5000"})
			http.Redirect(w, r, "/status", 302)
			return
		}
		t, _ := template.ParseFiles("views/addtoscan.html")
		t.Execute(w, nil)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Coming soon"))
	})
	http.ListenAndServe(":"+worker.ServerPort, nil)
}
