package main

import (
	"encoding/json"
	"fmt"
	"github.com/synodriver/goaria2/aria2"
	"time"
)

// todo ws:// http:// 有区别
func main() {
	var (
		token string
		url   string
	)
	fmt.Println("输入url 和 token")
	fmt.Scanf("%s %s", &url, &token)
	client := aria2.NewAria2Client("http://"+url,
		nil, nil, &token, nil, aria2.NewHttpRequestHandler())
	m, e := client.GetVersion()
	if e != nil {
		fmt.Println(e.Error())
	}
	b, e := json.Marshal(m)
	if e != nil {
		fmt.Println(e.Error())
	}
	fmt.Println(string(b))
	wshandler := aria2.NewWebsocketRequestHandler()
	client2 := aria2.NewAria2Client("ws://"+url,
		nil, nil, &token, nil, wshandler)
	finish := make(chan bool)
	client2.OnDownloadStart(func(client *aria2.Aria2Client, data aria2.RpcRequest) {
		b, e := json.Marshal(data)
		if e != nil {
			fmt.Println(e.Error())
		}
		fmt.Printf("开始下载%s\n", string(b))
	})
	client2.OnDownloadComplete(func(client *aria2.Aria2Client, data aria2.RpcRequest) {
		b, e := json.Marshal(data)
		if e != nil {
			fmt.Println(e.Error())
		}
		fmt.Printf("完成下载%s\n", string(b))
		finish <- true
	})
	client2.OnDownloadError(func(client *aria2.Aria2Client, d aria2.RpcRequest) {
		b, e := json.Marshal(d)
		if e != nil {
			fmt.Println(e.Error())
		}
		fmt.Printf("下载出错%s\n", string(b))
		finish <- true
	})
	client2.SetTimeout(time.Second * 5)
	client2.AddUri([]string{"https://google.com"}, nil, nil)
	<-finish
}
