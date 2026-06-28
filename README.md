# CPE MCP Server - 边缘路由控制总线

`cpe-mcp-server` 是一款面向下一代高可靠 SASE/SD-WAN 边缘网关的 **Model Context Protocol (MCP)** 服务端实现。
系统采用基于元数据驱动的微内核插件架构（Micro-kernel Architecture），将北向大模型工具契约（JSON Schema）与南向底盘物理驱动（FRR Zebra VTY / Netlink）彻底解耦，在保障极致高并发安全的同时，原生交付了边缘本地防失联自愈（Rollback）与掉电安全（Physical Save）能力。

## 🎨 核心工程美学

* **零开机实例化（Zero-Instantiation Init）**：组件注册（`RegisterRouting`）时将 Action 与 Schema 静态常驻内存，开机自检与 Schema 生成完全免除 Factory 调用，斩断重型南向逻辑提前触发的风险。
* **绝对无状态并发（Stateless Concurrent Safety）**：彻底消灭驱动插件的结构体成员变量（Fields）。参数按需解包、栈上分配（Stack Allocation），天然防御大模型高并发调度下的多线程踩踏死锁风险。
* **北向条件级强约束（OneOf Schema Alignment）**：动态聚合子插件 Schema，输出严谨的 `oneOf` 条件级约束，让大模型在传输层即完成参数校验，从根源上消除模型幻觉。
* **两阶段提交与防失联（2-Phase Commit & Anti-Disconnect）**：原生支持 `Apply` ➡️ `Verify (Watchdog)` ➡️ `Rollback / Save` 的事务控制流，保护 CPE 在跨境或复杂网络变更下永不失联。

---

## 🏗️ 系统架构拓扑
```shell
             ┌──────────────────────────────────────────────┐
             │            Northbound LLM / MCP Client       │
             └──────────────────────┬───────────────────────┘
                                    │ (JSON-RPC over SSE/Stdio)
                                    ▼
             ┌──────────────────────────────────────────────┐
             │             mcp.Registry  (总网关)            │
             └──────────────────────┬───────────────────────┘
                                    │
                                    ▼ [ExecuteTool]
┌───────────────────────────────────────────────────────────────────────────┐
│ RouteController (超级路由控制器总线)                                         │
│   ├── GetSchema() -> 动态多分支 oneOf 契约拼装                               │
│   └── ExecuteTool() -> 按需延迟懒加载实例化                                   │
└────────┬───────────────────────┬───────────────────────┬──────────────────┘
         │                       │                       │
         ▼ (Apply/Remove)        ▼ (Rollback)            ▼ (Save)
┌───────────────────────┐┌───────────────────────┐┌───────────────────────┐
│      Controller       ││       Rollable        ││         Saver         │
├───────────────────────┤├───────────────────────┤├───────────────────────┤
│ 穿透下刷至 FRR Zebra    ││ 5s 连通性测试失败       ││ 物理 Page Cache 强制    │
│ 同时写入 /etc/ 缓存页    ││ 本地时钟触发精准拔除     ││ 刷盘，规避整机掉电损坏    │
└───────────────────────┘└───────────────────────┘└───────────────────────┘
```
## 🛠️ 快速开始
1. 环境依赖
- Go 1.20+
- Linux 边缘底盘（需常驻 frr.zebra 进程并开启 VTYSH 权限）
- 运行配置常驻根路径：/etc/svxnetworks/routes/static/

2. 编译与本地自检
退回到项目根目录，通过目录级编译：
```Bash
# 正式编译
go build -o cpe-mcp-server .

# 带 Debug 深度自检树模式拉起（以 SSE 模式为例）
./cpe-mcp-server -m sse --debug=true
```
网关成功拉起后，会在 [INIT] 日志中打印出零实例化开销的静态自检组件树：
```Bash
2026/06/28 15:30:12 [INIT] (Total Services: 1)
2026/06/28 15:30:12   ► [Service] ROUTE        -> 状态: ONLINE
2026/06/28 15:30:12      └── ⚙️  [Action] static -> apply
2026/06/28 15:30:12      └── ⚙️  [Action] static -> remove
2026/06/28 15:30:12 [INIT] (Total Tools: 1)
2026/06/28 15:30:12   ⚙️  [Tool]    configure_route      | 描述: 对边缘 Linux route 矩阵下发动态控制策略
```
3. 大模型（或北向控制面）如何调用
当 MCP 客户端（如 Claude、网关编排系统等）向你的网关发送请求时，通过 tools/list 得到的 configure_route 会携带由 oneOf 动态拼装的强约束。大模型只要下发动作，必须符合以下三种标准的声明式报文规范：
- 场景 1：下刷一条新静态路由 (apply)
当大模型识别到引流需求时，下发此 JSON 载荷。主控总线会将参数完整解包到局部变量中并穿透到底盘：
```JSON
{
  "routing_type": "static",
  "action": "apply",
  "arguments": {
    "destination": "172.16.100.0",
    "netmask": "255.255.255.0",
    "nexthop": "10.0.0.1",
    "preference": 10
  }
}
```

- 场景 2：撤销一条静态路由 (remove)
```JSON
{
  "routing_type": "static",
  "action": "remove",
  "arguments": {
    "destination": "172.16.100.0",
    "netmask": "255.255.255.0",
    "nexthop": "10.0.0.1"
  }
}
```

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