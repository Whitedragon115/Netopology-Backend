package server

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
)

// TestContract compares the OpenAPI that the Go code generates with the hand-written contract
// (api/contract.yaml), which the frontend is built against.
//
// For every operation the Go code registers, these must match the contract: method + path,
// operationId, x-permission, tag, security (public / session / guest), success status,
// parameters (name, in, required), request and success-response schema names, and the property
// names + required list of every schema they use. Error statuses listed in the contract must be
// declared too (httpx.Errors).
//
// Operations in the contract that are not registered yet are reported as pending. Set
// CONTRACT_STRICT=1 to fail on pending operations (CI does this once WP-14 is done).
func TestContract(t *testing.T) {
	contract := loadYAML(t, "../../api/contract.yaml")
	genYAML, err := OpenAPI().YAML()
	if err != nil {
		t.Fatal(err)
	}
	var gen map[string]any
	if err := yaml.Unmarshal(genYAML, &gen); err != nil {
		t.Fatal(err)
	}

	cOps, gOps := operations(contract), operations(gen)
	var pending []string
	for key, cop := range cOps {
		gop, ok := gOps[key]
		if !ok {
			pending = append(pending, fmt.Sprintf("%-50s %s", key, str(cop["operationId"])))
			continue
		}
		t.Run(str(cop["operationId"]), func(t *testing.T) { compareOp(t, contract, gen, cop, gop) })
	}
	for key, gop := range gOps {
		if _, ok := cOps[key]; !ok {
			t.Errorf("%s (%s) is not in the contract: add it to api/contract.yaml first", key, str(gop["operationId"]))
		}
	}
	sort.Strings(pending)
	if len(pending) > 0 {
		msg := fmt.Sprintf("%d/%d contract operations not implemented yet:\n  %s",
			len(pending), len(cOps), strings.Join(pending, "\n  "))
		if os.Getenv("CONTRACT_STRICT") == "1" {
			t.Error(msg)
		} else {
			t.Log(msg)
		}
	}
}

// TestPermissionKeys: the authz catalog and the contract's x-permission keys are the same set.
func TestPermissionKeys(t *testing.T) {
	contract := loadYAML(t, "../../api/contract.yaml")
	want := map[string]bool{}
	for _, op := range operations(contract) {
		want[str(op["x-permission"])] = true
	}
	have := map[string]bool{}
	for _, p := range authz.All() {
		have[string(p)] = true
	}
	for k := range want {
		if !have[k] {
			t.Errorf("x-permission %q is missing from internal/authz/permission.go", k)
		}
	}
	for k := range have {
		if !want[k] {
			t.Errorf("internal/authz has %q but no contract operation uses it", k)
		}
	}
}

// TestEventTypes: events.AllTypes equals the contract's Event.type enum.
func TestEventTypes(t *testing.T) {
	contract := loadYAML(t, "../../api/contract.yaml")
	enum := dig(contract, "components", "schemas", "Event", "properties", "type", "enum")
	var want []string
	for _, v := range enum.([]any) {
		want = append(want, str(v))
	}
	var have []string
	for _, ty := range events.AllTypes {
		have = append(have, string(ty))
	}
	slices.Sort(want)
	slices.Sort(have)
	if !slices.Equal(want, have) {
		t.Errorf("event types differ\ncontract: %v\ngo:       %v", want, have)
	}
}

func compareOp(t *testing.T, contract, gen, cop, gop map[string]any) {
	t.Helper()
	for _, f := range []string{"operationId", "x-permission"} {
		if str(cop[f]) != str(gop[f]) {
			t.Errorf("%s: contract %q, go %q", f, str(cop[f]), str(gop[f]))
		}
	}
	if c, g := fmt.Sprint(cop["tags"]), fmt.Sprint(gop["tags"]); c != g {
		t.Errorf("tags: contract %s, go %s", c, g)
	}
	if c, g := securityKind(cop), securityKind(gop); c != g {
		t.Errorf("security: contract %s, go %s", c, g)
	}

	cOK, gOK := successStatus(cop), successStatus(gop)
	if cOK != gOK {
		t.Errorf("success status: contract %s, go %s", cOK, gOK)
	}
	for code := range asMap(cop["responses"]) {
		if !strings.HasPrefix(code, "2") && !strings.HasPrefix(code, "3") {
			if _, ok := asMap(gop["responses"])[code]; !ok {
				t.Errorf("error status %s is in the contract but not declared (httpx.Errors)", code)
			}
		}
	}

	if c, g := params(contract, cop), params(gen, gop); !slices.Equal(c, g) {
		t.Errorf("parameters differ\ncontract: %v\ngo:       %v", c, g)
	}

	cReq := bodySchema(contract, dig(cop, "requestBody", "content"))
	gReq := bodySchema(gen, dig(gop, "requestBody", "content"))
	compareSchema(t, "request body", contract, gen, cReq, gReq)

	cRes := bodySchema(contract, dig(cop, "responses", cOK, "content"))
	gRes := bodySchema(gen, dig(gop, "responses", gOK, "content"))
	compareSchema(t, "response "+cOK, contract, gen, cRes, gRes)
}

func compareSchema(t *testing.T, what string, contract, gen map[string]any, c, g schemaRef) {
	t.Helper()
	if c.name != g.name {
		t.Errorf("%s schema: contract %q, go %q (name the Go type like the contract schema)", what, c.name, g.name)
		return
	}
	if c.def == nil || g.def == nil {
		return
	}
	cp, gp := propNames(c.def), propNames(g.def)
	if !slices.Equal(cp, gp) {
		t.Errorf("%s %s properties differ\ncontract: %v\ngo:       %v", what, c.name, cp, gp)
	}
	cr, gr := strList(c.def["required"]), strList(g.def["required"])
	if !slices.Equal(cr, gr) {
		t.Errorf("%s %s required differ\ncontract: %v\ngo:       %v", what, c.name, cr, gr)
	}
}

// ---- helpers over the generic YAML tree ----

type schemaRef struct {
	name string // component schema name ("" for inline / none)
	def  map[string]any
}

func operations(doc map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for path, item := range asMap(doc["paths"]) {
		for method, op := range asMap(item) {
			switch method {
			case "get", "post", "put", "patch", "delete":
				out[strings.ToUpper(method)+" "+path] = asMap(op)
			}
		}
	}
	return out
}

func successStatus(op map[string]any) string {
	var codes []string
	for code := range asMap(op["responses"]) {
		if strings.HasPrefix(code, "2") || strings.HasPrefix(code, "3") {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	return strings.Join(codes, ",")
}

func securityKind(op map[string]any) string {
	sec, ok := op["security"]
	if !ok {
		return "session"
	}
	list, _ := sec.([]any)
	if len(list) == 0 {
		return "public"
	}
	var names []string
	for _, s := range list {
		for k := range asMap(s) {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	return strings.Join(names, "+")
}

// params returns "in:name:required" for every parameter, $refs resolved, sorted.
func params(doc, op map[string]any) []string {
	var out []string
	for _, p := range asSlice(op["parameters"]) {
		pm := resolve(doc, asMap(p))
		out = append(out, fmt.Sprintf("%s:%s:%v", str(pm["in"]), str(pm["name"]), pm["required"] == true))
	}
	sort.Strings(out)
	return out
}

func bodySchema(doc map[string]any, content any) schemaRef {
	for ct, media := range asMap(content) {
		if !strings.Contains(ct, "json") {
			continue
		}
		s := asMap(asMap(media)["schema"])
		name := ""
		for {
			ref := str(s["$ref"])
			if ref == "" {
				break
			}
			name = ref[strings.LastIndex(ref, "/")+1:]
			s = asMap(dig(doc, "components", "schemas", name))
		}
		return schemaRef{name: name, def: s}
	}
	return schemaRef{}
}

func resolve(doc, m map[string]any) map[string]any {
	ref := str(m["$ref"])
	if ref == "" {
		return m
	}
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	keys := make([]any, len(parts))
	for i, p := range parts {
		keys[i] = p
	}
	return resolve(doc, asMap(digAny(doc, keys...)))
}

func propNames(s map[string]any) []string {
	var out []string
	for k := range asMap(s["properties"]) {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func dig(m map[string]any, keys ...string) any {
	anyKeys := make([]any, len(keys))
	for i, k := range keys {
		anyKeys[i] = k
	}
	return digAny(m, anyKeys...)
}

func digAny(m map[string]any, keys ...any) any {
	var cur any = m
	for _, k := range keys {
		cur = asMap(cur)[fmt.Sprint(k)]
	}
	return cur
}

func asMap(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[fmt.Sprint(k)] = val
		}
		return out
	}
	return map[string]any{}
}

func asSlice(v any) []any { s, _ := v.([]any); return s }

func strList(v any) []string {
	var out []string
	for _, x := range asSlice(v) {
		out = append(out, str(x))
	}
	sort.Strings(out)
	return out
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func loadYAML(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
