package mcp

import (
	"encoding/json"
	"testing"
)

// TestSchemaNode_FluentChain 验证核心：全流式链式调用、多层击穿与完全对称的 End() 回溯能力
func TestSchemaNode_FluentChain(t *testing.T) {
	schema := NewServiceSchema("security_gateway", "工业边缘安全网关控制面")
	params := NewObjectNode("params", "业务入参总线")
	params.
		// 📂 击穿测试点 A：标准对象下钻与闭合
		AddObject("nat_policy", "NAT地址转换规则", false).
			AddField("outside_ip", "string", "公网物理出口IP", true).
			AddField("inside_ip", "string", "内网目标主机IP", true).
		End(). // ↩️ 成功返回到 params 节点

		AddArray("acl_rules", "安全拦截流水线", nil, true).
			AddField("priority", "int", "匹配规则优先级", true).
			AddField("action", "string", "策略动作: accept/drop", true).
			// 甚至在数组成员内再嵌套一层子对象（如高级匹配五元组）
			AddObject("match_criteria", "五元组匹配流条件", false).
				AddField("proto", "string", "传输层协议", true).
				AddField("dport", "int", "目的安全端口", false).
			End().
		End()

	schema.SetParamsNode(params)

	mcpMap := schema.ToMCPInputSchema()

	if mcpMap["type"] != "object" {
		t.Errorf("期望顶层类型为 'object'，实际得到: %v", mcpMap["type"])
	}

	properties, ok := mcpMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("无法解析顶层 properties 结构")
	}

	if _, hasAction := properties["action"]; !hasAction {
		t.Error("缺少默认的核心控制面调配动作字段 'action'")
	}

	paramsMap, ok := properties["params"].(map[string]any)
	if !ok {
		t.Fatal("缺少核心参数总线 'params' 节点")
	}

	paramsProps, ok := paramsMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("无法读取 'params' 节点内部的子模块属性池")
	}

	natNode, hasNat := paramsProps["nat_policy"].(map[string]any)
	if !hasNat {
		t.Fatal("链式断档：'nat_policy' 节点未能正确长在 'params' 下")
	}
	if natNode["type"] != "object" {
		t.Errorf("nat_policy 应该为 object 类型，实际为: %v", natNode["type"])
	}

	aclNode, hasAcl := paramsProps["acl_rules"].(map[string]any)
	if !hasAcl {
		t.Fatal("链式断档：连续 End() 导致拓扑错位，'acl_rules' 节点丢失")
	}
	if aclNode["type"] != "array" {
		t.Errorf("acl_rules 应该为 array 类型，实际为: %v", aclNode["type"])
	}

	// 验证数组内部的成员 Items
	aclItems, ok := aclNode["items"].(map[string]any)
	if !ok {
		t.Fatal("acl_rules 数组内缺少核心属性 'items' 成员模板")
	}
	if aclItems["type"] != "object" {
		t.Errorf("数组成员模板类型应该被归一化为 'object'，实际为: %v", aclItems["type"])
	}

	// 验证数组成员内的基础参数（尤其是类型归一化映射）
	itemProps, ok := aclItems["properties"].(map[string]any)
	if !ok {
		t.Fatal("无法解析数组成员模板内部的字段池")
	}

	priorityField, ok := itemProps["priority"].(map[string]any)
	if !ok {
		t.Error("数组成员模板内丢失 'priority' 字段")
	}
	if priorityField["type"] != "integer" {
		t.Errorf("Go 类型 'int' 未能正确转换为 JSON Schema 的 'integer'，实际为: %v", priorityField["type"])
	}

	matchNode, hasMatch := itemProps["match_criteria"].(map[string]any)
	if !hasMatch {
		t.Fatal("深度嵌套断档：数组成员模板内部通过 AddObject 添加的 'match_criteria' 丢失")
	}
	if matchNode["type"] != "object" {
		t.Errorf("match_criteria 应该为 object 类型，实际为: %v", matchNode["type"])
	}

	matchProps, ok := matchNode["properties"].(map[string]any)
	if !ok {
		t.Fatal("无法解析深度嵌套对象 'match_criteria' 的内部字段")
	}

	if _, hasProto := matchProps["proto"]; !hasProto {
		t.Error("深度嵌套对象内部丢失 'proto' 字段")
	}

	t.Log("🎉 所有断言边界验证通过！完全对称的双向树结构完全达标，具备极高工程稳定性。")
}

// TestSchemaNode_JsonOutput 可视化辅助测试：打印最终产出的完美契约
func TestSchemaNode_JsonOutput(t *testing.T) {
	schema := NewServiceSchema("tunnel_vpp", "高并发 VPP 数据面隧道编排")
	params := NewObjectNode("params", "入参")

	params.AddObject("wireguard", "WG安全加密", true).
		AddField("listen_port", "int", "本地监听物理端口", true).
		AddField("private_key", "string", "隧道密钥", true).
	End()

	schema.SetParamsNode(params)

	bytes, err := json.MarshalIndent(schema.ToMCPInputSchema(), "", "  ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	t.Logf("产出的标准 MCP InputSchema 如下:\n%s", string(bytes))
}