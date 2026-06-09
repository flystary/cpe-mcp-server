package vty

import (
	"context"
	"time"
)

const bgpdSocketPath = "/var/run/frr/bgpd.vty"

type BgpClient struct {
	vtyClient
}

var bgp = &BgpClient{
	vtyClient: vtyClient{socketPath: bgpdSocketPath},
}

// BgpExecute 写
func BgpExecute(ctx context.Context, commands []string) error {
	return bgp.execute(ctx, commands)
}

// BgpQuery 读
func BgpQuery(ctx context.Context, cmd string) (map[string][]RouteEntry, error) {
	raw, err := bgp.roundTrip(ctx, []string{cmd, "quit"}, time.Second*2)
	if err != nil {
		return nil, err
	}
	// 这里通过泛型解析器，直接转换成 map 数据供调用者使用
	return ParseFRRJSON[map[string][]RouteEntry](raw)

}
