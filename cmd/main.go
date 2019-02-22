package main

import (
	"flag"
	"github.com/nsecgo/proxy-scanner/controllers"
	"github.com/nsecgo/proxy-scanner/models"
	"github.com/nsecgo/proxy-scanner/worker"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	serverAddr := flag.String("l", "", "The External `Address`")
	serverPort := flag.String("p", "80", "The Listen `Port`")
	grab := flag.Bool("c", false, "Grab free proxy?")
	//file:proxy.db?cache=shared
	db := flag.String("db", "mysql", "database type(sqlite or mysql)")
	dsn := flag.String("dsn", "root@unix(/var/run/mysqld/mysqld.sock)/proxy?parseTime=true&loc=Asia%2FShanghai", "[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]")

	flag.Parse()
	worker.ServerAddr = *serverAddr
	worker.ServerPort = *serverPort
	models.Init(*db, *dsn)

	if worker.ServerAddr == "" {
		log.Fatal("The External Address should not be empty!")
	}
	worker.Run(*grab)

	http.HandleFunc("/", controllers.Root)
	http.HandleFunc("/status", controllers.Status)
	http.HandleFunc("/addtoscan", controllers.AddToScan)
	http.HandleFunc("/header", controllers.Header)
	log.Fatal(http.ListenAndServe(":"+worker.ServerPort, nil))
}
