package aria2

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"net/http"
)

type IRequestHandler interface {
	SendRequest(req RpcRequest) (interface{}, error)
	SetUrl(url string)
}

type HttpRequestHandler struct {
	Url    string
	Client *fasthttp.Client
}

func NewHttpRequestHandler() *HttpRequestHandler {
	return &HttpRequestHandler{Client: &fasthttp.Client{}}
}

func (self *HttpRequestHandler) SetUrl(url string) {
	self.Url = url
}

func (self *HttpRequestHandler) SendRequest(req RpcRequest) (interface{}, error) {
	httpreq := fasthttp.AcquireRequest()
	httpreq.SetRequestURI(self.Url)
	requestBody, err := json.Marshal(req)
	if err != nil {
		return RpcResponse{}, err
	}
	httpreq.SetBody(requestBody)
	httpreq.Header.SetContentType("application/json")
	httpreq.Header.SetMethod("POST")
	httpresp := fasthttp.AcquireResponse()
	if err := self.Client.Do(httpreq, httpresp); err != nil {
		return RpcResponse{}, err
	}
	b := httpresp.Body()
	res := &RpcResponse{}
	if err := json.Unmarshal(b, res); err != nil {
		return RpcResponse{}, err
	}
	if (*res).Error != nil {
		return RpcResponse{}, errors.New((*res).Error.(string))
	}
	return (*res).Result, nil
}

type WebsocketRequestHandler struct {
	Url         string
	Client      *Aria2Client
	Conn        *websocket.Conn
	functions   map[string][]Callback
	resultStore map[string]chan RpcResponse
}

func NewWebsocketRequestHandler() *WebsocketRequestHandler {
	functions := make(map[string][]Callback, 5)
	resultStore := make(map[string]chan RpcResponse, 5)
	return &WebsocketRequestHandler{functions: functions, resultStore: resultStore}
}

func (self *WebsocketRequestHandler) SetUrl(url string) {
	self.Url = url
}
func (self *WebsocketRequestHandler) SetClient(c *Aria2Client) {
	self.Client = c
}
func (self *WebsocketRequestHandler) listen() error {
	con, _, err := websocket.DefaultDialer.Dial(self.Url, http.Header{})
	if err != nil {
		return err
	}
	self.Conn = con
	defer func() { self.Conn.Close() }()
	for {
		buffer := NewBuffer()
		t, reader, err := self.Conn.NextReader()
		if err != nil {
			continue
		}
		_, err = buffer.ReadFrom(reader)
		if err != nil {
			continue
		}
		if t == websocket.TextMessage {
			go func(buffer *bytes.Buffer) {
				defer PutBuffer(buffer)
				self.handleEvent(buffer.Bytes())
			}(buffer)
		} else {
			PutBuffer(buffer)
		}
	}
	return nil
}
func (self *WebsocketRequestHandler) handleEvent(b []byte) {
	str := string(b)
	if result := gjson.Get(str, "result"); result.Exists() { // 是rpc返回
		res := &RpcResponse{}
		if err := json.Unmarshal(b, res); err != nil {
			return
		}
		if v, ok := self.resultStore[gjson.Get(str, "id").String()]; ok {
			v <- *res
		}
	} else { // 是notice
		if method := gjson.Get(str, "method"); method.Exists() {
			req := &RpcRequest{}
			if err := json.Unmarshal(b, req); err != nil {
				return
			}
			for _, function := range self.functions[method.String()] {
				go function(self.Client, *req)
			}
		}
	}
}

func (self *WebsocketRequestHandler) SendRequest(req RpcRequest) (interface{}, error) {
	self.resultStore[req.Id.(string)] = make(chan RpcResponse)
	defer delete(self.resultStore, req.Id.(string))

	if err := self.Conn.WriteJSON((&req).ToMap()); err != nil { //todo panic
		return RpcResponse{}, err
	}
	rpcres := <-self.resultStore[req.Id.(string)]
	return rpcres.Result, nil
}

func (self *WebsocketRequestHandler) register(function Callback, type_ string) {
	if v, ok := self.functions[type_]; ok {
		self.functions[type_] = append(v, function)
	} else {
		functions := append(make([]Callback, 0, 2), function)
		self.functions[type_] = functions
	}
}

func (self *WebsocketRequestHandler) OnDownloadStart(callback Callback) Callback {
	self.register(callback, "aria2.onDownloadStart")
	return callback
}
func (self *WebsocketRequestHandler) OnDownloadPause(callback Callback) Callback {
	self.register(callback, "aria2.onDownloadPause")
	return callback
}
func (self *WebsocketRequestHandler) OnDownloadStop(callback Callback) Callback {
	self.register(callback, "aria2.onDownloadStop")
	return callback
}
func (self *WebsocketRequestHandler) OnDownloadComplete(callback Callback) Callback {
	self.register(callback, "aria2.onDownloadComplete")
	return callback
}
func (self *WebsocketRequestHandler) OnDownloadError(callback Callback) Callback {
	self.register(callback, "aria2.onDownloadError")
	return callback
}
func (self *WebsocketRequestHandler) OnBtDownloadComplete(callback Callback) Callback {
	self.register(callback, "aria2.onBtDownloadComplete")
	return callback
}
