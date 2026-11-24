// Package settingsdoc generates Markdown documentation for settings.Settings
// and nested configuration structs from type information and AST comments.
package settingsdoc

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/getmiroir/trading-service/internal/settings"

	"golang.org/x/tools/go/packages"
)

const settingsPkg = "github.com/getmiroir/trading-service/internal/settings"

var documentedPkgPaths = map[string]bool{
	settingsPkg: true,
	"github.com/getmiroir/trading-service/internal/settings/subsettings": true,
	"github.com/getmiroir/trading-service/internal/settings/secrets":     true,
}

// Generate writes Markdown describing settings.Settings and all nested
// structs from packages under internal/settings (including subsettings and secrets).
// moduleDir is the module root (copy-service); if empty, "." is used.
func Generate(w io.Writer, moduleDir string) error {
	if moduleDir == "" {
		moduleDir = "."
	}
	pkgs, err := loadPackages(moduleDir)
	if err != nil {
		return err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return fmt.Errorf("package load had errors")
	}

	fieldIndex, typeDocs, err := buildFieldIndex(pkgs)
	if err != nil {
		return err
	}

	settingsNamed, err := findSettingsNamed(pkgs)
	if err != nil {
		return err
	}

	structOrder := bfsStructOrder(settingsNamed)
	sectionSet := make(map[*types.Named]bool, len(structOrder))
	for _, n := range structOrder {
		sectionSet[n] = true
	}

	enumValues := collectEnumValues(pkgs)

	defaults := settings.DefaultSettings()
	defaultRV := defaultStructRVByTypeKey(settingsNamed, reflect.ValueOf(defaults))

	writeDoc(w, structOrder, sectionSet, fieldIndex, typeDocs, enumValues, defaultRV)
	return nil
}

func loadPackages(moduleDir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedImports | packages.NeedFiles,
		Dir:  moduleDir,
	}
	patterns := []string{
		settingsPkg,
		"github.com/getmiroir/trading-service/internal/settings/subsettings",
		"github.com/getmiroir/trading-service/internal/settings/secrets",
	}
	return packages.Load(cfg, patterns...)
}

func findSettingsNamed(pkgs []*packages.Package) (*types.Named, error) {
	for _, pkg := range pkgs {
		if pkg.PkgPath != settingsPkg {
			continue
		}
		obj := pkg.Types.Scope().Lookup("Settings")
		if obj == nil {
			continue
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); !ok {
			return nil, fmt.Errorf("Settings is not a struct")
		}
		return named, nil
	}
	return nil, fmt.Errorf("could not find settings.Settings")
}

type fieldMeta struct {
	goName   string
	yamlName string
	notes    string
}

func buildFieldIndex(pkgs []*packages.Package) (map[typeKey][]fieldMeta, map[typeKey]string, error) {
	index := make(map[typeKey][]fieldMeta)
	typeDocs := make(map[typeKey]string)
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				typeDeclDoc := ""
				if gd.Doc != nil {
					typeDeclDoc = strings.TrimSpace(gd.Doc.Text())
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					obj := pkg.TypesInfo.Defs[ts.Name]
					if obj == nil {
						continue
					}
					tn, ok := obj.(*types.TypeName)
					if !ok {
						continue
					}
					named, ok := tn.Type().(*types.Named)
					if !ok {
						continue
					}
					key := typeKey{named.Obj().Pkg().Path(), named.Obj().Name()}
					doc := typeDeclDoc
					if ts.Doc != nil {
						if doc != "" {
							doc += "\n\n"
						}
						doc += strings.TrimSpace(ts.Doc.Text())
					}

					st, isStruct := ts.Type.(*ast.StructType)
					if !isStruct {
						if isEnumNamedType(named) && doc != "" {
							typeDocs[key] = doc
						}
						continue
					}
					if _, ok := named.Underlying().(*types.Struct); !ok {
						continue
					}
					rows, err := astStructFields(st)
					if err != nil {
						return nil, nil, err
					}
					index[key] = rows
					if doc != "" {
						typeDocs[key] = doc
					}
				}
			}
		}
	}
	return index, typeDocs, nil
}

func astStructFields(st *ast.StructType) ([]fieldMeta, error) {
	var rows []fieldMeta
	for _, field := range st.Fields.List {
		tag := ""
		if field.Tag != nil {
			tag = strings.Trim(field.Tag.Value, "`")
		}
		yamlFromTag := yamlKeyFromStructTag(tag)
		doc := ""
		if field.Doc != nil {
			doc = strings.TrimSpace(field.Doc.Text())
		}
		if field.Comment != nil {
			for _, c := range field.Comment.List {
				t := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
				if doc == "" {
					doc = t
				} else {
					doc += " " + t
				}
			}
		}
		if len(field.Names) == 0 {
			// Embedded field: skip for current settings schema (none with yaml in tree).
			continue
		}
		for _, ident := range field.Names {
			goName := ident.Name
			yamlName := yamlFromTag
			if yamlName == "" {
				yamlName = goName
			}
			rows = append(rows, fieldMeta{
				goName:   goName,
				yamlName: yamlName,
				notes:    doc,
			})
		}
	}
	return rows, nil
}

func yamlKeyFromStructTag(tag string) string {
	y := reflect.StructTag(tag).Get("yaml")
	if y == "" {
		return ""
	}
	if i := strings.Index(y, ","); i >= 0 {
		y = y[:i]
	}
	return y
}

type typeKey struct {
	pkgPath  string
	typeName string
}

// defaultStructRVByTypeKey mirrors bfsStructOrder and records a reflect.Value of
// each documented struct type as it appears under DefaultSettings(), using the
// first map/slice element when needed or a zero value when empty.
func defaultStructRVByTypeKey(rootNamed *types.Named, rootRV reflect.Value) map[typeKey]reflect.Value {
	out := make(map[typeKey]reflect.Value)
	if !isDocumentedStruct(rootNamed) {
		return out
	}
	seen := make(map[*types.Named]bool)
	type pair struct {
		n  *types.Named
		rv reflect.Value
	}
	var queue []pair
	seen[rootNamed] = true
	queue = append(queue, pair{rootNamed, rootRV})

	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]
		key := typeKey{it.n.Obj().Pkg().Path(), it.n.Obj().Name()}
		out[key] = it.rv

		st := it.n.Underlying().(*types.Struct)
		for j := 0; j < st.NumFields(); j++ {
			if !st.Field(j).Exported() {
				continue
			}
			ft := st.Field(j).Type()
			inner := rootNamedStruct(ft)
			if inner == nil || seen[inner] || !isDocumentedStruct(inner) {
				continue
			}
			fv := it.rv.Field(j)
			child, ok := representativeChildRV(fv, inner)
			if !ok || !child.IsValid() {
				continue
			}
			seen[inner] = true
			queue = append(queue, pair{inner, child})
		}
	}
	return out
}

func representativeChildRV(fv reflect.Value, inner *types.Named) (reflect.Value, bool) {
	for fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			zt := fv.Type().Elem()
			if zt.Kind() == reflect.Struct && zt.Name() == inner.Obj().Name() && zt.PkgPath() == inner.Obj().Pkg().Path() {
				return reflect.Zero(zt), true
			}
			return reflect.Value{}, false
		}
		fv = fv.Elem()
	}

	wantName := inner.Obj().Name()
	wantPkg := inner.Obj().Pkg().Path()
	matches := func(v reflect.Value) bool {
		return v.Kind() == reflect.Struct && v.Type().Name() == wantName && v.Type().PkgPath() == wantPkg
	}

	if matches(fv) {
		return fv, true
	}
	if fv.Kind() == reflect.Map {
		et := fv.Type().Elem()
		if et.Kind() == reflect.Ptr {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct || et.Name() != wantName || et.PkgPath() != wantPkg {
			return reflect.Value{}, false
		}
		if fv.Len() > 0 {
			v := fv.MapIndex(fv.MapKeys()[0])
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return reflect.Zero(et), true
				}
				v = v.Elem()
			}
			return v, true
		}
		return reflect.Zero(et), true
	}
	if fv.Kind() == reflect.Slice {
		et := fv.Type().Elem()
		if et.Kind() == reflect.Ptr {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct || et.Name() != wantName || et.PkgPath() != wantPkg {
			return reflect.Value{}, false
		}
		if fv.Len() > 0 {
			v := fv.Index(0)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return reflect.Zero(et), true
				}
				v = v.Elem()
			}
			return v, true
		}
		return reflect.Zero(et), true
	}
	return reflect.Value{}, false
}

func bfsStructOrder(root *types.Named) []*types.Named {
	seen := make(map[*types.Named]bool)
	var queue []*types.Named
	if isDocumentedStruct(root) {
		seen[root] = true
		queue = append(queue, root)
	}
	var order []*types.Named
	for i := 0; i < len(queue); i++ {
		n := queue[i]
		order = append(order, n)
		st := n.Underlying().(*types.Struct)
		for j := 0; j < st.NumFields(); j++ {
			inner := rootNamedStruct(st.Field(j).Type())
			if inner == nil || seen[inner] || !isDocumentedStruct(inner) {
				continue
			}
			seen[inner] = true
			queue = append(queue, inner)
		}
	}
	return order
}

// collectEnumValues returns, for each named "enum" type (a documented named type
// whose underlying type is a basic type, e.g. `type BybitRegion string`), the
// list of allowed values declared as package-level constants.
func collectEnumValues(pkgs []*packages.Package) map[typeKey][]string {
	out := make(map[typeKey][]string)
	for _, pkg := range pkgs {
		if !documentedPkgPaths[pkg.PkgPath] || pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			c, ok := scope.Lookup(name).(*types.Const)
			if !ok {
				continue
			}
			named, ok := c.Type().(*types.Named)
			if !ok || !isEnumNamedType(named) {
				continue
			}
			key := typeKey{named.Obj().Pkg().Path(), named.Obj().Name()}
			out[key] = append(out[key], formatConstValue(c.Val()))
		}
	}
	return out
}

func formatConstValue(v constant.Value) string {
	if v.Kind() == constant.String {
		return strconv.Quote(constant.StringVal(v))
	}
	return v.ExactString()
}

func isEnumNamedType(n *types.Named) bool {
	if n == nil || !documentedPkgPaths[n.Obj().Pkg().Path()] {
		return false
	}
	_, ok := n.Underlying().(*types.Basic)
	return ok
}

func rootNamedEnum(t types.Type) *types.Named {
	n, ok := unwrapPtrSliceMap(t).(*types.Named)
	if !ok || !isEnumNamedType(n) {
		return nil
	}
	return n
}

func isDocumentedStruct(n *types.Named) bool {
	if n == nil {
		return false
	}
	if _, ok := n.Underlying().(*types.Struct); !ok {
		return false
	}
	return documentedPkgPaths[n.Obj().Pkg().Path()]
}

func rootNamedStruct(t types.Type) *types.Named {
	t = unwrapPtrSliceMap(t)
	n, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	if _, ok := n.Underlying().(*types.Struct); !ok {
		return nil
	}
	return n
}

// fieldTypeHasOwnDocSection reports whether t is (or holds) a named struct type
// that has its own ## subsection, so inline defaults should be omitted.
func fieldTypeHasOwnDocSection(t types.Type, sectionSet map[*types.Named]bool) bool {
	n := rootNamedStruct(t)
	return n != nil && sectionSet[n]
}

func unwrapPtrSliceMap(t types.Type) types.Type {
	for {
		switch x := t.(type) {
		case *types.Pointer:
			t = x.Elem()
		case *types.Slice:
			t = x.Elem()
		case *types.Map:
			t = x.Elem()
		default:
			return t
		}
	}
}

func sectionTitle(n *types.Named) string {
	return n.Obj().Name()
}

func writeDoc(w io.Writer, structOrder []*types.Named, sectionSet map[*types.Named]bool, fieldIndex map[typeKey][]fieldMeta, typeDocs map[typeKey]string, enumValues map[typeKey][]string, defaultRV map[typeKey]reflect.Value) {
	fmt.Fprintf(w, "# Full settings reference\n\n")
	fmt.Fprintf(w, "This section contains the full reference of allowed settings for the copy service. ")
	fmt.Fprintf(w, "Each subsection lists the schema of a settings object, given by its YAML key, expected type, and default value for primitive types. ")
	fmt.Fprintf(w, "Complex types always have the schema of the nested types below them.\n\n")

	writtenEnums := make(map[typeKey]bool)
	for _, named := range structOrder {
		// Emit a section for each enum type referenced by this struct, right
		// before the struct that first uses it.
		st := named.Underlying().(*types.Struct)
		for j := 0; j < st.NumFields(); j++ {
			enum := rootNamedEnum(st.Field(j).Type())
			if enum == nil {
				continue
			}
			key := typeKey{enum.Obj().Pkg().Path(), enum.Obj().Name()}
			if writtenEnums[key] || len(enumValues[key]) == 0 {
				continue
			}
			writtenEnums[key] = true
			writeEnumSection(w, enum, typeDocs, enumValues[key])
		}
		writeStructSection(w, named, sectionSet, fieldIndex, typeDocs, defaultRV)
	}
}

func writeTypeDoc(w io.Writer, typeDocs map[typeKey]string, named *types.Named) {
	tkey := typeKey{named.Obj().Pkg().Path(), named.Obj().Name()}
	for _, para := range strings.Split(typeDocs[tkey], "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		fmt.Fprintf(w, "%s\n\n", para)
	}
}

func writeEnumSection(w io.Writer, named *types.Named, typeDocs map[typeKey]string, values []string) {
	fmt.Fprintf(w, "## %s\n\n", sectionTitle(named))
	writeTypeDoc(w, typeDocs, named)
	fmt.Fprintf(w, "Allowed values:\n\n")
	for _, v := range values {
		fmt.Fprintf(w, "- `%s`\n", v)
	}
	fmt.Fprintln(w)
}

func writeStructSection(w io.Writer, named *types.Named, sectionSet map[*types.Named]bool, fieldIndex map[typeKey][]fieldMeta, typeDocs map[typeKey]string, defaultRV map[typeKey]reflect.Value) {
	fmt.Fprintf(w, "## %s\n\n", sectionTitle(named))
	writeTypeDoc(w, typeDocs, named)

	st := named.Underlying().(*types.Struct)
	key := typeKey{named.Obj().Pkg().Path(), named.Obj().Name()}
	astRows := fieldIndex[key]
	byGoName := make(map[string]fieldMeta, len(astRows))
	for _, r := range astRows {
		byGoName[r.goName] = r
	}

	structRV, haveStructRV := defaultRV[key]

	for i := 0; i < st.NumFields(); i++ {
		tf := st.Field(i)
		if !tf.Exported() {
			continue
		}
		goName := tf.Name()
		var meta fieldMeta
		if i < len(astRows) && astRows[i].goName == goName {
			meta = astRows[i]
		} else if m, ok := byGoName[goName]; ok {
			meta = m
		} else {
			meta = fieldMeta{goName: goName, yamlName: goName}
			fmt.Fprintf(os.Stderr, "settingsdoc: no AST metadata for %s.%s\n", named.Obj().Name(), goName)
		}
		typeStr := formatTypePlain(tf.Type())
		nested := fieldTypeHasOwnDocSection(tf.Type(), sectionSet)
		var fv reflect.Value
		if !nested && haveStructRV && structRV.IsValid() && i < structRV.NumField() {
			fv = structRV.Field(i)
		}
		if !nested && fv.IsValid() && fv.CanInterface() {
			fmt.Fprintf(w, "- **`%s`** (_`%s`_): `%s`\n", meta.yamlName, typeStr, formatDefault(fv))
		} else {
			fmt.Fprintf(w, "- **`%s`** (_`%s`_)\n", meta.yamlName, typeStr)
		}
		notes := strings.TrimSpace(meta.notes)
		if notes != "" {
			for _, para := range strings.Split(notes, "\n\n") {
				para = strings.TrimSpace(para)
				if para == "" {
					continue
				}
				para = strings.ReplaceAll(para, "\n", " ")
				fmt.Fprintf(w, "    - %s\n", para)
			}
		}
	}
	fmt.Fprintln(w)
}

// settingsQualifier omits the package prefix for types declared in the
// documented settings packages (so `BybitRegion` is printed instead of its full
// import path), while keeping the prefix for external types like `time.Duration`.
func settingsQualifier(p *types.Package) string {
	if documentedPkgPaths[p.Path()] {
		return ""
	}
	return p.Name()
}

func formatTypePlain(t types.Type) string {
	return types.TypeString(t, settingsQualifier)
}

const (
	defaultValueMaxChars   = 280
	defaultMaxMapEntries   = 4
	defaultMaxSliceEntries = 3
)

func formatDefault(v reflect.Value) string {
	if !v.IsValid() {
		return "?"
	}
	s := formatDefaultInner(v, 2, defaultValueMaxChars)
	return strings.ReplaceAll(s, "`", "'")
}

func formatDefaultInner(v reflect.Value, depth, budget int) string {
	if budget <= 0 {
		return "…"
	}
	if !v.IsValid() {
		return "?"
	}

	switch v.Kind() {
	case reflect.Invalid:
		return "?"
	case reflect.Ptr:
		if v.IsNil() {
			return "nil"
		}
		return formatDefaultInner(v.Elem(), depth, budget)
	case reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		return formatDefaultInner(v.Elem(), depth, budget)
	case reflect.String:
		s := v.String()
		if len(s) > 80 {
			s = s[:77] + "…"
		}
		return strconv.Quote(s)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Duration]() {
			return v.Interface().(time.Duration).String()
		}
		if depth <= 0 {
			return "{…}"
		}
		return formatStructShallow(v, depth-1, budget)
	case reflect.Map:
		return formatMapDefault(v, budget)
	case reflect.Slice, reflect.Array:
		return formatSliceDefault(v, budget)
	default:
		if v.CanInterface() {
			t := fmt.Sprintf("%v", v.Interface())
			if len(t) > 120 {
				return t[:117] + "…"
			}
			return t
		}
		return "?"
	}
}

func formatStructShallow(v reflect.Value, depth, budget int) string {
	t := v.Type()
	var b strings.Builder
	b.WriteString("{ ")
	first := true
	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := v.Field(i)
		part := sf.Name + ": " + formatDefaultInner(fv, depth, min(budget-b.Len()-4, 80))
		if !first {
			b.WriteString(", ")
		}
		first = false
		if b.Len()+len(part) > budget {
			b.WriteString("…")
			break
		}
		b.WriteString(part)
	}
	b.WriteString(" }")
	return b.String()
}

func formatMapDefault(v reflect.Value, budget int) string {
	if v.Len() == 0 {
		return "{}"
	}
	keys := v.MapKeys()
	var b strings.Builder
	b.WriteString("{ ")
	n := min(len(keys), defaultMaxMapEntries)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		ks := formatDefaultInner(keys[i], 0, 40)
		vs := formatDefaultInner(v.MapIndex(keys[i]), 1, min(budget-b.Len()-4, 120))
		if b.Len()+len(ks)+len(vs)+4 > budget {
			b.WriteString("…")
			break
		}
		b.WriteString(ks)
		b.WriteString(": ")
		b.WriteString(vs)
	}
	if len(keys) > defaultMaxMapEntries {
		if b.Len()+4 <= budget {
			b.WriteString(", …")
		}
	}
	b.WriteString(" }")
	return b.String()
}

func formatSliceDefault(v reflect.Value, budget int) string {
	n := v.Len()
	if n == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteString("[ ")
	show := min(n, defaultMaxSliceEntries)
	for i := 0; i < show; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		s := formatDefaultInner(v.Index(i), 1, min(budget-b.Len(), 100))
		b.WriteString(s)
		if b.Len() > budget-4 {
			b.WriteString("…")
			break
		}
	}
	if n > show {
		b.WriteString(", …")
	}
	b.WriteString(" ]")
	return b.String()
}
