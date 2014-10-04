package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	log "github.com/cihub/seelog"
	"github.com/miekg/dns"

	//"github.com/PuerkitoBio/goquery"
)

type config struct {
	DNS               []string `toml:"dns"`
	Domain            []string
	MainChannelNumber int
}

var (
	c           = dns.Client{}
	timeout     = 5000
	mainChannel = make(chan int, 5) // 主线程
	wg          = sync.WaitGroup{}  // 用于等待所有 goroutine 结束
	re, _       = regexp.Compile("(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]).(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0).(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0).(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[0-9])")
)

func init() {
	c.Net = "tcp"
	c.ReadTimeout = time.Duration(timeout) * time.Millisecond
	c.WriteTimeout = time.Duration(timeout) * time.Millisecond
}
func main() {
	//读取配置文件
	var conf config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		fmt.Println(err)
		return
	}
	//设置线程数
	mainChannel = make(chan int, conf.MainChannelNumber)
	//循环
	for _, domain := range conf.Domain {
		for _, dns := range conf.DNS {
			if !strings.Contains(dns, ":") {
				dns = net.JoinHostPort(dns, "53")
			}
			//多进程
			mainChannel <- 1
			wg.Add(1)
			go Query(domain, dns)
		}
		log.Info("-------------", domain, "--done!------------")
	}

	//等待完成
	wg.Wait()
	log.Info("finished!")
}
func Query(domain, d string) {
	defer func() {
		wg.Done()
		<-mainChannel
	}()
	m := dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.RecursionDesired = true

	r, _, err := c.Exchange(&m, d)
	if r == nil {
		log.Error("*** error: %s\n", err.Error())
		return
	}

	if r.Rcode != dns.RcodeSuccess {
		log.Error(" *** invalid answer name %s after MX query for %s\n", domain, d)
		return
	}

	// Stuff must be in the answer section
	for _, a := range r.Answer {
		one := re.Find([]byte(a.String()))
		if len(string(one)) > 2 {
			log.Info(d, " Find: ", string(one))
		}
	}
}
