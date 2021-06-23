package aria2

import (
	"fmt"
)

type IdFactory func() string

type Aria2Client struct {
	Url     string
	Id      *IdFactory
	Mode    *string
	Token   *string
	Queue   *chan RpcRequest
	Handler IRequestHandler
}

func NewAria2Client(url string,
	id *IdFactory,
	mode *string,
	token *string,
	queue *chan RpcRequest,
	handler IRequestHandler) *Aria2Client {
	if id == nil {
		func_ := func() IdFactory {
			inner := 0
			return func() string {
				inner++
				return string(inner)
			}
		}()
		id = &func_
	}
	if mode == nil {
		mode_ := "normal"
		mode = &mode_
	}
	if queue == nil {
		ch := make(chan RpcRequest, 10)
		queue = &ch
	}
	handler.SetUrl(url)
	client := &Aria2Client{Url: url, Id: id, Mode: mode, Token: token, Queue: queue, Handler: handler}
	if v, ok := handler.(*WebsocketRequestHandler); ok {
		v.SetClient(client)
		go v.listen()
	}
	return client
}

func (self *Aria2Client) jsonrpc(method string, params []interface{}, prefix string) (interface{}, error) {
	if self.Token != nil {
		tokenStr := fmt.Sprintf("token:%s", *self.Token)
		if method == "multicall" {
			for _, param := range params[0].([]interface{}) {
				oldParam := param.(map[string]interface{})["params"].([]interface{})
				a := make([]interface{}, 0, 2)
				a = append(a, tokenStr)
				param.(map[string]interface{})["params"] = append(a, oldParam...)
			}
		} else {
			newParams := make([]interface{}, 0, 2)
			newParams = append(newParams, tokenStr)
			params = append(newParams, params...)
		}
	}
	reqObj := RpcRequest{Jsonrpc: "2.0", Id: (*self.Id)(), Method: prefix + method, Params: params}
	switch *self.Mode {
	case "batch":
		*self.Queue <- reqObj
		return nil, nil
	case "format":
		return reqObj, nil
	case "normal":
		return self.Handler.SendRequest(reqObj)
	default:
		return nil, nil
	}
}

// AddUri position一般是nil
//添加新的任务到下载队列
//:param uris: 要添加的链接 务必是list HTTP/FTP/SFTP/BitTorrent URIs (strings)
//:param options:附加参数
//:param position:在下载队列中的位置
//:return:包含结果的json
//{"result":"2089b05ecca3d829"}
func (self *Aria2Client) AddUri(uris []string, options *map[string]interface{}, position *int) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), uris)
	params = addOptionsAndPosition(params, options, position)
	return self.jsonrpc("addUri", params, "aria2.")
}

// AddTorrent 下载种子
//:param torrent: base64编码的种子文件 base64.b64encode(open("xxx.torrent","rb").read())
//:param uris: uri用于播种 对于单个文件，URI可以是指向资源的完整URI;如果URI以/结尾，则添加到torrent文件。
//对于多文件的torrent，则会在torrent文件中添加名称和路径以生成每个文件的URI。
//:param options:参数字典
//:param position:在下载队列中的位置
//:return:包含结果的json
//{"result":"2089b05ecca3d829"}
func (self *Aria2Client) AddTorrent(torrent string, uris *[]string, options *map[string]interface{}, position *int) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), torrent)
	if uris != nil {
		params = append(params, *uris)
	}
	params = addOptionsAndPosition(params, options, position)
	return self.jsonrpc("addTorrent", params, "aria2.")
}

// AddMetalink 此方法通过上载一个来添加一个Metalink下载 metalink是一个用base64编码的字符串，其中包含“.metalink”文件。
//:param metalink: base64编码的字符串 base64.b64encode(open('file.meta4',"rb").read())
//:param options:参数字典
//:param position:在下载队列中的位置
//:return:包含结果的json
//{"result":"2089b05ecca3d829"}
func (self *Aria2Client) AddMetalink(metalink []string, options *map[string]interface{}, position *int) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), metalink)
	params = addOptionsAndPosition(params, options, position)
	return self.jsonrpc("addMetalink", params, "aria2.")
}

// Remove 正在下载的停止下载 停止的删除状态
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:包含结果的json
//{"result":"2089b05ecca3d829"}
func (self *Aria2Client) Remove(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("remove", params, "aria2.")
}

// ForceRemove 此方法删除由gid表示的下载。这个方法的行为就像aria2.remove(),但是会立即生效，而不执行任何需要时间的操作，
//例如联系BitTorrent跟踪器先取消下载。
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:包含结果的json
func (self *Aria2Client) ForceRemove(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("forceRemove", params, "aria2.")
}

// Pause 此方法暂停由gid(字符串)表示的下载。暂停下载的状态变为暂停。如果下载是活动的，下载将放在等待队列的前面。
//当状态暂停时，下载不会启动。要将状态更改为等待，请使用aria2.unpause()方法
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:包含结果的json
func (self *Aria2Client) Pause(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("pause", params, "aria2.")
}

// PauseAll 这个方法相当于为每个活动/等待的下载调用aria2.pause()。这个方法返回OK。
//:return:包含结果的json
func (self *Aria2Client) PauseAll() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("pauseAll", params, "aria2.")
}

// ForcePause 此方法暂停由gid表示的下载。这个方法的行为就像aria2.pause()，只是这个方法暂停下载，不执行任何需要时间的操作，
//比如联系BitTorrent tracker先取消下载。
//:param gid:GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:包含结果的json
func (self *Aria2Client) ForcePause(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("forcePause", params, "aria2.")
}

// ForcePauseAll 这个方法相当于对每个活动/等待的下载调用aria2.forcePause()。这个方法返回OK
//:return:包含结果的json
func (self *Aria2Client) ForcePauseAll() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("forcePauseAll", params, "aria2.")
}

// Unpause 此方法将由gid (string)表示的下载状态从暂停更改为等待，从而使下载符合重新启动的条件。此方法返回未暂停下载的GID。
//:param gid:GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:包含结果的json
func (self *Aria2Client) Unpause(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("unpause", params, "aria2.")
}

// UnpauseAll 这个方法相当于对每个暂停的下载调用aria2.unpause()。这个方法返回OK
//:return:包含结果的json
func (self *Aria2Client) UnpauseAll() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("unpauseAll", params, "aria2.")
}

// TellStatus /*
func (self *Aria2Client) TellStatus(gid string, keys *[]string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	if keys != nil {
		params = append(params, *keys)
	}
	return self.jsonrpc("tellStatus", params, "aria2.")
}

// GetUris 此方法返回由gid(字符串)表示的下载中使用的uri。响应是一个json，它包含以下键。值是字符串
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:json格式的结果
//[{'status': 'used',  如果url已经使用就是used ，还在队列中就是waiting
//'uri': 'http://exa
func (self *Aria2Client) GetUris(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("getUris", params, "aria2.")
}

// GetFiles 返回下载文件列表
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:
//[{'index': '1',  件的索引，从1开始，与文件在多文件中出现的顺序相同
//'length': '34896138',  文件大小 byte
//'completedLength': '34896138',  此文件的完整长度(以字节为单位)。请注意，
//completedLength的和可能小于aria2.tellStatus()方法返回的completedLength。
//这是因为在aria2.getFiles()中completedLength只包含完成的片段。
//另一方面，在aria2.tellStatus()中完成的长度也包括部分完成的片段。
//'path': '/downloads/file',   路径
//'selected': 'true',   如果此文件是由——select-file选项选择的，则为true。
//如果——select-file没有指定，或者这是单文件的torrent文件，或者根本不是torrent下载，那么这个值总是为真。否则错误。
//'uris': [{'status': 'used',  返回此文件的uri列表。元素类型与aria2.getUris()方法中使用的结构相同。
//'uri': 'http://example.org/file'}]}]
func (self *Aria2Client) GetFiles(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("getFiles", params, "aria2.")
}

// GetPeers 返回下载对象，仅适用于bt
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:
//[{'amChoking': 'true',
//'bitfield': 'ffffffffffffffffffffffffffffffffffffffff',
//'downloadSpeed': '10602',
//'ip': '10.0.0.9',
//'peerChoking': 'false',
//'peerId': 'aria2%2F1%2E10%2E5%2D%87%2A%EDz%2F%F7%E6',
//'port': '6881',
//'seeder': 'true',
//'uploadSpeed': '0'},
//{'amChoking': 'false',
//'bitfield': 'ffffeff0fffffffbfffffff9fffffcfff7f4ffff',
//'downloadSpeed': '8654',
//'ip': '10.0.0.30',
//'peerChoking': 'false',
//'peerId': 'bittorrent client758',
//'port': '37842',
//'seeder': 'false',
//'uploadSpeed': '6890'}]
func (self *Aria2Client) GetPeers(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("getPeers", params, "aria2.")
}

// GetServers 此方法返回当前连接的HTTP(S)/FTP/SFTP服务器的下载，用gid(字符串)表示。响应是一个结构数组，包含以下key。值是字符串。
//:param gid:GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:
//[{'index': '1',
//'servers': [{'currentUri': 'http://example.org/file',  # 正在使用的
//'downloadSpeed': '10467',    # 下载速度(byte/sec)
//'uri': 'http://example.org/file'}]}]}  #原url
func (self *Aria2Client) GetServers(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("getServers", params, "aria2.")
}

// TellActive 此方法返回活动下载列表。响应是一个与aria2.tellStatus()方法返回的结构相同的数组。关于keys参数，请参考aria2.tellStatus()方法。
//:param keys: 如果指定，则返回结果只包含keys数组中的键。如果键keys空或省略，则返回结果包含所有键。
//:return:
//json格式的结果
//{'bitfield': '0000000000',
//'completedLength': '901120',
//'connections': '1',
//'dir': '/downloads',
//'downloadSpeed': '15158',
//'files': [{'index': '1',
//'length': '34896138',
//'completedLength': '34896138',
//'path': '/downloads/file',
//'selected': 'true',
//'uris': [{'status': 'used',
//'uri': 'http://example.org/file'}]}],
//'gid': '2089b05ecca3d829',
//'numPieces': '34',
//'pieceLength': '1048576',
//'status': 'active',
//'totalLength': '34896138',
//'uploadLength': '0',
//'uploadSpeed': '0'}
func (self *Aria2Client) TellActive(keys *[]string) (interface{}, error) {
	params := make([]interface{}, 0, 2)
	if keys != nil {
		params = append(params, *keys)
	}
	return self.jsonrpc("tellActive", params, "aria2.")
}

// TellWaiting 此方法返回等待下载的列表，包括暂停的下载。偏移量是一个整数，它指定等待在前面的下载的偏移量。
//num是一个整数，指定最大值。要返回的下载数量。关于keys参数，请参考aria2.tellStatus()方法。
//:param offset: 起始索引
//:param num: 数量
//:param keys: 同上
//:return: 同上
func (self *Aria2Client) TellWaiting(offset int, num int, keys *[]string) (interface{}, error) {
	params := make([]interface{}, 0, 3)
	params = append(params, offset, num)
	if keys != nil {
		params = append(params, *keys)
	}
	return self.jsonrpc("tellWaiting", params, "aria2.")
}

// TellStopped 此方法返回停止下载的列表 关于keys参数，请参考aria2.tellStatus()方法。
//:param offset: 起始索引
//:param num: 数量
//:param keys: 同上
//:return: 同上
func (self *Aria2Client) TellStopped(offset int, num int, keys *[]string) (interface{}, error) {
	params := make([]interface{}, 0, 3)
	params = append(params, offset, num)
	if keys != nil {
		params = append(params, *keys)
	}
	return self.jsonrpc("tellStopped", params, "aria2.")
}

// ChangePosition 此方法更改队列中由gid表示的下载位置。pos是一个整数。how是一个字符串。
//如果how是POS_SET，它将下载移动到相对于队列开头的位置。
//如果how是POS_CUR，它将下载移动到相对于当前位置的位置。
//如果how是POS_END，它将下载移动到相对于队列末尾的位置。
//如果目标位置小于0或超过队列的末尾，则将下载分别移动到队列的开头或末尾。响应是一个表示结果位置的整数。
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:param pos: 偏移量
//:param how: 方法
//:return：位置 int
func (self *Aria2Client) ChangePosition(gid string, pos int, how string) (interface{}, error) {
	params := make([]interface{}, 0, 3)
	params = append(params, gid, pos, how)
	return self.jsonrpc("changePosition", params, "aria2.")
}

// ChangeUri 此方法从delUris中删除uri，并将addUris中的uri附加到以gid表示的下载中。
//delUris和addUris是字符串列表。下载可以包含多个文件，每个文件都附加了uri。
//fileIndex用于选择要删除/附加哪个文件。fileIndex从0开始。
//
//当位置被省略时，uri被附加到列表的后面。这个方法首先执行删除，然后执行添加。
//position是删除uri后的位置，而不是调用此方法时的位置。在删除URI时，如果下载中存在相同的URI，
//则对于deluri中的每个URI只删除一个URI。换句话说，如果有三个uri http://example.org/aria2，并且您希望将它们全部删除，
//则必须在delUris中指定(至少)3个http://example.org/aria2。这个方法返回一个包含两个整数的列表。第一个整数是删除uri的数目。
//第二个整数是添加的uri的数量。
//
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:param fileIndex:用于选择要删除/附加哪个文件。fileIndex从0开始。
//:param delUris: 要删除的
//:param addUris: 要添加的
//:param position: position用于指定在现有的等待URI列表中插入URI的位置 0开始
//:return:
//[0, 1]
func (self *Aria2Client) ChangeUri(gid string, fileIndex int, delUris []string, addUris []string, position *int) (interface{}, error) {
	params := make([]interface{}, 0, 5)
	params = append(params, gid, fileIndex, delUris, addUris)
	if position != nil {
		params = append(params, *position)
	}
	return self.jsonrpc("changeUri", params, "aria2.")
}

// GetOption 此方法返回由gid表示的下载选项。
//注意，此方法不会返回没有默认值,也没有在配置文件或RPC方法的命令行上设置这些的选项
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:return:
//{'allow-overwrite': 'false',
//'allow-piece-length-change': 'false',
//'always-resume': 'true',
//'async-dns': 'true',
func (self *Aria2Client) GetOption(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid)
	return self.jsonrpc("getOption", params, "aria2.")
}

// ChangeOption 此方法动态地更改由gid (string)表示的下载选项。options是一个字典。输入文件小节中列出的选项是可用的，但以下选项除外:
//dry-run
//metalink-base-uri
//parameterized-uri
//pause
//piece-length
//rpc-save-upload-metadata
//除了以下选项外，更改活动下载的其他选项将使其重新启动(重新启动本身由aria2管理，不需要用户干预):
//bt-max-peers
//bt-request-peer-speed-limit
//bt-remove-unselected-file
//force-save
//max-download-limit
//max-upload-limit
//此方法返回OK表示成功。
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值。
//:param options:
//:return:
//"OK"
func (self *Aria2Client) ChangeOption(gid string, options map[string]interface{}) (interface{}, error) {
	params := append(make([]interface{}, 0, 2), gid, options)
	return self.jsonrpc("changeOption", params, "aria2.")
}

// GetGlobalOption 此方法返回全局选项。响应是一个结构体。它的键是选项的名称。值是字符串。
//注意，此方法不会返回没有默认值的选项，也不会在配置文件或RPC方法的命令行上设置这些选项。
//因为全局选项用作新添加下载选项的模板，所以响应包含aria2.getOption()方法返回的键。
//:return:
func (self *Aria2Client) GetGlobalOption() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("getGlobalOption", params, "aria2.")
}

// ChangeGlobalOption 此方法动态更改全局选项。options是一个字典。以下是可供选择的方案:
//bt-max-open-files
//download-result
//keep-unfinished-download-result
//log
//log-level
//max-concurrent-downloads
//max-download-result
//max-overall-download-limit
//max-overall-upload-limit
//optimize-concurrent-downloads
//save-cookies
//save-session
//server-stat-of
//:param options: 参数字典
//:return:  "OK"
func (self *Aria2Client) ChangeGlobalOption(options map[string]interface{}) (interface{}, error) {
	params := append(make([]interface{}, 0, 1), options)
	return self.jsonrpc("changeGlobalOption", params, "aria2.")
}

// GetGlobalStat 此方法返回全局统计信息，如总下载和上传速度。响应是一个字典，包含以下键。值是字符串
//:return:
//{'downloadSpeed': '21846',
//'numActive': '2',  #活动下载数
//'numStopped': '0',  #  当前会话中停止的下载数量。以 --max-download-result 选项为上限
//'numWaiting': '0',  # 等待下载数
//'uploadSpeed': '0'}
func (self *Aria2Client) GetGlobalStat() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("getGlobalStat", params, "aria2.")
}

// PurgeDownloadResult 此方法将已完成/错误/删除的下载清除到空闲内存。这个方法返回OK。
//:return: "OK"
func (self *Aria2Client) PurgeDownloadResult() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("purgeDownloadResult", params, "aria2.")
}

// RemoveDownloadResult 此方法从内存中删除由gid表示的已完成/错误/已删除的下载。此方法返回OK表示成功。
//:param gid: GID(或GID)是管理每个下载的密钥。每个下载将被分配一个唯一的GID。GID在aria2中存储为64位二进制值
//:return: "OK"
func (self *Aria2Client) RemoveDownloadResult(gid string) (interface{}, error) {
	params := append(make([]interface{}, 0, 1), gid)
	return self.jsonrpc("removeDownloadResult", params, "aria2.")
}

// GetVersion 此方法返回aria2的版本和启用的特性列表
//:return:一个字典，包含以下键
//version: aria2的版本
//enabledFeatures: 启用功能的列表。每个特性都以字符串的形式给出
func (self *Aria2Client) GetVersion() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("getVersion", params, "aria2.")
}

// GetSessionInfo 返回会话信息
//:return:字典，包含以下键
//sessionId: 每次调用aria2时生成的会话id
func (self *Aria2Client) GetSessionInfo() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("getSessionInfo", params, "aria2.")
}

// Shutdown 关闭aria2
//:return: "OK"
func (self *Aria2Client) Shutdown() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("shutdown", params, "aria2.")
}

// ForceShutdown 此方法将当前会话保存到由——save-session选项指定的文件中。
//:return:"OK"
func (self *Aria2Client) ForceShutdown() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("forceShutdown", params, "aria2.")
}

// SaveSession 此方法将当前会话保存到由——save-session选项指定的文件中。
//:return:"OK"
func (self *Aria2Client) SaveSession() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("saveSession", params, "aria2.")
}

// Multicall 此方法将多个方法调用封装在单个请求中
//:param methods: 字典数组。结构包含两个键:methodName和params。methodName是要调用的方法名，params是包含方法调用参数的数组。
//此方法返回一个响应数组。元素要么是一个包含方法调用返回值的单条目数组，要么是一个封装的方法调用失败时的fault元素结构。
//example: [{'methodName':'aria2.addUri',
//'params':[['http://example.org']]},
//{'methodName':'aria2.addTorrent',
//'params':[base64.b64encode(open('file.torrent').read())]}]
//:return:
func (self *Aria2Client) Multicall(methods []map[string]interface{}) (interface{}, error) {
	params := append(make([]interface{}, 0, 1), methods)
	return self.jsonrpc("multicall", params, "system.")
}

// ListMethods 此方法在字符串数组中返回所有可用的RPC方法。与其他方法不同，此方法不需要秘密令牌。这是安全的，因为这个方法只返回可用的方法名。
//:return:
func (self *Aria2Client) ListMethods() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("listMethods", params, "system.")
}

// ListNotifications 此方法以字符串数组的形式返回所有可用的RPC通知。与其他方法不同，此方法不需要秘密令牌。
//这是安全的，因为这个方法只返回可用的通知名称。
//:return:
func (self *Aria2Client) ListNotifications() (interface{}, error) {
	params := make([]interface{}, 0, 0)
	return self.jsonrpc("listNotifications", params, "system.")
}
