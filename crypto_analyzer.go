package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// CryptoConfig 加密配置
type CryptoConfig struct {
	Name       string         `json:"name"`        // 配置名称
	AESKey     string         `json:"aes_key"`     // AES密钥 (原始字符串或hex)
	AESIV      string         `json:"aes_iv"`      // AES IV (原始字符串或hex)
	HeaderSize int            `json:"header_size"` // 头部大小
	MsgNames   map[int]string `json:"msg_names"`   // 消息ID映射
}

// PacketHeader 数据包头部
type PacketHeader struct {
	TotalLen   uint32 `json:"total_len"`
	MsgID      uint32 `json:"msg_id"`
	MsgName    string `json:"msg_name"`
	Seq1       uint32 `json:"seq1"`
	Seq2       uint32 `json:"seq2"`
	Identifier uint32 `json:"identifier"`
}

// DecryptedPacket 解密后的数据包
type DecryptedPacket struct {
	Index        int          `json:"index,omitempty"`     // 数据包索引
	Direction    string       `json:"direction,omitempty"` // 方向: "上行" 或 "下行"
	Header       PacketHeader `json:"header"`
	RawHex       string       `json:"raw_hex"`
	PayloadHex   string       `json:"payload_hex"`
	DecryptedHex string       `json:"decrypted_hex"`
	ProtobufTree string       `json:"protobuf_tree"`
	Error        string       `json:"error,omitempty"`
}

// CryptoAnalyzer 加密分析器
type CryptoAnalyzer struct {
	configs map[string]*CryptoConfig
	current string
	mu      sync.RWMutex
}

// 全局加密分析器实例
var cryptoAnalyzer *CryptoAnalyzer

// InitCryptoAnalyzer 初始化全局加密分析器
func InitCryptoAnalyzer() {
	cryptoAnalyzer = NewCryptoAnalyzer()
	cryptoAnalyzer.LoadDefaultConfig()
}

// NewCryptoAnalyzer 创建新的加密分析器
func NewCryptoAnalyzer() *CryptoAnalyzer {
	return &CryptoAnalyzer{
		configs: make(map[string]*CryptoConfig),
		current: "",
	}
}

// LoadDefaultConfig 加载三国杀默认配置
func (c *CryptoAnalyzer) LoadDefaultConfig() {
	c.mu.Lock()
	defer c.mu.Unlock()

	defaultConfig := &CryptoConfig{
		Name:       "三国杀",
		AESKey:     "Eeo1hSnvNVW9DoLr",
		AESIV:      "FGuuBlp66dtu3M6l",
		HeaderSize: 20,
		MsgNames:   make(map[int]string),
	}

	// 添加一些常见的消息ID映射
	defaultConfig.MsgNames[30000] = "心跳"
	defaultConfig.MsgNames[30001] = "心跳返回"
	defaultConfig.MsgNames[30002] = "登录请求"
	defaultConfig.MsgNames[30003] = "登录返回"

	c.configs[defaultConfig.Name] = defaultConfig
	c.current = defaultConfig.Name
}

// AddConfig 添加配置
func (c *CryptoAnalyzer) AddConfig(config *CryptoConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if config.MsgNames == nil {
		config.MsgNames = make(map[int]string)
	}
	c.configs[config.Name] = config
}

// SetCurrentConfig 设置当前配置
func (c *CryptoAnalyzer) SetCurrentConfig(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.configs[name]; !ok {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}
	c.current = name
	return nil
}

// GetCurrentConfig 获取当前配置
func (c *CryptoAnalyzer) GetCurrentConfig() *CryptoConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.current == "" {
		return nil
	}
	return c.configs[c.current]
}

// GetAllConfigs 获取所有配置
func (c *CryptoAnalyzer) GetAllConfigs() []*CryptoConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*CryptoConfig, 0, len(c.configs))
	for _, config := range c.configs {
		result = append(result, config)
	}
	return result
}

// GetConfig 获取指定名称的配置
func (c *CryptoAnalyzer) GetConfig(name string) *CryptoConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.configs[name]
}

// DeleteConfig 删除配置
func (c *CryptoAnalyzer) DeleteConfig(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.configs[name]; !ok {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}

	delete(c.configs, name)

	// 如果删除的是当前配置，重置当前配置
	if c.current == name {
		c.current = ""
		for k := range c.configs {
			c.current = k
			break
		}
	}
	return nil
}

// getKeyAndIV 获取密钥和IV的字节数组
func (c *CryptoAnalyzer) getKeyAndIV() ([]byte, []byte, error) {
	config := c.GetCurrentConfig()
	if config == nil {
		return nil, nil, errors.New("未选择加密配置")
	}

	var key, iv []byte

	// 尝试解析为hex，如果失败则作为普通字符串处理
	if k, err := hex.DecodeString(config.AESKey); err == nil && len(k) == 16 {
		key = k
	} else {
		key = []byte(config.AESKey)
	}

	if v, err := hex.DecodeString(config.AESIV); err == nil && len(v) == 16 {
		iv = v
	} else {
		iv = []byte(config.AESIV)
	}

	// 验证密钥长度
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, nil, fmt.Errorf("AES密钥长度必须是16、24或32字节，当前长度: %d", len(key))
	}

	if len(iv) != aes.BlockSize {
		return nil, nil, fmt.Errorf("AES IV长度必须是%d字节，当前长度: %d", aes.BlockSize, len(iv))
	}

	return key, iv, nil
}

// Decrypt 解密数据 (AES-CBC)
func (c *CryptoAnalyzer) Decrypt(data []byte) ([]byte, error) {
	key, iv, err := c.getKeyAndIV()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errors.New("数据为空")
	}

	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("加密数据长度必须是%d的倍数，当前长度: %d", aes.BlockSize, len(data))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	decrypted := make([]byte, len(data))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, data)

	// 去除PKCS7填充
	decrypted, err = pkcs7Unpad(decrypted)
	if err != nil {
		// 如果去除填充失败，返回原始解密数据
		return decrypted, nil
	}

	return decrypted, nil
}

// Encrypt 加密数据 (AES-CBC)
func (c *CryptoAnalyzer) Encrypt(data []byte) ([]byte, error) {
	key, iv, err := c.getKeyAndIV()
	if err != nil {
		return nil, err
	}

	// PKCS7填充
	data = pkcs7Pad(data, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)

	return encrypted, nil
}

// pkcs7Pad PKCS7填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpad 去除PKCS7填充
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("数据为空")
	}

	padding := int(data[len(data)-1])
	if padding > len(data) || padding > aes.BlockSize || padding == 0 {
		return nil, errors.New("无效的PKCS7填充")
	}

	// 验证填充是否正确
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("无效的PKCS7填充")
		}
	}

	return data[:len(data)-padding], nil
}

// ParsePacketHeader 解析数据包头部
func (c *CryptoAnalyzer) ParsePacketHeader(data []byte) (*PacketHeader, error) {
	config := c.GetCurrentConfig()
	if config == nil {
		return nil, errors.New("未选择加密配置")
	}

	if len(data) < config.HeaderSize {
		return nil, fmt.Errorf("数据长度(%d)小于头部大小(%d)", len(data), config.HeaderSize)
	}

	header := &PacketHeader{}

	// 解析头部字段 (大端序)
	if config.HeaderSize >= 4 {
		header.TotalLen = binary.BigEndian.Uint32(data[0:4])
	}
	if config.HeaderSize >= 8 {
		header.MsgID = binary.BigEndian.Uint32(data[4:8])
	}
	if config.HeaderSize >= 12 {
		header.Seq1 = binary.BigEndian.Uint32(data[8:12])
	}
	if config.HeaderSize >= 16 {
		header.Seq2 = binary.BigEndian.Uint32(data[12:16])
	}
	if config.HeaderSize >= 20 {
		header.Identifier = binary.BigEndian.Uint32(data[16:20])
	}

	// 查找消息名称
	if name, ok := config.MsgNames[int(header.MsgID)]; ok {
		header.MsgName = name
	} else {
		header.MsgName = fmt.Sprintf("未知消息(%d)", header.MsgID)
	}

	return header, nil
}

// ParsePacket 解析完整数据包
func (c *CryptoAnalyzer) ParsePacket(data []byte) (*DecryptedPacket, error) {
	result := &DecryptedPacket{
		RawHex: formatHex(data),
	}

	config := c.GetCurrentConfig()
	if config == nil {
		result.Error = "未选择加密配置"
		return result, errors.New(result.Error)
	}

	// 解析头部
	header, err := c.ParsePacketHeader(data)
	if err != nil {
		result.Error = fmt.Sprintf("解析头部失败: %v", err)
		return result, err
	}
	result.Header = *header

	// 提取负载
	if len(data) <= config.HeaderSize {
		result.PayloadHex = ""
		result.DecryptedHex = ""
		result.ProtobufTree = ""
		return result, nil
	}

	payload := data[config.HeaderSize:]
	result.PayloadHex = formatHex(payload)

	// 解密负载
	decrypted, err := c.Decrypt(payload)
	if err != nil {
		result.Error = fmt.Sprintf("解密失败: %v", err)
		result.DecryptedHex = result.PayloadHex // 解密失败时显示原始数据
	} else {
		result.DecryptedHex = formatHex(decrypted)

		// 尝试解析Protobuf
		pbTree := c.ParseProtobuf(decrypted, 0)
		if pbTree != "" {
			result.ProtobufTree = pbTree
		}
	}

	return result, nil
}

// ParseProtobuf 解析Protobuf数据
func (c *CryptoAnalyzer) ParseProtobuf(data []byte, skip int) string {
	if len(data) <= skip {
		return ""
	}

	// 使用现有的 _PbToJson 函数
	return _PbToJson(data, skip)
}

// ParseProtobufHex 从hex字符串解析Protobuf
func (c *CryptoAnalyzer) ParseProtobufHex(hexStr string, skip int) (string, error) {
	data, err := hex.DecodeString(strings.ReplaceAll(hexStr, " ", ""))
	if err != nil {
		return "", fmt.Errorf("hex解码失败: %v", err)
	}
	return c.ParseProtobuf(data, skip), nil
}

// DecryptHex 从hex字符串解密
func (c *CryptoAnalyzer) DecryptHex(hexStr string) (string, error) {
	data, err := hex.DecodeString(strings.ReplaceAll(hexStr, " ", ""))
	if err != nil {
		return "", fmt.Errorf("hex解码失败: %v", err)
	}

	decrypted, err := c.Decrypt(data)
	if err != nil {
		return "", err
	}

	return formatHex(decrypted), nil
}

// EncryptHex 加密并返回hex字符串
func (c *CryptoAnalyzer) EncryptHex(hexStr string) (string, error) {
	data, err := hex.DecodeString(strings.ReplaceAll(hexStr, " ", ""))
	if err != nil {
		return "", fmt.Errorf("hex解码失败: %v", err)
	}

	encrypted, err := c.Encrypt(data)
	if err != nil {
		return "", err
	}

	return formatHex(encrypted), nil
}

// ParsePacketHex 从hex字符串解析数据包
func (c *CryptoAnalyzer) ParsePacketHex(hexStr string) (*DecryptedPacket, error) {
	data, err := hex.DecodeString(strings.ReplaceAll(hexStr, " ", ""))
	if err != nil {
		return nil, fmt.Errorf("hex解码失败: %v", err)
	}

	return c.ParsePacket(data)
}

// LoadMsgNamesFromJSON 从JSON文件加载消息名称映射
func (c *CryptoAnalyzer) LoadMsgNamesFromJSON(filePath string) error {
	config := c.GetCurrentConfig()
	if config == nil {
		return errors.New("未选择加密配置")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	var msgNames map[string]string
	if err := json.Unmarshal(data, &msgNames); err != nil {
		return fmt.Errorf("JSON解析失败: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 转换为 int -> string 映射
	for k, v := range msgNames {
		var id int
		if _, err := fmt.Sscanf(k, "%d", &id); err == nil {
			config.MsgNames[id] = v
		}
	}

	return nil
}

// ExportMsgNamesToJSON 导出消息名称映射到JSON
func (c *CryptoAnalyzer) ExportMsgNamesToJSON(filePath string) error {
	config := c.GetCurrentConfig()
	if config == nil {
		return errors.New("未选择加密配置")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 转换为 string -> string 映射
	msgNames := make(map[string]string)
	for k, v := range config.MsgNames {
		msgNames[fmt.Sprintf("%d", k)] = v
	}

	data, err := json.MarshalIndent(msgNames, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// SetMsgName 设置消息名称
func (c *CryptoAnalyzer) SetMsgName(msgID int, name string) error {
	config := c.GetCurrentConfig()
	if config == nil {
		return errors.New("未选择加密配置")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	config.MsgNames[msgID] = name
	return nil
}

// GetMsgName 获取消息名称
func (c *CryptoAnalyzer) GetMsgName(msgID int) string {
	config := c.GetCurrentConfig()
	if config == nil {
		return ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if name, ok := config.MsgNames[msgID]; ok {
		return name
	}
	return ""
}

// formatHex 格式化hex输出 (每2个字符加空格)
func formatHex(data []byte) string {
	hexStr := hex.EncodeToString(data)
	var builder strings.Builder
	for i := 0; i < len(hexStr); i += 2 {
		if i > 0 {
			builder.WriteString(" ")
		}
		if i+2 <= len(hexStr) {
			builder.WriteString(strings.ToUpper(hexStr[i : i+2]))
		}
	}
	return builder.String()
}

// DecryptTCPFlow 解密TCP数据流中的所有数据包
func (c *CryptoAnalyzer) DecryptTCPFlow(theology int) ([]*DecryptedPacket, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	var results []*DecryptedPacket
	for _, socketData := range h.SocketData {
		if socketData == nil || socketData.Body == nil {
			continue
		}

		packet, err := c.ParsePacket(socketData.Body)
		if err != nil {
			// 即使解析失败也添加到结果中
			packet = &DecryptedPacket{
				RawHex: formatHex(socketData.Body),
				Error:  err.Error(),
			}
		}
		results = append(results, packet)
	}

	return results, nil
}

// ParseMultiplePackets 解析多个数据包（用于粘包情况）
func (c *CryptoAnalyzer) ParseMultiplePackets(data []byte) ([]*DecryptedPacket, error) {
	config := c.GetCurrentConfig()
	if config == nil {
		return nil, errors.New("未选择加密配置")
	}

	var results []*DecryptedPacket
	offset := 0

	for offset < len(data) {
		if len(data)-offset < config.HeaderSize {
			// 剩余数据不足以构成一个完整的头部
			break
		}

		// 读取包长度
		totalLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		if totalLen <= 0 || offset+totalLen > len(data) {
			// 包长度无效或数据不完整
			break
		}

		// 解析单个数据包
		packetData := data[offset : offset+totalLen]
		packet, err := c.ParsePacket(packetData)
		if err != nil {
			packet = &DecryptedPacket{
				RawHex: formatHex(packetData),
				Error:  err.Error(),
			}
		}
		results = append(results, packet)

		offset += totalLen
	}

	return results, nil
}

// init 初始化加密分析器
func init() {
	// 直接初始化，确保在使用前已经完成初始化
	InitCryptoAnalyzer()
}
