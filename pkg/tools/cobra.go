package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pluginmd/haravan-cli/internal/cmdutil"
)

// DepsBuilder returns a ready Deps from a cobra command. It resolves auth,
// builds the Haravan client, and wires IO — once, when the command runs.
type DepsBuilder func(cmd *cobra.Command) (*Deps, error)

// BindCobra turns a Tool into a cobra.Command. The returned command
// registers one pflag per declared Flag, parses them into an Input map,
// builds Deps via the provided builder, invokes Handler, and writes the
// result to stdout.
//
// The command name is the portion of Tool.Name after the leading
// "haravan_<category>_" prefix, e.g. "haravan_orders_list" → "list".
func BindCobra(t *Tool, f *cmdutil.Factory, build DepsBuilder) *cobra.Command {
	subName := deriveSubcommandName(t)

	long := t.Long
	if long == "" {
		long = t.Short
	}

	cmd := &cobra.Command{
		Use:   subName,
		Short: t.Short,
		Long:  long,
	}

	// Store one pointer per flag so we can materialize values after Parse.
	type slot struct {
		flag  Flag
		str   *string
		i64   *int64
		i     *int
		b     *bool
		slist *[]string
	}
	slots := make([]slot, 0, len(t.Flags))

	for _, fl := range t.Flags {
		s := slot{flag: fl}
		switch fl.Type {
		case FlagString, FlagJSON, FlagDateTimeISO:
			def, _ := fl.Default.(string)
			s.str = stringFlag(cmd.Flags(), fl, def)
		case FlagInt:
			def, _ := fl.Default.(int)
			s.i = intFlag(cmd.Flags(), fl, def)
		case FlagInt64:
			def, _ := fl.Default.(int64)
			s.i64 = int64Flag(cmd.Flags(), fl, def)
		case FlagBool:
			def, _ := fl.Default.(bool)
			s.b = boolFlag(cmd.Flags(), fl, def)
		case FlagStringList:
			var def []string
			if d, ok := fl.Default.([]string); ok {
				def = d
			}
			s.slist = stringSliceFlag(cmd.Flags(), fl, def)
		default:
			def, _ := fl.Default.(string)
			s.str = stringFlag(cmd.Flags(), fl, def)
		}
		if fl.Required {
			_ = cmd.MarkFlagRequired(fl.Name)
		}
		slots = append(slots, s)
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		input := Input{}
		for _, s := range slots {
			if !cmd.Flags().Changed(s.flag.Name) && s.flag.Default == nil {
				continue
			}
			val, err := materialize(s, cmd.Flags())
			if err != nil {
				return fmt.Errorf("flag --%s: %w", s.flag.Name, err)
			}
			if val != nil {
				input[s.flag.Name] = val
			}
		}

		deps, err := build(cmd)
		if err != nil {
			return err
		}
		res, err := t.Handler(cmd.Context(), deps, input)
		if err != nil {
			return err
		}
		return writeResult(f, res)
	}
	return cmd
}

func materialize(s struct {
	flag  Flag
	str   *string
	i64   *int64
	i     *int
	b     *bool
	slist *[]string
}, flags *pflag.FlagSet) (any, error) {
	switch s.flag.Type {
	case FlagString, FlagDateTimeISO:
		return *s.str, nil
	case FlagJSON:
		return resolveJSONValue(*s.str)
	case FlagInt:
		return *s.i, nil
	case FlagInt64:
		return *s.i64, nil
	case FlagBool:
		return *s.b, nil
	case FlagStringList:
		return append([]string{}, *s.slist...), nil
	default:
		return *s.str, nil
	}
}

func stringFlag(fs *pflag.FlagSet, fl Flag, def string) *string {
	var out string
	if fl.Short != "" {
		fs.StringVarP(&out, fl.Name, fl.Short, def, fl.Description)
	} else {
		fs.StringVar(&out, fl.Name, def, fl.Description)
	}
	return &out
}

func intFlag(fs *pflag.FlagSet, fl Flag, def int) *int {
	var out int
	if fl.Short != "" {
		fs.IntVarP(&out, fl.Name, fl.Short, def, fl.Description)
	} else {
		fs.IntVar(&out, fl.Name, def, fl.Description)
	}
	return &out
}

func int64Flag(fs *pflag.FlagSet, fl Flag, def int64) *int64 {
	var out int64
	if fl.Short != "" {
		fs.Int64VarP(&out, fl.Name, fl.Short, def, fl.Description)
	} else {
		fs.Int64Var(&out, fl.Name, def, fl.Description)
	}
	return &out
}

func boolFlag(fs *pflag.FlagSet, fl Flag, def bool) *bool {
	var out bool
	if fl.Short != "" {
		fs.BoolVarP(&out, fl.Name, fl.Short, def, fl.Description)
	} else {
		fs.BoolVar(&out, fl.Name, def, fl.Description)
	}
	return &out
}

func stringSliceFlag(fs *pflag.FlagSet, fl Flag, def []string) *[]string {
	var out []string
	if fl.Short != "" {
		fs.StringSliceVarP(&out, fl.Name, fl.Short, def, fl.Description)
	} else {
		fs.StringSliceVar(&out, fl.Name, def, fl.Description)
	}
	return &out
}

func deriveSubcommandName(t *Tool) string {
	name := t.Name
	prefix := fmt.Sprintf("haravan_%s_", t.Category)
	if strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	// Smart tools live under `haravan-cli smart <name>` without the
	// haravan_ prefix in the sub-slug.
	if strings.HasPrefix(name, "hrv_") {
		return strings.TrimPrefix(name, "hrv_")
	}
	return name
}

func writeResult(f *cmdutil.Factory, r *Result) error {
	if r == nil {
		return nil
	}
	if len(r.Data) > 0 {
		var pretty []byte
		if buf, err := prettyJSON(r.Data); err == nil {
			pretty = buf
		} else {
			pretty = r.Data
		}
		_, err := fmt.Fprintln(f.IOStreams.Out, string(pretty))
		if r.IsError && err == nil {
			return fmt.Errorf("tool reported error")
		}
		return err
	}
	if r.Text != "" {
		_, err := fmt.Fprintln(f.IOStreams.Out, r.Text)
		if r.IsError && err == nil {
			return fmt.Errorf("tool reported error")
		}
		return err
	}
	return nil
}

func prettyJSON(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}
