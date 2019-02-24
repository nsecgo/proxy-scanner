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

type Status uint8

const (
	Close Status = iota
	Transparent
	Anonymous
)

type HttpProxy struct {
	Ip        string
	Port      uint16
	Status    Status
	Connect   bool
	LastAlive time.Time
	LastCheck time.Time
}
type SocksProxy struct {
	Ip        string
	Port      uint16
	Socks5    bool
	LastAlive time.Time
	LastCheck time.Time
}

var db *sql.DB
var insertOrUpdateHttpStmt *sql.Stmt
var insertOrUpdateSocksStmt *sql.Stmt

var disableHttpStmt *sql.Stmt
var disableSocksStmt *sql.Stmt

func Init(dbType, dsn string) {
	var err error
	if dbType == "mysql" {
		//must set parseTime=true
		if strings.Contains(dsn, "?") {
			if !strings.Contains(dsn, "parseTime=true") {
				dsn += "&parseTime=true"
			}
		} else {
			dsn += "?parseTime=true"
		}

		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Fatal(err)
		}
		db.SetMaxIdleConns(10)

		insertOrUpdateHttpStmt, err = db.Prepare(`INSERT INTO http_proxies(ip,port,status,connect,last_alive) VALUES (?,?,?,?,?)
on duplicate key update port=?,anonymous=?,connect=?,last_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
		insertOrUpdateSocksStmt, err = db.Prepare(`INSERT INTO socks5_proxies(ip,port,socks5,last_alive) VALUES (?,?,?,?)
on duplicate key update port=?,socks5=?,last_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		db, err = sql.Open("sqlite3", dsn)
		if err != nil {
			log.Fatal(err)
		}

		insertOrUpdateHttpStmt, err = db.Prepare(`INSERT INTO http_proxies(ip,port,status,connect,last_alive) VALUES (?,?,?,?,?)
ON CONFLICT(ip) DO UPDATE SET port=?,status=?,connect=?,last_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
		insertOrUpdateSocksStmt, err = db.Prepare(`INSERT INTO socks5_proxies(ip,port,socks5,last_alive) VALUES (?,?,?,?)
ON CONFLICT(ip) DO UPDATE SET port=?,socks5=?,last_alive=?`)
		if err != nil {
			log.Fatal(err)
		}
	}
	disableHttpStmt, err = db.Prepare("update http_proxies set status=" + strconv.Itoa(int(Close)) + ",connect=FALSE where ip=?")
	if err != nil {
		log.Fatal(err)
	}
	disableSocksStmt, err = db.Prepare("update socks5_proxies set socks5=FALSE where ip=?")
	if err != nil {
		log.Fatal(err)
	}
}
func (p *HttpProxy) InsertOrUpdate(reCheck bool) {
	if p.Status != Close || p.Connect {
		p.LastAlive = time.Now()
		_, err := insertOrUpdateHttpStmt.Exec(p.Ip, p.Port, p.Status, p.Connect, p.LastAlive, p.Port, p.Status, p.Connect, p.LastAlive)
		if err != nil {
			log.Println(err)
			return
		}
	} else if reCheck {
		disableHttpStmt.Exec(p.Ip)
	}
}
func (p *SocksProxy) InsertOrUpdate(reCheck bool) {
	if p.Socks5 {
		p.LastAlive = time.Now()
		_, err := insertOrUpdateSocksStmt.Exec(p.Ip, p.Port, p.Socks5, p.LastAlive, p.Port, p.Socks5, p.LastAlive)
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

//And Delete offline for more than 24 hours
func SendToReCheck(waitCheckch chan string, WaitCheckCount *uint32) {
	lc := time.Now().Add(-10 * time.Hour)
	la := time.Now().Add(-24 * time.Hour)
	var ip string
	var port uint16
	// SQL_BUFFER_RESULT forces the result to be put into a temporary table.
	// This helps MySQL free the table locks early and helps in cases where it takes a long time to send the result set to the client.
	// This modifier can be used only for top-level SELECT statements, not for subqueries or following UNION.
	db.Query("delete from http_proxies where last_alive<?", la)
	db.Query("delete from socks5_proxies where last_alive<?", la)
	rows, err := db.Query("select ip,port from http_proxies where status>0 and last_check<?", lc)
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		rows.Scan(&ip, &port)
		waitCheckch <- "http://" + ip + ":" + strconv.Itoa(int(port))
		atomic.AddUint32(WaitCheckCount, 1)
	}
	rows, err = db.Query("select ip,port from socks5_proxies where socks5=true and last_check<?", lc)
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		rows.Scan(&ip, &port)
		waitCheckch <- "socks5://" + ip + ":" + strconv.Itoa(int(port))
		atomic.AddUint32(WaitCheckCount, 1)
	}
}
