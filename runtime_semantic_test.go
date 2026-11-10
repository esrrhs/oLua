package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Print effects are observed in order, including before an error. Fixtures return
// any additional side effects. Numbers include their Lua integer/float subtype.
const semanticDriver = `
assert(math.maxinteger == 9223372036854775807, "64-bit Lua integers required")
local source = io.read("*a")
local function value(v)
 local t = type(v)
 if t == "number" then return (math.type and math.type(v) or t)..":"..tostring(v) end
 if t == "string" then return "string:"..string.format("%q", v) end
 if t == "nil" or t == "boolean" then return t..":"..tostring(v) end
 error("fixture must serialize observable "..t.." explicitly")
end
local f, err = load(source, "@case")
if not f then error(err) end
local r = table.pack(pcall(f))
io.write("\n@@RESULT ", tostring(r[1]), " ", r.n - 1, "\n")
for i = 2, r.n do
 if not r[1] and type(r[i]) == "string" then
  r[i] = r[i]:gsub("case:%d+: ", "")
 end
 io.write(value(r[i]), "\n")
end
`

func luaRuntimes(t *testing.T) []string {
	t.Helper()
	names := []string{"lua5.4", "lua5.3", "lua"}
	explicit := os.Getenv("OLUA_LUA") != ""
	if requested := os.Getenv("OLUA_LUA"); requested != "" {
		names = strings.Split(requested, string(os.PathListSeparator))
	}
	var paths []string
	seen := map[string]bool{}
	for _, name := range names {
		p, err := exec.LookPath(name)
		if err != nil {
			if explicit {
				t.Fatalf("required Lua runtime %q missing: %v", name, err)
			}
			continue
		}
		real, _ := filepath.EvalSymlinks(p)
		if seen[real] {
			continue
		}
		version, err := exec.Command(p, "-e", `io.write(_VERSION)`).CombinedOutput()
		if err != nil || (string(version) != "Lua 5.3" && string(version) != "Lua 5.4") {
			if explicit {
				t.Fatalf("required Lua runtime %q must be Lua 5.3/5.4: %s %v", name, version, err)
			}
			continue
		}
		seen[real] = true
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		t.Fatal("Lua 5.3/5.4 required for semantic tests; install lua5.4 or set OLUA_LUA (no silent skip)")
	}
	return paths
}

func luaObserve(t *testing.T, runtime, source string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, runtime, "-e", semanticDriver)
	cmd.Stdin = strings.NewReader(source)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Lua harness failed: %v\n%s\nsource:\n%s", err, out, source)
	}
	return string(out)
}

func optimizeSource(t *testing.T, source string, mask int, staticEnvironment ...bool) string {
	t.Helper()
	snap := snapshotPassFlags()
	defer restorePassFlags(snap)
	disableAllPasses()
	*unsafe_assume_static_globals = len(staticEnvironment) > 0 && staticEnvironment[0]
	*opt_const_fold = mask&1 != 0
	*opt_table_constructor = mask&2 != 0
	*opt_localize_globals = mask&4 != 0
	*opt_licm = mask&8 != 0
	*opt_table_access = mask&16 != 0
	gfilename = "semantic.lua"
	goptcount = 0
	gfilecontent = strings.Split(strings.TrimSuffix(source, "\n"), "\n")
	if err := runOptLuaIterations(); err != nil {
		t.Fatalf("optimize mask=%d: %v\n%s", mask, err, strings.Join(gfilecontent, "\n"))
	}
	return strings.Join(gfilecontent, "\n") + "\n"
}

func TestRuntimeSemantics(t *testing.T) {
	runtimes := luaRuntimes(t)
	files, err := filepath.Glob("testdata/semantic/*.lua")
	if err != nil || len(files) == 0 {
		t.Fatal("missing semantic fixtures", err)
	}
	for _, runtime := range runtimes {
		for _, file := range files {
			t.Run(filepath.Base(runtime)+"/"+filepath.Base(file), func(t *testing.T) {
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				source := string(data)
				want := luaObserve(t, runtime, source)
				// Complete powerset of five passes, deterministic 0..31 order.
				for mask := 0; mask < 32; mask++ {
					t.Run(fmt.Sprintf("mask_%02d", mask), func(t *testing.T) {
						optimized := optimizeSource(t, source, mask)
						got := luaObserve(t, runtime, optimized)
						if got != want {
							t.Errorf("behavior differs\noriginal: %q\noptimized: %q\n%s", want, got, optimized)
						}
						again := optimizeSource(t, optimized, mask)
						if !bytes.Equal([]byte(optimized), []byte(again)) {
							t.Errorf("not idempotent\nfirst:\n%s\nsecond:\n%s", optimized, again)
						}
					})
				}
			})
		}
	}
}

// Safety restrictions must not turn the whole optimizer into a no-op. These
// assertions complement, rather than replace, runtime behavior comparisons.
func TestSafePassesActuallyTransform(t *testing.T) {
	for _, tc := range []struct {
		file string
		mask int
	}{
		{"private_tables", 8}, {"private_tables", 16}, {"private_zero_trip", 8}, {"private_loops", 8}, {"private_constructor", 2}, {"dce_scope", 1},
	} {
		t.Run(tc.file+fmt.Sprint(tc.mask), func(t *testing.T) {
			source, err := os.ReadFile("testdata/semantic/" + tc.file + ".lua")
			if err != nil {
				t.Fatal(err)
			}
			got := optimizeSource(t, string(source), tc.mask)
			if got == string(source) || !strings.Contains(got, "-- opt by oLua") {
				t.Fatal("expected proven-safe optimization", got)
			}
		})
	}
}

func TestStaticEnvironmentLocalizationContract(t *testing.T) {
	source := `local abs=42
local function f()
    local a=math.abs(-1)
    local b=math.abs(-2)
    return a,b,abs
end
return f()
`
	for _, runtime := range luaRuntimes(t) {
		withPassFlags(t, func() { *unsafe_assume_static_globals = true; *opt_localize_globals = true }, func() {
			gfilecontent = strings.Split(strings.TrimSuffix(source, "\n"), "\n")
			if err := runOptLuaIterations(); err != nil {
				t.Fatal(err)
			}
			got := strings.Join(gfilecontent, "\n") + "\n"
			if !strings.Contains(got, "-- opt by oLua") {
				t.Fatal("static-env opt-in must still work")
			}
			if luaObserve(t, runtime, got) != luaObserve(t, runtime, source) {
				t.Fatal("stable environment behavior changed")
			}
			first := got
			if err := runOptLuaIterations(); err != nil {
				t.Fatal(err)
			}
			if first != strings.Join(gfilecontent, "\n")+"\n" {
				t.Fatal("static-env localization not idempotent")
			}
		})
	}
}

func TestBenchmarkWorkloadSemantics(t *testing.T) {
	files, err := filepath.Glob("testdata/bench/*.lua")
	if err != nil || len(files) == 0 {
		t.Fatal("missing benchmarks", err)
	}
	for _, runtime := range luaRuntimes(t) {
		for _, file := range files {
			t.Run(filepath.Base(runtime)+"/"+filepath.Base(file), func(t *testing.T) {
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				// Execute the returned workload without changing its optimizer-visible AST.
				wrap := func(s string) string { return "local workload=(function()\n" + s + "\nend)()\nreturn workload()\n" }
				original := string(data)
				optimized := optimizeSource(t, original, 31)
				if luaObserve(t, runtime, wrap(original)) != luaObserve(t, runtime, wrap(optimized)) {
					t.Fatal("benchmark checksum differs")
				}
				if optimizeSource(t, optimized, 31) != optimized {
					t.Fatal("benchmark not idempotent")
				}
			})
		}
	}
}

func TestRuntimeNumericExpressions(t *testing.T) {
	operands := []string{"0", "-1", "3", "08", "9007199254740993", "9223372036854775807", "(-9223372036854775807 - 1)", "0xffffffffffffffff", "2.5", "-0.0", "9007199254740992.0"}
	var expressions []string
	for _, a := range operands {
		for _, b := range operands {
			for _, op := range []string{"+", "-", "*", "//", "%", "==", "<"} {
				if (op == "//" || op == "%") && (b == "0" || b == "-0.0") {
					continue
				}
				expressions = append(expressions, "("+a+") "+op+" ("+b+")")
			}
		}
	}
	for _, runtime := range luaRuntimes(t) {
		for start := 0; start < len(expressions); start += 24 {
			end := start + 24
			if end > len(expressions) {
				end = len(expressions)
			}
			source := "local function numeric()\n    return " + strings.Join(expressions[start:end], ",\n        ") + "\nend\nreturn numeric()\n"
			want := luaObserve(t, runtime, source)
			for _, mask := range []int{1, 31} {
				got := optimizeSource(t, source, mask)
				if observed := luaObserve(t, runtime, got); observed != want {
					t.Fatalf("numeric chunk=%d mask=%d\nwant=%q\ngot=%q\n%s", start, mask, want, observed, got)
				}
				if optimizeSource(t, got, mask) != got {
					t.Fatal("numeric folding not idempotent")
				}
			}
		}
	}
}

func TestStaticEnvironmentSourceSafety(t *testing.T) {
	sources := []string{
		`local function f()
    local t={tonumber=17}
    local a=tonumber("1")
    local b=tonumber("2")
    return a,b,t.tonumber
end
return f()
`,
		`local function f()
    local text=[=[
tonumber math.abs
]=]
    local a=tonumber("1")
    local b=tonumber("2")
    return a,b,text
end
return f()
`,
	}
	for _, runtime := range luaRuntimes(t) {
		for _, source := range sources {
			withPassFlags(t, func() { *unsafe_assume_static_globals = true; *opt_localize_globals = true }, func() {
				gfilecontent = strings.Split(strings.TrimSuffix(source, "\n"), "\n")
				if err := runOptLuaIterations(); err != nil {
					t.Fatal(err)
				}
				got := strings.Join(gfilecontent, "\n") + "\n"
				if luaObserve(t, runtime, got) != luaObserve(t, runtime, source) {
					t.Fatal("source token rewrite changed literals/keys", got)
				}
			})
		}
	}
}

// Exercise actual localization together with the other passes under its stated
// static-environment contract, in addition to the default-safe corpus matrix.
func TestStaticEnvironmentCombinationMatrix(t *testing.T) {
	source := `local function private()
    local cfg={weight={x=2,y=3}}
    local sum=0
    for i=1,3 do
        sum=sum+cfg.weight.x
        sum=sum+cfg.weight.y
    end
    return sum
end
local function public()
    local record={}
    record.x=2
    record.y=3
    local n=2+3
    local a=math.abs(n)
    local b=math.abs(n-1)
    return a+b+record.x+record.y+n
end
return public(),private()
`
	for _, runtime := range luaRuntimes(t) {
		want := luaObserve(t, runtime, source)
		for mask := 0; mask < 32; mask++ {
			t.Run(filepath.Base(runtime)+fmt.Sprintf("/mask_%02d", mask), func(t *testing.T) {
				got := optimizeSource(t, source, mask, true)
				if luaObserve(t, runtime, got) != want {
					t.Fatal("static-env combination changed behavior", got)
				}
				if optimizeSource(t, got, mask, true) != got {
					t.Fatal("static-env combination not idempotent", got)
				}
				if mask == 31 && !strings.Contains(got, " = math.abs -- opt by oLua") {
					t.Fatal("matrix must exercise actual localization", got)
				}
			})
		}
	}
}
