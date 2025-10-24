package enum

type Protocol string

const (
	ProtocolHTTP  = "http"
	ProtocolHTTPS = "https"
	ProtocolGRPC  = "grpc"
)

func (p Protocol) String() string {
	return string(p)
}
