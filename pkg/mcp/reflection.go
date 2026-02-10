package mcp

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-json"

	"github.com/Wh1teJ0ker/GoT0MCP/pkg/abi"
)

// AddTool 使用 Go 函数注册一个工具，并自动生成 Schema。
// 函数签名必须为: func(context.Context, ArgumentsStruct) (Result, error)
// ArgumentsStruct 的字段应包含 `json` 标签。
// `jsonschema` 标签可用于提供字段描述，例如 `jsonschema:"description=这是一个描述"`。
func (s *Server) AddTool(name string, description string, fn any) error {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("fn 必须是一个函数")
	}

	// 验证签名
	// 输入: (context.Context, struct)
	if fnType.NumIn() != 2 {
		return fmt.Errorf("fn 必须接收 2 个参数: context.Context 和 参数结构体")
	}
	if fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return fmt.Errorf("第一个参数必须是 context.Context")
	}
	argsType := fnType.In(1)
	if argsType.Kind() != reflect.Struct {
		return fmt.Errorf("第二个参数必须是一个结构体")
	}

	// 输出: (any, error)
	if fnType.NumOut() != 2 {
		return fmt.Errorf("fn 必须返回 2 个值: result 和 error")
	}
	if fnType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("第二个返回值必须是 error")
	}

	// 生成 Schema
	schema := generateSchema(argsType)

	spec := abi.ToolSpec{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}

	// 创建 Handler
	handler := func(ctx context.Context, argsRaw json.RawMessage) (map[string]any, error) {
		// 创建参数结构体的新实例
		argsPtr := reflect.New(argsType)
		argsInterface := argsPtr.Interface()

		if len(argsRaw) > 0 {
			if err := json.Unmarshal(argsRaw, argsInterface); err != nil {
				return nil, fmt.Errorf("无法解析参数: %w", err)
			}
		}

		// 调用函数
		results := fnValue.Call([]reflect.Value{reflect.ValueOf(ctx), argsPtr.Elem()})

		// 检查错误
		errVal := results[1]
		if !errVal.IsNil() {
			return nil, errVal.Interface().(error)
		}

		// 返回结果
		resVal := results[0]
		resInterface := resVal.Interface()

		// 格式化输出
		// MCP 协议期望工具结果为内容列表。
		// 这里我们将结果转换为字符串或 JSON 字符串。
		var contentText string
		if s, ok := resInterface.(string); ok {
			contentText = s
		} else {
			// 尝试 Marshal 为 JSON 字符串
			b, _ := json.Marshal(resInterface)
			contentText = string(b)
		}

		return map[string]any{
			"content": []map[string]string{
				{
					"type": "text",
					"text": contentText,
				},
			},
		}, nil
	}

	s.RegisterTool(spec, handler)
	return nil
}

func generateSchema(t reflect.Type) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]any)
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]

		prop := map[string]any{}

		// 类型映射
		switch field.Type.Kind() {
		case reflect.String:
			prop["type"] = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			prop["type"] = "integer"
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			prop["type"] = "integer"
		case reflect.Float32, reflect.Float64:
			prop["type"] = "number"
		case reflect.Bool:
			prop["type"] = "boolean"
		case reflect.Slice:
			prop["type"] = "array"
			// 简单数组支持
			prop["items"] = map[string]any{"type": "string"} // 默认回退
		default:
			prop["type"] = "string" // 默认回退
		}

		// 从 jsonschema 标签获取描述
		desc := field.Tag.Get("jsonschema")
		if desc != "" {
			// 简单解析: jsonschema:"description=foo"
			if strings.HasPrefix(desc, "description=") {
				prop["description"] = strings.TrimPrefix(desc, "description=")
			} else {
				prop["description"] = desc
			}
		}

		properties[name] = prop
		required = append(required, name)
	}

	schema["required"] = required
	return schema
}
