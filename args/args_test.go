package args_test

import (
	"testing"
	"time"

	declargs "github.com/tomerfiliba/decl/args"

	"github.com/stretchr/testify/assert"
)

type Spec struct {
	Verbose   bool          `arg:"v,verbose"`
	Quiet     bool          `arg:"quiet=false"`
	Level     int           `arg:"l=5"`
	Required  int           `arg:"r"`
	Interval  time.Duration `arg:"d=5s"`
	Timestamp time.Time     `arg:"t=Mon Jan 2 03:04:05 UTC 2000"`
	Filename  string        `arg:"*"`
}

func TestGoodPath(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "--quiet", "-r7", "/tmp/bar"})
	assert.NoError(t, err)

	ts, err := time.Parse(time.RFC3339, "2000-01-02T03:04:05Z")
	assert.NoError(t, err)

	assert.True(t, spec.Verbose)
	assert.Equal(t, 5, spec.Level)
	assert.Equal(t, 7, spec.Required)
	assert.Equal(t, ts, spec.Timestamp)
	assert.Equal(t, 5*time.Second, spec.Interval)
	assert.True(t, spec.Quiet)
	assert.Equal(t, spec.Filename, "/tmp/bar")
}

func TestUnexpectedVal(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v6", "/tmp/bar"})
	assert.ErrorContains(t, err, "does not expect a value")
}

func TestMissingVal(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-r"})
	assert.ErrorContains(t, err, "expects a value")
}

func TestMissingReqArg(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "/tmp/bar"})
	assert.ErrorContains(t, err, "Missing required arguments")
}

func TestUnknownLongArg(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "--spam", "/tmp/bar"})
	assert.ErrorContains(t, err, "Unknown")
}

func TestUnknownShortArg(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-s", "/tmp/bar"})
	assert.ErrorContains(t, err, "Unknown")
}

func TestMissingPosArg(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "-r7"})
	assert.ErrorContains(t, err, "Missing")
	assert.ErrorContains(t, err, "Filename")
}

func TestExtraPosArgs(t *testing.T) {
	spec := Spec{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "-r7", "/tmp/bar", "/tmp/extra"})
	assert.ErrorContains(t, err, "Unconsumed")
	assert.ErrorContains(t, err, "/tmp/extra")
}

func TestCaptureAll(t *testing.T) {
	spec := struct {
		Verbose bool     `arg:"v,verbose"`
		Pattern string   `arg:"*"`
		Files   []string `arg:"*"`
	}{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "kaki", "/tmp/bar", "/tmp/spam"})
	assert.NoError(t, err)

	assert.True(t, spec.Verbose)
	assert.Equal(t, spec.Pattern, "kaki")
	assert.Equal(t, spec.Files, []string{"/tmp/bar", "/tmp/spam"})
}

func TestAfterCaptureAll(t *testing.T) {
	spec := struct {
		Verbose bool     `arg:"v,verbose"`
		Files   []string `arg:"*"`
		Pattern string   `arg:"*"`
	}{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "kaki", "/tmp/bar", "/tmp/spam"})
	assert.ErrorContains(t, err, "capture-all")
}

func TestTwoCaptureAll(t *testing.T) {
	spec := struct {
		Verbose bool     `arg:"v,verbose"`
		Pattern []string `arg:"*"`
		Files   []string `arg:"*"`
	}{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "-v", "kaki", "/tmp/bar", "/tmp/spam"})
	assert.ErrorContains(t, err, "capture-all")
}

func TestAllTypes(t *testing.T) {
	spec := struct {
		F   bool    `arg:"flag"`
		I64 int     `arg:"i,int64"`
		U64 uint    `arg:"u,uint64"`
		F64 float64 `arg:"f"`
		S   string  `arg:"s"`
	}{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "--flag", "--int64=3", "--uint64", "7", "-f2.5", "-s", "hello"})
	assert.NoError(t, err)
	assert.True(t, spec.F)
	assert.Equal(t, spec.I64, 3)
	assert.Equal(t, spec.U64, uint(7))
	assert.Equal(t, spec.F64, 2.5)
	assert.Equal(t, spec.S, "hello")
}

func TestEndOfSwitches(t *testing.T) {
	spec := struct {
		F        bool     `arg:"flag"`
		CatchAll []string `arg:"*"`
	}{}
	err := declargs.LoadArgsSpecFrom(&spec, []string{"foo", "--", "--flag", "spam"})
	assert.NoError(t, err)
	assert.False(t, spec.F)
	assert.Equal(t, spec.CatchAll, []string{"--flag", "spam"})
}
