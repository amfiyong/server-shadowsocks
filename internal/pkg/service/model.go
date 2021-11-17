package service

const (
	protocol = "shadowsocks"
	TCP      = "tcp"
	UDP      = "udp"
)

// Service is the interface of all the services running in the panel
type Service interface {
	Start() error
	Close() error
}
