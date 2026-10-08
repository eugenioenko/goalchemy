package swift

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/ir"
)

type LibraryBoundaryError struct{ Message string }

func (e *LibraryBoundaryError) Error() string { return e.Message }
func boundary(format string, args ...any) error {
	return &LibraryBoundaryError{fmt.Sprintf(format, args...)}
}

func sourceContext(t *ir.Type) bool {
	return t.U().Kind == ir.KOpaque && t.U().Name == "context.Context"
}
func sourceError(t *ir.Type) bool { return t.Name == "error" }
func cryptoKey(t *ir.Type) bool {
	return t.U().Kind == ir.KPointer && t.U().Elem.U().Kind == ir.KOpaque && t.U().Elem.U().Name == "crypto.Key"
}
func libraryValue(t *ir.Type, seen map[*ir.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	u := t.U()
	switch u.Kind {
	case ir.KBool, ir.KInt, ir.KFloat, ir.KString:
		return true
	case ir.KArray, ir.KSlice:
		return libraryValue(u.Elem, seen)
	case ir.KStruct:
		for _, f := range u.Fields {
			if !token.IsExported(f.Name) || f.Embedded || !libraryValue(f.Type, seen) {
				return false
			}
		}
		return true
	}
	return cryptoKey(t)
}
func swiftPublic(name string) string { return "`" + name + "`" }
func (e *emitter) library() (string, error) {
	var b strings.Builder
	names := map[*ir.Type]string{}
	used := map[string]bool{}
	reserved := map[string]bool{}
	for _, n := range strings.Fields("GoString GoalchemyFailure GoalchemyKey CallOptions CancellationToken CallbackProvider CallbackRequest Operation GValue GTypes GFrame GOwner GNative GCell Array String Bool Int Int8 Int16 Int32 Int64 UInt UInt8 UInt16 UInt32 UInt64 Float Double Error Result Data Date UUID Task Thread DispatchQueue NSLock NSRecursiveLock NSCondition Set Dictionary Optional Never Void Foundation Swift Self Any AnyObject Locale ProcessInfo FileHandle ObjectIdentifier DispatchTime DispatchTimeInterval TimeInterval Unicode UTF8 UnsafeBufferPointer UnsafeMutableBufferPointer UnsafePointer UnsafeMutablePointer UnsafeRawPointer UnsafeMutableRawPointer OpaquePointer CallOptions CallbackProvider CallbackRequest CancellationToken GAction GAggregate GBounds GBuffer GCallback GCell GChannel GChannelWait GContext GCrypto GCryptoInput GCryptoKey GCryptoOutput GError GFatal GFault GFrame GFunction GHTTP GHTTPInput GHTTPOutput GHeap GHostOperation GInterface GLibrary GLibraryRetention GMailbox GManaged GMap GMapEntry GMapIterator GMutex GNative GNumeric GOpaque GOwner GPanic GPointer GProviderLease GReject GSelectCase GSelection GSlice GString GTask GTimer GType GTypes GValue GWaitGroup GWeak GWeakCryptoKey GoString GoalchemyFailure GoalchemyKey Operation") {
		reserved[n] = true
	}
	roots := map[string]bool{}
	for _, f := range e.p.Exports {
		roots[f.Pkg] = true
	}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KNamed && roots[t.Pkg] && token.IsExported(t.Obj) && t.U().Kind == ir.KStruct && libraryValue(t, map[*ir.Type]bool{}) {
			if used[t.Obj] || reserved[t.Obj] {
				return "", boundary("Swift public name collision: %s", t.Obj)
			}
			used[t.Obj] = true
			names[t.U()] = t.Obj
		}
	}
	var typ func(*ir.Type) string
	typ = func(t *ir.Type) string {
		u := t.U()
		switch u.Kind {
		case ir.KBool:
			return "Bool"
		case ir.KInt:
			prefix := "Int"
			if !u.Int.Signed() {
				prefix = "UInt"
			}
			return fmt.Sprintf("%s%d", prefix, u.Int.Bits())
		case ir.KFloat:
			if u.FloatBits == 32 {
				return "Float"
			}
			return "Double"
		case ir.KString:
			return "GoString"
		case ir.KArray:
			return "[" + typ(u.Elem) + "]"
		case ir.KSlice:
			return "[" + typ(u.Elem) + "]?"
		case ir.KStruct:
			if n := names[u]; n != "" {
				return swiftPublic(n)
			}
			n := fmt.Sprintf("Value%d", u.ID)
			for used[n] || reserved[n] {
				n += "_"
			}
			used[n] = true
			names[u] = n
			return swiftPublic(n)
		}
		if cryptoKey(t) {
			return "GoalchemyKey?"
		}
		panic("swift public type " + t.String())
	}
	var defaultValue func(*ir.Type) string
	defaultValue = func(t *ir.Type) string {
		u := t.U()
		switch u.Kind {
		case ir.KBool:
			return "false"
		case ir.KInt, ir.KFloat:
			return "0"
		case ir.KString:
			return "GoString(\"\")"
		case ir.KArray:
			return fmt.Sprintf("Array(repeating: %s, count: %d)", defaultValue(u.Elem), u.Len)
		case ir.KSlice:
			return "nil"
		case ir.KStruct:
			return typ(t) + "()"
		}
		return "nil"
	}
	seen := map[*ir.Type]bool{}
	var convert func(*ir.Type)
	convert = func(t *ir.Type) {
		u := t.U()
		if seen[u] {
			return
		}
		seen[u] = true
		if u.Kind == ir.KStruct {
			n := typ(t)
			for _, f := range u.Fields {
				convert(f.Type)
			}
			fmt.Fprintf(&b, "public struct %s {\n", n)
			var init []string
			for _, f := range u.Fields {
				fmt.Fprintf(&b, "    public var %s: %s\n", swiftPublic(f.Name), typ(f.Type))
				init = append(init, fmt.Sprintf("%s: %s = %s", swiftPublic(f.Name), typ(f.Type), defaultValue(f.Type)))
			}
			fmt.Fprintf(&b, "    public init(%s) {\n", strings.Join(init, ", "))
			for _, f := range u.Fields {
				fmt.Fprintf(&b, "        self.%s = %s\n", swiftPublic(f.Name), swiftPublic(f.Name))
			}
			b.WriteString("    }\n}\n")
		} else if u.Kind == ir.KArray || u.Kind == ir.KSlice {
			convert(u.Elem)
		}
		fmt.Fprintf(&b, "private func gInput%d(_ v: %s) throws -> GValue {\n", u.ID, typ(t))
		switch u.Kind {
		case ir.KBool:
			b.WriteString("    return .bool(v)\n")
		case ir.KInt:
			fmt.Fprintf(&b, "    return GNumeric.normalize(UInt64(truncatingIfNeeded: v), %d, %t)\n", u.Int.Bits(), u.Int.Signed())
		case ir.KFloat:
			fmt.Fprintf(&b, "    return .float(Double(v), %d)\n", u.FloatBits)
		case ir.KString:
			b.WriteString("    return .string(v.bytes)\n")
		case ir.KStruct:
			fmt.Fprintf(&b, "    let result = GAggregate(%d)\n", u.ID)
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "    result.fields[%d].value = try gInput%d(v.%s)\n", i, f.Type.U().ID, swiftPublic(f.Name))
			}
			b.WriteString("    return .aggregate(result)\n")
		case ir.KSlice, ir.KArray:
			if u.Kind == ir.KSlice {
				fmt.Fprintf(&b, "    guard let v = v else { return .slice(GSlice(nil, 0, 0, 0, %d)) }\n", u.Elem.ID)
			} else {
				fmt.Fprintf(&b, "    if v.count != %d { throw GoalchemyFailure(\"invalid_argument\", \"array length must be %d\") }\n", u.Len, u.Len)
			}
			if elem := u.Elem.U(); elem.Kind == ir.KInt && elem.Int.Bits() == 8 && !elem.Int.Signed() {
				fmt.Fprintf(&b, "    let buffer = GBuffer(bytes: v, elem: %d)\n", u.Elem.ID)
			} else {
				fmt.Fprintf(&b, "    let buffer = GBuffer(v.count, %d)\n    for i in v.indices { buffer.write(i, try gInput%d(v[i])) }\n", u.Elem.ID, u.Elem.U().ID)
			}
			if u.Kind == ir.KSlice {
				fmt.Fprintf(&b, "    return .slice(GSlice(buffer, 0, v.count, v.count, %d))\n", u.Elem.ID)
			} else {
				fmt.Fprintf(&b, "    let result = GAggregate(%d); result.buffer = buffer; return .aggregate(result)\n", u.ID)
			}
		default:
			if cryptoKey(t) {
				b.WriteString("    guard let v = v else { return .nilValue }; return .pointer(v.cell)\n")
			}
		}
		b.WriteString("}\n")
		fmt.Fprintf(&b, "private func gOutput%d(_ v: GValue) throws -> %s {\n", u.ID, typ(t))
		switch u.Kind {
		case ir.KBool:
			b.WriteString("    return v.boolValue\n")
		case ir.KInt:
			fmt.Fprintf(&b, "    return %s(truncatingIfNeeded: v.unsigned)\n", typ(t))
		case ir.KFloat:
			fmt.Fprintf(&b, "    guard case .float(let number, _) = v else { throw GFault(\"float boundary\") }; return %s(number)\n", typ(t))
		case ir.KString:
			b.WriteString("    return GoString(bytes: v.bytes)\n")
		case ir.KStruct:
			fmt.Fprintf(&b, "    guard case .aggregate(let a) = v else { throw GFault(\"struct boundary\") }; return %s(\n", typ(t))
			var fs []string
			for i, f := range u.Fields {
				fs = append(fs, fmt.Sprintf("        %s: try gOutput%d(a.fields[%d].value)", swiftPublic(f.Name), f.Type.U().ID, i))
			}
			b.WriteString(strings.Join(fs, ",\n") + "\n    )\n")
		case ir.KSlice, ir.KArray:
			if u.Kind == ir.KSlice {
				b.WriteString("    if case .slice(let s) = v, s.storage == nil { return nil }\n")
			}
			if elem := u.Elem.U(); elem.Kind == ir.KInt && elem.Int.Bits() == 8 && !elem.Int.Signed() {
				b.WriteString("    guard let bytes = GSlice.byteSource(v) else { throw GFault(\"byte boundary\") }; return Array(bytes)\n")
			} else {
				fmt.Fprintf(&b, "    return try GElements(v).map { try gOutput%d($0) }\n", u.Elem.U().ID)
			}
		default:
			if cryptoKey(t) {
				b.WriteString("    if case .nilValue = v { return nil }; guard case .pointer(let p) = v, case .opaque(let k) = p.value, let key = k as? GCryptoKey else { throw GFault(\"key boundary\") }; return GoalchemyKey(key, p)\n")
			}
		}
		b.WriteString("}\n")
	}
	for _, f := range e.p.Exports {
		if f.Sig.Variadic {
			return "", boundary("variadic Swift export %s", f.Name)
		}
		for _, t := range f.Sig.Params {
			if sourceContext(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", boundary("unsupported Swift public parameter %s", t.String())
			}
			convert(t)
		}
		for i, t := range f.Sig.Results {
			if i == len(f.Sig.Results)-1 && sourceError(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", boundary("unsupported Swift public result %s", t.String())
			}
			convert(t)
		}
	}
	var errorTypes []*ir.Type
	for _, t := range e.p.Types.All {
		if t.U().Kind == ir.KPointer && t.U().Elem.U().Kind == ir.KStruct && t.Boxed && libraryValue(t.U().Elem, map[*ir.Type]bool{}) {
			convert(t.U().Elem)
			errorTypes = append(errorTypes, t)
		}
	}
	b.WriteString("private func gSourceError(_ value: GValue) throws -> GoalchemyFailure? {\n    if case .nilValue = value { return nil }\n    guard case .interface(let box) = value else { return GoalchemyFailure(\"source\", \"source error\") }\n")
	for _, t := range errorTypes {
		fmt.Fprintf(&b, "    if box.type == %d, case .pointer(let cell) = box.value {\n        let v = try gOutput%d(cell.value)\n        return GoalchemyFailure(\"source\", \"source error\", [\n", t.ID, t.U().Elem.U().ID)
		var fs []string
		for _, f := range t.U().Elem.U().Fields {
			fs = append(fs, fmt.Sprintf("            %s: v.%s as Any", quote(f.Name), swiftPublic(f.Name)))
		}
		b.WriteString(strings.Join(fs, ",\n") + "\n        ])\n    }\n")
	}
	b.WriteString("    return GoalchemyFailure(\"source\", String(decoding: GFormat(value), as: UTF8.self))\n}\nprivate func gLibraryReset() {\n    _ = gTypeRegistration\n    gSetupMethods()\n")
	for _, g := range e.p.Globals {
		fmt.Fprintf(&b, "    %s.value = GTypes.zero(%d)\n", e.names.Symbol(g.Sym), g.Type.ID)
	}
	b.WriteString("}\n")
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		if used[name] || reserved[name] {
			return "", boundary("Swift public export collision %s", name)
		}
		used[name] = true
		n := len(f.Sig.Results)
		hasErr := n > 0 && sourceError(f.Sig.Results[n-1])
		if hasErr {
			n--
		}
		result := "Void"
		if n == 1 {
			result = typ(f.Sig.Results[0])
		} else if n > 1 {
			var ts []string
			for _, t := range f.Sig.Results[:n] {
				ts = append(ts, typ(t))
			}
			result = "(" + strings.Join(ts, ", ") + ")"
		}
		var ps, inputs, args []string
		inputIndex := 0
		for i, t := range f.Sig.Params {
			if sourceContext(t) {
				args = append(args, "context")
				continue
			}
			ps = append(ps, fmt.Sprintf("_ a%d: %s", i, typ(t)))
			inputs = append(inputs, fmt.Sprintf("try gInput%d(a%d)", t.U().ID, i))
			args = append(args, fmt.Sprintf("args[%d]", inputIndex))
			inputIndex++
		}
		ps = append(ps, "_ options: CallOptions = CallOptions()")
		fmt.Fprintf(&b, "public func %s(%s) -> Operation<%s> {\n    do {\n        let args = try GLibrary.prepare {\n            _ = gTypeRegistration\n            return [%s]\n        }\n", swiftPublic(name), strings.Join(ps, ", "), result, strings.Join(inputs, ", "))
		fmt.Fprintf(&b, "        return GLibrary.submit(args, options, gLibraryReset, { %s([], []) }, { args, context in %s([%s], []) }, { rv in\n", e.names.Symbol(e.p.Init.Sym), e.names.Symbol(f.Sym), strings.Join(args, ", "))
		if hasErr {
			fmt.Fprintf(&b, "            if let error = try gSourceError(rv[%d]) { throw error }\n", n)
		}
		if n == 0 {
			b.WriteString("            return ()\n")
		} else {
			var xs []string
			for i, t := range f.Sig.Results[:n] {
				xs = append(xs, fmt.Sprintf("try gOutput%d(rv[%d])", t.U().ID, i))
			}
			ret := strings.Join(xs, ", ")
			if n > 1 {
				ret = "(" + ret + ")"
			}
			fmt.Fprintf(&b, "            return %s\n", ret)
		}
		b.WriteString("        })\n    } catch { return Operation(options.cancellation) { throw error } }\n}\n")
	}
	return b.String(), nil
}
