package tools

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/a121400/sunnymcptool/mcp"
)

// CleanHexString 清理十六进制字符串
func CleanHexString(hexStr string) string {
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "\t", "")
	hexStr = strings.ReplaceAll(hexStr, "\n", "")
	hexStr = strings.ReplaceAll(hexStr, "\r", "")
	hexStr = strings.ReplaceAll(hexStr, ",", "")
	hexStr = strings.ReplaceAll(hexStr, "0x", "")
	hexStr = strings.ReplaceAll(hexStr, "0X", "")
	return hexStr
}

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "decrypt_packet",
		Description: "解密单个数据包，返回解密后的数据包详情（包括头部信息、原始数据、解密数据、Protobuf解析）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{"type": "string", "description": "原始数据包的十六进制字符串"},
			},
			"required": []string{"data"},
		},
		Handler: decryptPacketHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "parse_protobuf",
		Description: "解析Protobuf数据，返回字段树结构",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{"type": "string", "description": "Protobuf数据的十六进制字符串"},
			},
			"required": []string{"data"},
		},
		Handler: parseProtobufHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "crypto_config_get",
		Description: "获取当前使用的加密配置详情",
		InputSchema: noParamsSchema(),
		Handler:     cryptoConfigGetHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "crypto_config_set",
		Description: "设置加密配置（AES密钥、IV、头部大小等）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":        map[string]interface{}{"type": "string", "description": "配置名称"},
				"aes_key":     map[string]interface{}{"type": "string", "description": "AES密钥（16/24/32字节的字符串或hex）"},
				"aes_iv":      map[string]interface{}{"type": "string", "description": "AES IV（16字节的字符串或hex）"},
				"header_size": map[string]interface{}{"type": "integer", "description": "数据包头部大小（字节数）", "default": 20},
			},
			"required": []string{"name", "aes_key", "aes_iv"},
		},
		Handler: cryptoConfigSetHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "crypto_config_list",
		Description: "列出所有已配置的加密配置",
		InputSchema: noParamsSchema(),
		Handler:     cryptoConfigListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "decrypt_tcp_flow",
		Description: "解密指定TCP连接的完整数据流，返回所有解密后的数据包列表",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "TCP连接的唯一ID (Theology)"},
			},
			"required": []string{"theology"},
		},
		Handler: decryptTcpFlowHandler,
	})
}

func decryptPacketHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	dataHex := v.RequireString("data")
	if err := v.Error(); err != nil {
		return nil, err
	}

	ca := ctx().CryptoAnalyzer
	if ca == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	cleanedHex := CleanHexString(dataHex)
	if len(cleanedHex)%2 != 0 {
		cleanedHex = "0" + cleanedHex
	}
	data, err := hex.DecodeString(cleanedHex)
	if err != nil {
		return nil, fmt.Errorf("无效的十六进制数据: %v", err)
	}

	header, rawHex, payloadHex, decryptedHex, protobufTree, err := ca.ParsePacket(data)
	if err != nil {
		return nil, fmt.Errorf("解析数据包失败: %v", err)
	}

	return map[string]interface{}{
		"success":      true,
		"header":       header,
		"rawHex":       rawHex,
		"payloadHex":   payloadHex,
		"decryptedHex": decryptedHex,
		"protobufTree": protobufTree,
	}, nil
}

func parseProtobufHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	dataHex := v.RequireString("data")
	if err := v.Error(); err != nil {
		return nil, err
	}

	ca := ctx().CryptoAnalyzer
	if ca == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	cleanedHex := CleanHexString(dataHex)
	if len(cleanedHex)%2 != 0 {
		cleanedHex = "0" + cleanedHex
	}
	data, err := hex.DecodeString(cleanedHex)
	if err != nil {
		return nil, fmt.Errorf("无效的十六进制数据: %v", err)
	}

	tree := ca.ParseProtobuf(data, 0)
	return map[string]interface{}{
		"success":      true,
		"protobufTree": tree,
		"dataLength":   len(data),
	}, nil
}

func cryptoConfigGetHandler(args map[string]interface{}) (interface{}, error) {
	ca := ctx().CryptoAnalyzer
	if ca == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	name, aesKey, aesIV, headerSize, msgNames, ok := ca.GetCurrentConfig()
	if !ok {
		return nil, errors.New("没有配置当前加密配置")
	}

	return map[string]interface{}{
		"success":    true,
		"name":       name,
		"aesKey":     aesKey,
		"aesIV":      aesIV,
		"headerSize": headerSize,
		"msgNames":   msgNames,
	}, nil
}

func cryptoConfigSetHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	name := v.RequireString("name")
	aesKey := v.RequireString("aes_key")
	aesIV := v.RequireString("aes_iv")
	headerSize := v.OptionalInt("header_size", 20)
	if err := v.Error(); err != nil {
		return nil, err
	}

	ca := ctx().CryptoAnalyzer
	if ca == nil {
		return nil, errors.New("加密分析器未初始化")
	}
	if name == "" {
		return nil, errors.New("配置名称不能为空")
	}
	if aesKey == "" {
		return nil, errors.New("AES密钥不能为空")
	}
	if aesIV == "" {
		return nil, errors.New("AES IV不能为空")
	}
	if headerSize < 0 {
		return nil, errors.New("头部大小不能为负数")
	}

	ca.AddConfig(name, aesKey, aesIV, headerSize)
	ca.SetCurrentConfig(name)

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("加密配置 '%s' 已设置", name),
		"config": map[string]interface{}{
			"name": name, "aesKey": aesKey, "aesIV": aesIV, "headerSize": headerSize,
		},
	}, nil
}

func cryptoConfigListHandler(args map[string]interface{}) (interface{}, error) {
	ca := ctx().CryptoAnalyzer
	if ca == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	configs := ca.GetAllConfigs()
	configList := make([]map[string]interface{}, 0, len(configs))
	for _, cfg := range configs {
		configList = append(configList, map[string]interface{}{
			"name": cfg.Name, "aesKey": cfg.AESKey, "aesIV": cfg.AESIV,
			"headerSize": cfg.HeaderSize, "isCurrent": cfg.IsCurrent,
		})
	}

	return map[string]interface{}{
		"success":     true,
		"configs":     configList,
		"total":       len(configList),
		"currentName": ca.GetCurrentConfigName(),
	}, nil
}

func decryptTcpFlowHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	if err := v.Error(); err != nil {
		return nil, err
	}

	ca := ctx().CryptoAnalyzer
	if ca == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	h := ctx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("TCP连接 %d 不存在", theology)
	}
	if h.TcpConn == nil && !strings.Contains(strings.ToUpper(h.Way), "TCP") {
		return nil, fmt.Errorf("请求 %d 不是TCP连接", theology)
	}

	socketData := h.SocketData
	if socketData == nil || len(socketData) == 0 {
		return map[string]interface{}{
			"success": true, "theology": theology, "packets": []interface{}{}, "total": 0,
			"message": "没有捕获到数据包",
		}, nil
	}

	packets := make([]map[string]interface{}, 0, len(socketData))
	for i, sd := range socketData {
		if sd == nil || sd.Body == nil || len(sd.Body) == 0 {
			continue
		}
		packet := map[string]interface{}{
			"index": i, "direction": sd.Info.Ico, "time": sd.Info.Time,
			"length": sd.Info.Length, "rawHex": hex.EncodeToString(sd.Body),
		}
		header, _, _, decryptedHex, protobufTree, err := ca.ParsePacket(sd.Body)
		if err != nil {
			packet["decryptError"] = err.Error()
		} else {
			packet["header"] = header
			packet["decryptedHex"] = decryptedHex
			packet["protobufTree"] = protobufTree
		}
		packets = append(packets, packet)
	}

	return map[string]interface{}{
		"success": true, "theology": theology, "url": h.URL, "way": h.Way,
		"packets": packets, "total": len(packets),
	}, nil
}
