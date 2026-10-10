package main

import (
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	_ "github.com/opentdf/platform/protocol/go/authorization"
	_ "github.com/opentdf/platform/protocol/go/authorization/v2"
	_ "github.com/opentdf/platform/protocol/go/kas"
	_ "github.com/opentdf/platform/protocol/go/policy/attributes"
)

func main() {
	roots := []string{"authorization/authorization.proto", "authorization/v2/authorization.proto", "kas/kas.proto", "policy/attributes/attributes.proto"}
	seen := map[string]bool{}
	set := &descriptorpb.FileDescriptorSet{}
	var add func(fd protoreflect.FileDescriptor)
	add = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		imps := fd.Imports()
		for i := 0; i < imps.Len(); i++ {
			add(imps.Get(i).FileDescriptor)
		}
		set.File = append(set.File, protodesc.ToFileDescriptorProto(fd))
	}
	for _, r := range roots {
		fd, err := protoregistry.GlobalFiles.FindFileByPath(r)
		if err != nil {
			panic(err)
		}
		add(fd)
	}
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(b)
}
