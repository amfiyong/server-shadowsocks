package service

import (
	"encoding/json"
	"fmt"
	"github.com/xflash-panda/server-shadowsocks/internal/pkg/api"
	"github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

//InboundBuilder build Inbound config for different protocol
func InboundBuilder(nodeInfo *api.NodeInfo) (*core.InboundHandlerConfig, error) {
	var (
		streamSetting *conf.StreamConfig
		setting       json.RawMessage
	)
	inboundDetourConfig := &conf.InboundDetourConfig{}
	// Build Port
	portRange := &conf.PortRange{From: uint32(nodeInfo.ServerPort), To: uint32(nodeInfo.ServerPort)}
	inboundDetourConfig.PortRange = portRange
	// Build Tag
	inboundDetourConfig.Tag = fmt.Sprintf("%s_%d", protocol, nodeInfo.ServerPort)
	// SniffingConfig
	sniffingConfig := &conf.SniffingConfig{
		Enabled: false,
	}
	inboundDetourConfig.SniffingConfig = sniffingConfig

	// Build Protocol and Protocol setting
	proxySetting := &conf.ShadowsocksServerConfig{}
	randomPasswd := uuid.New()
	defaultUser := &conf.ShadowsocksUserConfig{
		Cipher:   nodeInfo.Cipher,
		Password: randomPasswd.String(),
		Level:    0,
	}
	proxySetting.Users = append(proxySetting.Users, defaultUser)
	proxySetting.NetworkList = &conf.NetworkList{TCP, UDP}

	setting, err := json.Marshal(proxySetting)
	if err != nil {
		return nil, fmt.Errorf("marshal proxy %s config fialed: %s", protocol, err)
	}

	// Build streamSettings
	streamSetting = new(conf.StreamConfig)
	transportProtocol := conf.TransportProtocol(TCP)

	tcpSetting := &conf.TCPConfig{
		AcceptProxyProtocol: false,
		HeaderConfig:        nil,
	}
	streamSetting.TCPSettings = tcpSetting
	streamSetting.Network = &transportProtocol

	inboundDetourConfig.Protocol = protocol
	inboundDetourConfig.StreamSetting = streamSetting
	inboundDetourConfig.Settings = &setting

	return inboundDetourConfig.Build()
}
