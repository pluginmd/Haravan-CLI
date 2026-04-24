package tools

// FlagType enumerates the primitive shapes a Flag can take. It drives both
// cobra flag registration and JSON Schema generation.
type FlagType string

const (
	FlagString      FlagType = "string"
	FlagInt         FlagType = "int"
	FlagInt64       FlagType = "int64"
	FlagBool        FlagType = "bool"
	FlagStringList  FlagType = "string_list"  // comma-separated on CLI, JSON array over MCP
	FlagJSON        FlagType = "json"         // inline JSON string or @file path on CLI
	FlagDateTimeISO FlagType = "datetime_iso" // ISO 8601 timestamp (string under the hood)
)

// Flag describes one input parameter for a Tool.
//
// The same Flag is rendered as:
//   - a cobra flag named --<Name> (or -<Short> if set)
//   - a JSON Schema property under /input/properties/<Name>
//
// Required flags become required schema properties. Default values are
// used by cobra but NOT advertised in the schema (MCP clients should
// send explicit values).
type Flag struct {
	Name        string   // snake_case, used as both CLI flag name and JSON key
	Short       string   // optional single-char shorthand for cobra (-l, -p, etc.)
	Type        FlagType // one of the FlagType constants
	Description string
	Required    bool
	Default     any      // default value for cobra; type must match Type
	Enum        []string // optional enumeration (validated at schema level)
	Example     string   // optional example value used in help text
}
