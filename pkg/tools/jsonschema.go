package tools

// BuildJSONSchema returns a JSON Schema (draft-07 style) object describing
// the tool's inputs. Shape:
//
//	{
//	  "type": "object",
//	  "properties": { "<flag>": { "type": ..., "description": ... }, ... },
//	  "required": [ "<flag>", ... ],
//	  "additionalProperties": false
//	}
//
// The output is ready to embed as the "input_schema" of an MCP tool.
func BuildJSONSchema(t *Tool) map[string]any {
	props := map[string]any{}
	var required []string

	for _, f := range t.Flags {
		entry := map[string]any{"description": f.Description}
		switch f.Type {
		case FlagString, FlagDateTimeISO:
			entry["type"] = "string"
			if f.Type == FlagDateTimeISO {
				entry["format"] = "date-time"
			}
		case FlagInt, FlagInt64:
			entry["type"] = "integer"
		case FlagBool:
			entry["type"] = "boolean"
		case FlagStringList:
			entry["type"] = "array"
			entry["items"] = map[string]any{"type": "string"}
		case FlagJSON:
			// JSON means "any valid JSON value" — deliberately untyped so
			// callers can send objects, arrays, or primitives per endpoint.
			// Leave "type" unset.
		default:
			entry["type"] = "string"
		}
		if len(f.Enum) > 0 {
			anys := make([]any, len(f.Enum))
			for i, v := range f.Enum {
				anys[i] = v
			}
			entry["enum"] = anys
		}
		if f.Example != "" {
			entry["examples"] = []any{f.Example}
		}
		props[f.Name] = entry
		if f.Required {
			required = append(required, f.Name)
		}
	}

	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
