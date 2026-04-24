package tools

import (
	"encoding/json"
	"testing"
)

func TestBuildJSONSchemaProducesExpectedShape(t *testing.T) {
	tool := &Tool{
		Name: "haravan_test",
		Flags: []Flag{
			{Name: "order_id", Type: FlagInt64, Description: "Order ID", Required: true},
			{Name: "status", Type: FlagString, Description: "status", Enum: []string{"open", "closed"}},
			{Name: "tags", Type: FlagStringList, Description: "tags"},
			{Name: "fetch_all", Type: FlagBool, Description: "paginate"},
			{Name: "body", Type: FlagJSON, Description: "payload"},
			{Name: "date_from", Type: FlagDateTimeISO, Description: "from"},
		},
	}
	schema := BuildJSONSchema(tool)
	raw, _ := json.Marshal(schema)
	var got map[string]any
	_ = json.Unmarshal(raw, &got)

	if got["type"] != "object" {
		t.Errorf("type: %v", got["type"])
	}
	if got["additionalProperties"] != false {
		t.Errorf("additionalProperties: %v", got["additionalProperties"])
	}

	props := got["properties"].(map[string]any)
	if props["order_id"].(map[string]any)["type"] != "integer" {
		t.Errorf("order_id type: %v", props["order_id"])
	}
	if props["tags"].(map[string]any)["type"] != "array" {
		t.Errorf("tags type: %v", props["tags"])
	}
	if props["fetch_all"].(map[string]any)["type"] != "boolean" {
		t.Errorf("fetch_all type: %v", props["fetch_all"])
	}
	if t2 := props["body"].(map[string]any)["type"]; t2 != nil {
		t.Errorf("body should be untyped (any JSON), got %v", t2)
	}
	dt := props["date_from"].(map[string]any)
	if dt["type"] != "string" || dt["format"] != "date-time" {
		t.Errorf("date_from: %v", dt)
	}
	st := props["status"].(map[string]any)
	if _, ok := st["enum"]; !ok {
		t.Errorf("status missing enum")
	}

	req := got["required"].([]any)
	if len(req) != 1 || req[0] != "order_id" {
		t.Errorf("required: %v", req)
	}
}
