package aria2

type RpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}
func(req *RpcRequest) ToMap() map[string]interface{} {
	return map[string]interface{}{"jsonrpc":req.Jsonrpc,"id":req.Id,"method":req.Method,"params":req.Params}
}

type RpcResponse struct {
	ID      string      `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

type Callback func(client *Aria2Client, data RpcRequest)

type ErrorMsg struct {
	Code    int
	Message string
	Data    string
}
