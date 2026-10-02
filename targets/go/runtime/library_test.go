package rt

import (
	"context"
	"errors"
	"testing"
)

func TestLibraryRetiresAcquiredKeysOnEveryExit(t *testing.T) {
	for _, mode := range []string{"success", "source", "fault", "constructor"} {
		t.Run(mode, func(t *testing.T) {
			var key *Key
			_, err := RunLibrary(context.Background(), nil, func(c Context) Frame {
				if mode == "constructor" {
					sched.retire = append(sched.retire, func() { key = nil })
					panic(HostFault{Value: "constructor"})
				}
				return Sync(func() []any {
					task := sched.cur
					LibCryptoGenerateP256(task)
					key = task.RV[0].(*Key)
					switch mode {
					case "source":
						panic("source")
					case "fault":
						panic(HostFault{Value: "adapter"})
					}
					return nil
				})
			}, nil, nil)
			if mode == "success" && err != nil {
				t.Fatal(err)
			}
			if mode != "success" && err == nil {
				t.Fatal("fault accepted")
			}
			if key != nil {
				if _, e := key.PublicPEM(); e == nil {
					t.Fatal("native key survived retirement")
				}
			}
			if sched.library || sched.main != nil || len(sched.operations) != 0 {
				t.Fatal("retired owner retained")
			}
		})
	}
}
func TestLibraryInitializationPanicAndReservationRelease(t *testing.T) {
	_, err := RunLibrary(context.Background(), nil, func(Context) Frame {
		return LibrarySequence(Sync(func() []any { panic("init") }), func() Frame { t.Fatal("export after failed init"); return nil })
	}, nil, nil)
	var e *LibraryError
	if !errors.As(err, &e) || e.Kind != "source_panic" {
		t.Fatal(err)
	}
	_, err = RunLibrary(context.Background(), nil, func(Context) Frame { return Sync(func() []any { return []any{7} }) }, nil, nil)
	if err != nil {
		t.Fatal("reservation", err)
	}
}

type brokenLibraryContext struct{ context.Context }

func (brokenLibraryContext) Done() <-chan struct{} { panic("native context failure") }
func TestLibraryEarlyNativeBoundaryFault(t *testing.T) {
	_, err := RunLibrary(brokenLibraryContext{context.Background()}, nil, func(Context) Frame { t.Fatal("constructed after boundary fault"); return nil }, nil, nil)
	var e *LibraryError
	if !errors.As(err, &e) || e.Kind != "host_fault" {
		t.Fatal(err)
	}
	_, err = RunLibrary(context.Background(), nil, func(Context) Frame { return Sync(func() []any { return nil }) }, nil, nil)
	if err != nil {
		t.Fatal("reservation after early fault", err)
	}
}
