package vty

import (
	"context"
	"time"
)

const zebraSocketPath = "/var/run/frr/zebra.vty"

type ZebraClient struct {
	vtyClient
}

var zebra = &ZebraClient{
	vtyClient: vtyClient{socketPath: zebraSocketPath},
}

// ZebraExecute 写
func ZebraExecute(ctx context.Context, commands []string) error {
	return zebra.execute(ctx, commands)
}

// ZebraQuery 读
func ZebraQuery(ctx context.Context, cmd string) (map[string][]RouteEntry, error) {
	raw, err := zebra.roundTrip(ctx, []string{cmd, "quit"}, time.Second*2)
	if err != nil {
		return nil, err
	}
	// 这里通过泛型解析器，直接转换成 map 数据供调用者使用
	return ParseFRRJSON[map[string][]RouteEntry](raw)
}
