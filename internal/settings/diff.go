package settings

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

var yamlUnmarshalerType = reflect.TypeFor[yaml.Unmarshaler]()

// marshalNonDefault writes s as YAML, keeping only values that differ from
// DefaultSettings(), with the exception of the version field.
func marshalNonDefault(s *Settings) ([]byte, error) {
	def := DefaultSettings()
	node, isDef, err := diffNode(reflect.ValueOf(s).Elem(), reflect.ValueOf(def))
	if err != nil {
		return nil, err
	}
	if isDef || node == nil {
		node = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	// Always write version
	setSettingsVersion(node, currentSettingsVersion)
	return yaml.Marshal(node)
}

// diffNode returns the YAML representation of cur, or (nil, true) when cur
// equals def and the parent should drop the field entirely.
func diffNode(cur, def reflect.Value) (*yaml.Node, bool, error) {
	if !cur.IsValid() || !def.IsValid() {
		return nil, true, nil
	}
	switch cur.Kind() {
	case reflect.Pointer, reflect.Interface:
		curNil, defNil := cur.IsNil(), def.IsNil()
		if curNil && defNil {
			return nil, true, nil
		}
		if curNil || defNil {
			return encodeNode(cur.Interface())
		}
		return diffNode(cur.Elem(), def.Elem())
	case reflect.Struct:
		return diffStruct(cur, def)
	case reflect.Map:
		return diffMap(cur, def)
	default:
		return diffAtomic(cur, def)
	}
}

func encodeNode(v any) (*yaml.Node, bool, error) {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		return nil, false, err
	}
	return &n, false, nil
}

// diffAtomic compares cur and def by their marshaled bytes and emits cur
// verbatim when they differ. Used for slices, scalars, opaque maps, and any
// type the structural walker chooses not to descend into.
func diffAtomic(cur, def reflect.Value) (*yaml.Node, bool, error) {
	curBytes, err := yaml.Marshal(cur.Interface())
	if err != nil {
		return nil, false, err
	}
	defBytes, err := yaml.Marshal(def.Interface())
	if err != nil {
		return nil, false, err
	}
	if bytes.Equal(curBytes, defBytes) {
		return nil, true, nil
	}
	return encodeNode(cur.Interface())
}

func yamlKey(field reflect.StructField) (key string, skip bool) {
	tag := field.Tag.Get("yaml")
	if tag == "-" {
		return "", true
	}
	if tag == "" {
		return strings.ToLower(field.Name), false
	}
	name := strings.SplitN(tag, ",", 2)[0]
	if name == "" {
		return strings.ToLower(field.Name), false
	}
	return name, false
}

func diffStruct(cur, def reflect.Value) (*yaml.Node, bool, error) {
	t := cur.Type()
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		key, skip := yamlKey(field)
		if skip {
			continue
		}
		childNode, isDef, err := diffNode(cur.Field(i), def.Field(i))
		if err != nil {
			return nil, false, err
		}
		if isDef {
			continue
		}
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			childNode,
		)
	}
	if len(out.Content) == 0 {
		return nil, true, nil
	}
	return out, false, nil
}

// diffMap diffs maps on a per-key basis when the element type knows its own
// defaults (i.e. implements yaml.Unmarshaler — every settings sub-struct does,
// since their UnmarshalYAML assigns Default<T>() first). For element types
// without that affordance (primitive values, maps of primitives) we treat the
// whole map atomically.
func diffMap(cur, def reflect.Value) (*yaml.Node, bool, error) {
	elemType := cur.Type().Elem()
	hasPerKeyDefault := reflect.PointerTo(elemType).Implements(yamlUnmarshalerType) ||
		elemType.Implements(yamlUnmarshalerType)
	if !hasPerKeyDefault || cur.IsNil() {
		return diffAtomic(cur, def)
	}

	keys := cur.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})

	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range keys {
		var (
			perKeyDef    reflect.Value
			keyInDefault bool
		)
		if !def.IsNil() {
			if dv := def.MapIndex(k); dv.IsValid() {
				perKeyDef = dv
				keyInDefault = true
			}
		}
		if !keyInDefault {
			ptr := reflect.New(elemType)
			if err := yaml.Unmarshal([]byte("{}"), ptr.Interface()); err != nil {
				return nil, false, err
			}
			perKeyDef = ptr.Elem()
		}

		// Addressable copies so structural recursion (which may take field
		// addresses) is well-defined.
		curCopy := reflect.New(elemType).Elem()
		curCopy.Set(cur.MapIndex(k))
		defCopy := reflect.New(elemType).Elem()
		defCopy.Set(perKeyDef)

		childNode, isDef, err := diffNode(curCopy, defCopy)
		if err != nil {
			return nil, false, err
		}
		if isDef {
			if keyInDefault {
				continue
			}
			// User added this key; preserve its existence even though every
			// field happens to equal the synthesized default.
			childNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		}
		keyNode := &yaml.Node{}
		if err := keyNode.Encode(k.Interface()); err != nil {
			return nil, false, err
		}
		out.Content = append(out.Content, keyNode, childNode)
	}

	if len(out.Content) == 0 {
		return diffAtomic(cur, def)
	}
	return out, false, nil
}
