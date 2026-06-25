package mcp

import "strings"

type SchemaNode struct {
	Name        string                 `json:"-"`
	Type        string                 `json:"type"` // "string", "integer", "boolean", "object", "array"
	Description string                 `json:"description,omitempty"`
	Properties  map[string]*SchemaNode `json:"properties,omitempty"` // 当 Type == "object" 时使用
	Items       *SchemaNode            `json:"items,omitempty"`      // 当 Type == "array" 时使用
	Required    []string               `json:"required,omitempty"`   // 当 Type == "object" 时收集必填子项
	parent      *SchemaNode            `json:"-"`                    // 影子双向指针
}

func NewObjectNode(name, desc string) *SchemaNode {
	return &SchemaNode{
		Name:        name,
		Type:        "object",
		Description: desc,
		Properties:  make(map[string]*SchemaNode),
	}
}

func NewArrayNode(name, desc string, itemNode *SchemaNode) *SchemaNode {
	return &SchemaNode{
		Name:        name,
		Type:        "array",
		Description: desc,
		Items:       itemNode,
	}
}

// End 用于闭合当前子对象/数组模板的上下文，返回父节点
func (n *SchemaNode) End() *SchemaNode {
	if n.parent != nil {
		return n.parent
	}
	return n
}

// AddField 链式添加基础叶子字段 (返回当前节点，用于横向平铺)
func (n *SchemaNode) AddField(name, fType, desc string, required bool) *SchemaNode {
	if n.Properties == nil {
		n.Properties = make(map[string]*SchemaNode)
	}
	if fType == "int" {
		fType = "integer"
	}

	n.Properties[name] = &SchemaNode{
		Type:        fType,
		Description: desc,
		parent:      n, // 补齐基础字段的 parent，防止异常调用 End() 发生崩溃
	}
	if required {
		n.Required = append(n.Required, name)
	}
	return n
}

// AddObject 链式嵌套添加一个子对象 (下钻：返回子对象节点)
func (n *SchemaNode) AddObject(name, desc string, required bool) *SchemaNode {
	if n.Properties == nil {
		n.Properties = make(map[string]*SchemaNode)
	}

	subObj := NewObjectNode(name, desc)
	subObj.parent = n

	n.Properties[name] = subObj
	if required {
		n.Required = append(n.Required, name)
	}
	return subObj
}

// AddArray 🛠️【完全体修正】：支持深度下钻数组内部的对象模板
func (n *SchemaNode) AddArray(name, desc string, itemNode *SchemaNode, required bool) *SchemaNode {
	if n.Properties == nil {
		n.Properties = make(map[string]*SchemaNode)
	}

	// 如果 itemNode 为空，默认创建一个 object 骨架作为数组成员模板
	if itemNode == nil {
		itemNode = NewObjectNode(name+"_item", desc+"原子成员模板")
	}

	// 关键建立双向绑定
	arrNode := NewArrayNode(name, desc, itemNode)
	arrNode.parent = n
	itemNode.parent = arrNode // 核心：让模板的 End() 能够一路杀回 Array 节点

	n.Properties[name] = arrNode
	if required {
		n.Required = append(n.Required, name)
	}

	return itemNode
}

func (n *SchemaNode) ToMap() map[string]any {
	res := map[string]any{
		"type": n.Type,
	}
	if n.Description != "" {
		res["description"] = n.Description
	}
	if n.Type == "object" && len(n.Properties) > 0 {
		props := make(map[string]any)
		for k, v := range n.Properties {
			props[k] = v.ToMap()
		}
		res["properties"] = props
		if len(n.Required) > 0 {
			res["required"] = n.Required
		}
	}

	if n.Type == "array" && n.Items != nil {
		res["items"] = n.Items.ToMap()
	}

	return res
}

type ServiceSchema struct {
	Name        string
	Description string
	rootNode    *SchemaNode
}

func NewServiceSchema(name, desc string) *ServiceSchema {
	root := NewObjectNode(name, desc)
	root.AddField("action", "string", "要执行的控制面调配指令动作", true)

	return &ServiceSchema{
		Name:        strings.ToLower(name),
		Description: desc,
		rootNode:    root,
	}
}

func (s *ServiceSchema) SetParamsNode(paramsNode *SchemaNode) *ServiceSchema {
	if s.rootNode.Properties == nil {
		s.rootNode.Properties = make(map[string]*SchemaNode)
	}
	s.rootNode.Properties["params"] = paramsNode
	s.rootNode.Required = append(s.rootNode.Required, "params")
	return s
}

func (s *ServiceSchema) ToMCPInputSchema() map[string]any {
	return s.rootNode.ToMap()
}

func (s *ServiceSchema) ToDumpExtensions() map[string]any {
	if params, ok := s.rootNode.Properties["params"]; ok {
		return params.ToMap()
	}
	return nil
}
