`platform.binpb` is a serialized `FileDescriptorSet` holding the OpenTDF
platform's authorization (v1 and v2), KAS and policy attribute services and
every file they import, taken from `github.com/opentdf/platform/protocol/go`
v0.41.0. Regenerate it with:

    (cd internal/protoc/testdata/descriptors && go run . > ../platform.binpb)

The `descriptors` directory is its own module so the platform dependency stays
out of Goalchemy's `go.mod`.
