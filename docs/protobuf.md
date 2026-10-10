# Protobuf messages

`protoc-gen-goalchemy` is a `protoc` and `buf` plugin that turns `.proto`
files into Goalchemy-subset Go: one struct per message and proto3 JSON
encoding through `std/encoding/protojson`. The generated code is ordinary
source, so it compiles to every target with the rest of a program.

```bash
go install github.com/eugenioenko/goalchemy/cmd/protoc-gen-goalchemy@latest
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - local: protoc-gen-goalchemy
    out: gen
    opt:
      - go_import_prefix=example.com/app/gen
      - include=policy.attributes.ListAttributesResponse;kas.RewrapRequest
```

## Parameters

| Parameter | Effect |
| --- | --- |
| `go_import_prefix=<path>` | Places each file's package at `<path>/<proto directory>` instead of its `go_package`. |
| `include=<full.Name>[;<full.Name>...]` | Generates only these messages and enums and the types they reference, across files. Without it every message in the requested files is generated. |
| `getters=false` | Omits the nil-safe `GetX` accessors. |

Generate only what a program uses: every message adds code, and Swift and
Rust build time grows with the amount of code.

## Generated code

Names follow `protoc-gen-go`: `KeyAccess`, `AttributeRuleTypeEnum_ATTRIBUTE_RULE_TYPE_ENUM_HIERARCHY`,
`EntityIdentifier_Token` for a oneof member, and an `isEntityIdentifier_Identifier`
interface for the oneof field.

| Proto | Go |
| --- | --- |
| scalar | `bool`, `int32`, `int64`, `uint32`, `uint64`, `float32`, `float64`, `string`, `[]byte` |
| `optional` scalar | pointer to the scalar |
| enum | named `int32` with `_name` and `_value` maps and `String` |
| message | pointer to the generated struct |
| `repeated T`, `map<K, V>` | `[]T`, `map[K]V` |
| `oneof` | interface field set to a wrapper struct per member |
| `google.protobuf.Timestamp`, `Duration`, `Empty`, `FieldMask`, `Any` | `*protojson.Timestamp` and so on |
| `google.protobuf.Struct`, `Value`, `ListValue` | `jsonvalue.Value` (zero means unset) |
| wrappers such as `BoolValue` | pointer to the scalar (`*bool`); `BytesValue` is `[]byte` |

Every message has `EncodeProtoJSON` and `DecodeProtoJSON` (the
`protojson.Message` interface), `MarshalJSON` and `UnmarshalJSON`, and,
unless disabled, `GetX` accessors. Each generated file registers its
messages with `protojson.Register` so that `Any` payloads of those types are
decoded and re-encoded canonically.

```go
attr := &policy.Attribute{Name: "classification", Rule: policy.AttributeRuleTypeEnum_ATTRIBUTE_RULE_TYPE_ENUM_HIERARCHY}
data, err := protojson.Marshal(attr)

var resp attributes.ListAttributesResponse
err = protojson.Unmarshal(body, &resp)
for _, a := range resp.GetAttributes() {
	println(a.GetFqn())
}
```

## Differences from `google.golang.org/protobuf/encoding/protojson`

Output is `protojson.Marshal`'s with insignificant whitespace removed, and
input follows `protojson.UnmarshalOptions{DiscardUnknown: true}`, the options
Connect clients use. The `proto_platform` language fixture checks both
against protojson on every target, using the OpenTDF platform's messages.

- **Unregistered `Any` payloads** are kept as received instead of failing
  with "unable to resolve". Payloads of generated messages and of the
  well-known types are decoded and re-encoded as protojson does.
- **Error messages** use protojson's wording where practical but do not
  include byte offsets.
- Extensions, groups and proto2 defaults are not supported.
