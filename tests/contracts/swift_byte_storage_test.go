package contracts

import (
	"testing"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// These programs observe value copies, capacity growth, overlapping aliases,
// and arbitrary byte values through source code, against the native Go oracle.
func TestSwiftByteValuesNativeOracle(t *testing.T) {
	for _, fixture := range []testutil.Fixture{
		{Name: "growth", Dir: "testdata/byte_growth"},
		{Name: "values", Dir: "testdata/java_byte_values", Gate: "cooperative"},
		{Name: "storage", Dir: "../language/testdata/byte_storage"},
	} {
		t.Run(fixture.Name, func(t *testing.T) {
			testutil.RunFixture(t, fixture, []string{"go", "swift"})
		})
	}
}
