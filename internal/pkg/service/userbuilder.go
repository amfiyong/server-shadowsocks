package service

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/xflash-panda/server-shadowsocks/internal/pkg/api"
	cProtocol "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	"strings"
)

func buildUser(tag string, userInfo []*api.UserInfo, method string) (users []*cProtocol.User) {
	users = make([]*cProtocol.User, 0)
	cypherMethod := cipherFromString(method)
	log.Infof("user cypher method: %s", cypherMethod)
	for _, user := range userInfo {
		ssAccount := &shadowsocks.Account{
			Password:   user.UUID,
			CipherType: cypherMethod,
		}
		users = append(users, &cProtocol.User{
			Level:   0,
			Email:   buildUserEmail(tag, user.ID, user.UUID),
			Account: serial.ToTypedMessage(ssAccount),
		})
	}
	return users
}

func buildUserEmail(tag string, id int, uuid string) string {
	return fmt.Sprintf("%s|%d|%s", tag, id, uuid)
}

func cipherFromString(c string) shadowsocks.CipherType {
	switch strings.ToLower(c) {
	case "aes-128-gcm", "aead_aes_128_gcm":
		return shadowsocks.CipherType_AES_128_GCM
	case "aes-256-gcm", "aead_aes_256_gcm":
		return shadowsocks.CipherType_AES_256_GCM
	case "chacha20-poly1305", "aead_chacha20_poly1305", "chacha20-ietf-poly1305":
		return shadowsocks.CipherType_CHACHA20_POLY1305
	case "none", "plain":
		return shadowsocks.CipherType_NONE
	default:
		return shadowsocks.CipherType_UNKNOWN
	}
}
