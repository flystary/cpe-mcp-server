# cp-mcp-server

## 完整调用链
```bash
MCP Engine (ProcessMessage)
    ↓
Registry.ExecuteTool() (触发绑定的闭包 Handler)
    ↓
RouteService.Execute() (拆解路由协议 Protocol 类型，向内透传数据)
    ↓
RouteEngine.Dispatch() (通过读写锁安全调度指定协议驱动)
    ↓
Static.Validate() (驱动层对扁平参数执行强类型反序列化与安全清洗)
    ↓
Static.Dispatch() (承接校验后的 Payload 并按 Action 进行路由分流)
    ↓
Static.apply() (执行物理存储持久化，并通过 vty.Zebra 下刷内核路由表)
```