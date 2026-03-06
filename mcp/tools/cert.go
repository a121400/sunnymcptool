package tools

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/a121400/sunnymcptool/CommAnd"
	"github.com/a121400/sunnymcptool/mcp"
	"github.com/qtgolang/SunnyNet/public"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "cert_install",
		Description: "安装SunnyNet默认CA证书到系统信任列表",
		InputSchema: noParamsSchema(),
		Handler:     toolCertInstallHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "cert_export",
		Description: "导出SunnyNet默认CA证书到指定路径",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "证书导出路径（包含文件名，如：C:/cert/SunnyNet.cer）",
				},
			},
			"required": []string{"path"},
		},
		Handler: toolCertExportHandler,
	})
}

func toolCertInstallHandler(args map[string]interface{}) (interface{}, error) {
	result := CommAnd.InstallCert([]byte(public.RootCa))
	success := strings.Contains(result, "成功") || strings.Contains(strings.ToLower(result), "success")
	return map[string]interface{}{
		"success": success,
		"message": result,
	}, nil
}

func toolCertExportHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	path := v.RequireString("path")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errors.New("路径不能为空")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".cer") {
		path += ".cer"
	}
	err := os.WriteFile(path, []byte(public.RootCa), 0644)
	if err != nil {
		return nil, fmt.Errorf("导出证书失败: %s", err.Error())
	}
	return map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "证书已导出",
	}, nil
}
