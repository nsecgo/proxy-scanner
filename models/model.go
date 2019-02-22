package models

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type HttpMode uint8

const (
	None HttpMode = iota
	TransparentHttp
	AnonymousHttp
	DistortingHttp
	EliteHttp
)

type HttpProxy struct {
	Ip        string
	Port      uint16
	Mode      HttpMode
	Connect   bool
	LastAlive time.Time
	KeepAlive bool
	LastCheck time.Time
}
type SocksProxy struct {
	Ip        string
	Port      uint16
	Socks5    bool
	LastAlive time.Time
	KeepAlive bool
	LastCheck time.Time
}

var db *sql.DB
var insertOrUpdateHttpStmt *sql.Stmt

var insertOrUpdateSocksStmt *sql.Stmt

var disableHttpStmt *sql.Stmt
var disableSocksStmt *sql.Stmt

func Init(dbType, dsn string) {
	if dbType == "mysql" {
		//must set parseTime=true
		if strings.Contains(dsn, "?") {
			if !strings.Contains(dsn, "parseTime=true") {
				dsn += "&parseTime=true"
			}
		} else {
			dsn += "?parseTime=true"
		}
		var err error
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Fatal(err)
		}
		db.SetMaxIdleConns(10)

		insertOrUpdateHttpStmt, err = db.Prepare(`INSERT INTO http_proxies(ip,port,mode,connect,last_alive,keep_alive) VALUES (?,?,?,?,?,?)
on duplicate key update port=?,mode=?,connect=?,last_alive=?,keep_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
		insertOrUpdateSocksStmt, err = db.Prepare(`INSERT INTO socks_proxies(ip,port,socks5,last_alive,keep_alive) VALUES (?,?,?,?,?)
on duplicate key update port=?,socks5=?,last_alive=?,keep_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
		disableHttpStmt, err = db.Prepare("update http_proxies set keep_alive=FALSE where ip=?")
		if err != nil {
			log.Fatal(err)
		}
		disableSocksStmt, err = db.Prepare("update socks_proxies set keep_alive=FALSE where ip=?")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var err error
		db, err = sql.Open("sqlite3", dsn)
		if err != nil {
			log.Fatal(err)
		}

		insertOrUpdateHttpStmt, err = db.Prepare(`INSERT INTO http_proxies(ip,port,mode,connect,last_alive,keep_alive) VALUES (?,?,?,?,?,?)
ON CONFLICT(ip) DO UPDATE SET port=?,mode=?,connect=?,last_alive=?,keep_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
		insertOrUpdateSocksStmt, err = db.Prepare(`INSERT INTO socks_proxies(ip,port,socks5,last_alive,keep_alive) VALUES (?,?,?,?,?)
ON CONFLICT(ip) DO UPDATE SET port=?,socks5=?,last_alive=?,keep_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
		disableHttpStmt, err = db.Prepare("update http_proxies set keep_alive=FALSE where ip=?")
		if err != nil {
			log.Fatal(err)
		}
		disableSocksStmt, err = db.Prepare("update socks_proxies set keep_alive=FALSE where ip=?")
		if err != nil {
			log.Fatal(err)
		}
	}

}
func (p *HttpProxy) InsertOrUpdate(reCheck bool) {
	now := time.Now()
	if p.Mode != None || p.Connect {
		p.KeepAlive = true
		p.LastAlive = now
		_, err := insertOrUpdateHttpStmt.Exec(p.Ip, p.Port, p.Mode, p.Connect, p.LastAlive, p.KeepAlive, p.Port, p.Mode, p.Connect, p.LastAlive, p.KeepAlive)
		if err != nil {
			log.Println(err)
			return
		}
	} else if reCheck {
		disableHttpStmt.Exec(p.Ip)
	}
}
func (p *SocksProxy) InsertOrUpdate(reCheck bool) {
	now := time.Now()

	if p.Socks5 {
		p.KeepAlive = true
		p.LastAlive = now
		_, err := insertOrUpdateSocksStmt.Exec(p.Ip, p.Port, p.Socks5, p.LastAlive, p.KeepAlive, p.Port, p.Socks5, p.LastAlive, p.KeepAlive)
		if err != nil {
			log.Println(err)
			return
		}
	} else if reCheck {
		disableSocksStmt.Exec(p.Ip)
	}
}
func Write2File4Scan(ips *sync.Map) string {
	f, err := ioutil.TempFile("", "spider_")
	if err != nil {
		log.Fatal(err)
	}
	ips.Range(func(key, value interface{}) bool {
		_, err = f.WriteString(key.(string) + "\n")
		if err != nil {
			log.Println(err)
		}
		return true
	})
	f.Sync()
	f.Close()
	return f.Name()
}
func SendToCheck(waitCheckch chan string, WaitCheckCount *uint32) {
	t := time.Now().Add(-12 * time.Hour)
	var ip string
	var port uint16
	// SQL_BUFFER_RESULT forces the result to be put into a temporary table.
	// This helps MySQL free the table locks early and helps in cases where it takes a long time to send the result set to the client.
	// This modifier can be used only for top-level SELECT statements, not for subqueries or following UNION.
	rows, err := db.Query("select ip,port from http_proxies where keep_alive=true and last_check<?", t)
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		rows.Scan(&ip, &port)
		waitCheckch <- "http://" + ip + ":" + strconv.Itoa(int(port))
		atomic.AddUint32(WaitCheckCount, 1)
	}
	rows, err = db.Query("select ip,port from socks_proxies where keep_alive=true and last_check<?", t)
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		rows.Scan(&ip, &port)
		waitCheckch <- "socks://" + ip + ":" + strconv.Itoa(int(port))
		atomic.AddUint32(WaitCheckCount, 1)
	}
}
