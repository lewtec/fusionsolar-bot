package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

func main() {
	b := rod.New().ControlURL(os.Getenv("BROWSER_CDP")).MustConnect()
	p := b.MustPage("")
	go p.EachEvent(func(e *proto.NetworkLoadingFailed) { fmt.Println("NETFAIL", e.ErrorText, e.BlockedReason) }, func(e *proto.NetworkResponseReceived) {
		if e.Response.Status >= 400 {
			fmt.Println("HTTP", e.Response.Status, e.Response.URL)
		}
	}, func(e *proto.RuntimeExceptionThrown) {
		fmt.Println("EXC", e.ExceptionDetails.Text, e.ExceptionDetails.Exception)
	}, func(e *proto.NetworkRequestWillBeSent) {
		if len(e.Request.URL) > 8 && !contains(e.Request.URL) {
			fmt.Println("REQ", e.Request.URL[:min(120, len(e.Request.URL))])
		}
	})()
	proto.NetworkEnable{}.Call(p)
	proto.RuntimeEnable{}.Call(p)
	p.MustNavigate("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")
	p.MustWaitLoad()
	time.Sleep(5 * time.Second)
	p.MustElement("div#username input").MustInput(os.Getenv("FUSIONSOLAR_USER"))
	pw := p.MustElement("div#password input")
	pw.MustInput(os.Getenv("FUSIONSOLAR_PASSWORD"))
	pw.MustType(input.Enter)
	time.Sleep(40 * time.Second)
	p.MustNavigate("https://intl.fusionsolar.huawei.com")
	time.Sleep(25 * time.Second)
	fmt.Println("AFTERNAV", p.MustEval(`() => document.querySelectorAll("a.nco-home-list-text-ellipsis").length`).String())
	p.MustEval(`() => [...document.querySelectorAll("a.threeMenu_title")].find(a=>a.innerText.trim()=="List View").click()`)
	time.Sleep(25 * time.Second)
	fmt.Println("URL", p.MustInfo().URL)
	d := os.Getenv("OUT_DIR")
	p.MustScreenshot(d + "/dbg.png")
	os.WriteFile(d+"/dbg.html", []byte(p.MustHTML()), 0o644)
	fmt.Println(p.MustEval(`() => [...document.querySelectorAll('iframe')].map(f=>f.src).join('\n')+'\n--\n'+[...new Set(performance.getEntriesByType('resource').map(r=>new URL(r.name).host+' '+r.responseStatus))].join('\n')`).String())
	fmt.Println(p.MustEval(`() => document.querySelectorAll("a.nco-home-list-text-ellipsis").length + " :: " + document.body.innerText.slice(0,1500)`).String())
	fmt.Println(p.MustEval(`() => [...document.querySelectorAll("a")].map(a=>a.className+' | '+a.innerText.slice(0,40)).filter(s=>s.trim()!=' |').slice(0,60).join('\n')`).String())
}

func contains(u string) bool { return len(u) > 35 && u[8:35] == "intl.fusionsolar.huawei.com" }
