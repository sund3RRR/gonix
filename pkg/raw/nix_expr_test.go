package raw

import (
	"math"
	"sort"
	"strings"
	"testing"
)

func newTestExprState(t *testing.T) (*NixCContext, *Store, *EvalState) {
	t.Helper()

	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	store := newTestStore(t, ctx)
	if got := LibexprInit(ctx); got != NixOk {
		t.Fatalf("LibexprInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	state := StateCreate(ctx, StringArray{}, store)
	if state == nil {
		t.Fatalf("StateCreate returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StateFree(state)
	})

	return ctx, store, state
}

func allocTestValue(t *testing.T, ctx *NixCContext, state *EvalState) *NixValue {
	t.Helper()

	value := AllocValue(ctx, state)
	if value == nil {
		t.Fatalf("AllocValue returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		if got := ValueDecref(ctx, value); got != NixOk {
			t.Fatalf("ValueDecref = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	})

	return value
}

func evalTestExpr(t *testing.T, ctx *NixCContext, state *EvalState, expr string) *NixValue {
	t.Helper()

	value := allocTestValue(t, ctx, state)
	if got := ExprEvalFromString(ctx, state, expr, ".", value); got != NixOk {
		t.Fatalf("ExprEvalFromString(%q) = %v, want %v: %s", expr, got, NixOk, errMsgString(t, ctx))
	}
	if got := ValueForce(ctx, state, value); got != NixOk {
		t.Fatalf("ValueForce(%q) = %v, want %v: %s", expr, got, NixOk, errMsgString(t, ctx))
	}

	return value
}

func exprStringArray(values ...string) StringArray {
	items := make([]StringItem, len(values))
	for i, value := range values {
		items[i] = StringItem{Value: []byte(value), Len: uint64(len(value))}
	}
	return StringArray{Items: items, Len: uint64(len(items))}
}

func TestNixExprInitStateBuilderAndStateLifecycle(t *testing.T) {
	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	store := newTestStore(t, ctx)
	if got := LibexprInit(ctx); got != NixOk {
		t.Fatalf("LibexprInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	builder := EvalStateBuilderNew(ctx, store)
	if builder == nil {
		t.Fatalf("EvalStateBuilderNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		EvalStateBuilderFree(builder)
	})
	if got := EvalStateBuilderSetLookupPath(ctx, builder, StringArray{}); got != NixOk {
		t.Fatalf("EvalStateBuilderSetLookupPath = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := EvalStateBuilderSetLookupPath(ctx, builder, exprStringArray("nixpkgs=/no-such-path")); got != NixOk {
		t.Fatalf("EvalStateBuilderSetLookupPath(non-NUL item) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	state := EvalStateBuild(ctx, builder)
	if state == nil {
		t.Fatalf("EvalStateBuild returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StateFree(state)
	})
}

func TestNixExprEvaluateSimpleValues(t *testing.T) {
	ctx, _, state := newTestExprState(t)

	intValue := evalTestExpr(t, ctx, state, "1 + 2")
	if got := GetType(ctx, intValue); got != NixTypeInt {
		t.Fatalf("GetType(1 + 2) = %v, want %v", got, NixTypeInt)
	}
	if got := GetInt(ctx, intValue); got != 3 {
		t.Fatalf("GetInt(1 + 2) = %d, want 3", got)
	}
	if typename := ownedCString(t, GetTypename(ctx, intValue)); strings.TrimSpace(typename) == "" {
		t.Fatal("GetTypename returned an empty string")
	}

	stringValue := evalTestExpr(t, ctx, state, `"hello"`)
	if got := GetType(ctx, stringValue); got != NixTypeString {
		t.Fatalf("GetType(string) = %v, want %v", got, NixTypeString)
	}
	if got := ownedCString(t, GetString(ctx, stringValue)); got != "hello" {
		t.Fatalf("GetString = %q, want hello", got)
	}

	boolValue := evalTestExpr(t, ctx, state, "true")
	if got := GetType(ctx, boolValue); got != NixTypeBool {
		t.Fatalf("GetType(bool) = %v, want %v", got, NixTypeBool)
	}
	if !GetBool(ctx, boolValue) {
		t.Fatal("GetBool(true) = false, want true")
	}

	floatValue := evalTestExpr(t, ctx, state, "1.25")
	if got := GetType(ctx, floatValue); got != NixTypeFloat {
		t.Fatalf("GetType(float) = %v, want %v", got, NixTypeFloat)
	}
	if got := GetFloat(ctx, floatValue); math.Abs(got-1.25) > 0.000001 {
		t.Fatalf("GetFloat = %f, want 1.25", got)
	}

	nullValue := evalTestExpr(t, ctx, state, "null")
	if got := GetType(ctx, nullValue); got != NixTypeNull {
		t.Fatalf("GetType(null) = %v, want %v", got, NixTypeNull)
	}
}

func TestNixExprListsAndAttrs(t *testing.T) {
	ctx, _, state := newTestExprState(t)

	listValue := evalTestExpr(t, ctx, state, "[ 1 2 ]")
	if got := GetType(ctx, listValue); got != NixTypeList {
		t.Fatalf("GetType(list) = %v, want %v", got, NixTypeList)
	}
	if got := GetListSize(ctx, listValue); got != 2 {
		t.Fatalf("GetListSize = %d, want 2", got)
	}
	listItem := GetListByidx(ctx, listValue, state, 1)
	if listItem == nil {
		t.Fatalf("GetListByidx returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		if got := ValueDecref(ctx, listItem); got != NixOk {
			t.Fatalf("ValueDecref(list item) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	})
	if got := GetInt(ctx, listItem); got != 2 {
		t.Fatalf("GetInt(list[1]) = %d, want 2", got)
	}

	attrValue := evalTestExpr(t, ctx, state, `{ a = 1; b = "x"; }`)
	if got := GetType(ctx, attrValue); got != NixTypeAttrs {
		t.Fatalf("GetType(attrs) = %v, want %v", got, NixTypeAttrs)
	}
	if got := GetAttrsSize(ctx, attrValue); got != 2 {
		t.Fatalf("GetAttrsSize = %d, want 2", got)
	}
	if !HasAttrByname(ctx, attrValue, state, "a") {
		t.Fatal("HasAttrByname(a) = false, want true")
	}
	attrA := GetAttrByname(ctx, attrValue, state, "a")
	if attrA == nil {
		t.Fatalf("GetAttrByname(a) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		if got := ValueDecref(ctx, attrA); got != NixOk {
			t.Fatalf("ValueDecref(attr a) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	})
	if got := GetInt(ctx, attrA); got != 1 {
		t.Fatalf("GetInt(attr a) = %d, want 1", got)
	}

	names := []string{
		ownedCString(t, GetAttrNameByidx(ctx, attrValue, state, 0)),
		ownedCString(t, GetAttrNameByidx(ctx, attrValue, state, 1)),
	}
	sort.Strings(names)
	if names[0] != "a" || names[1] != "b" {
		t.Fatalf("attribute names = %v, want [a b]", names)
	}

	attrByIndex := GetAttrByidx(ctx, attrValue, state, 0)
	if attrByIndex == nil {
		t.Fatalf("GetAttrByidx returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		if got := ValueDecref(ctx, attrByIndex); got != NixOk {
			t.Fatalf("ValueDecref(attr by index) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	})
}

func TestNixExprInitializersBuildersAndCallMulti(t *testing.T) {
	ctx, _, state := newTestExprState(t)

	intValue := allocTestValue(t, ctx, state)
	if got := InitInt(ctx, intValue, 7); got != NixOk {
		t.Fatalf("InitInt = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	stringValue := allocTestValue(t, ctx, state)
	if got := InitString(ctx, stringValue, "seven"); got != NixOk {
		t.Fatalf("InitString = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	boolValue := allocTestValue(t, ctx, state)
	if got := InitBool(ctx, boolValue, true); got != NixOk {
		t.Fatalf("InitBool = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	nullValue := allocTestValue(t, ctx, state)
	if got := InitNull(ctx, nullValue); got != NixOk {
		t.Fatalf("InitNull = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	pathValue := allocTestValue(t, ctx, state)
	if got := InitPathString(ctx, state, pathValue, "/nix/store"); got != NixOk {
		t.Fatalf("InitPathString = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if path := ownedCString(t, GetPathString(ctx, pathValue)); path != "/nix/store" {
		t.Fatalf("GetPathString = %q, want /nix/store", path)
	}

	listOut := allocTestValue(t, ctx, state)
	listBuilder := MakeListBuilder(ctx, state, 2)
	if listBuilder == nil {
		t.Fatalf("MakeListBuilder returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		ListBuilderFree(listBuilder)
	})
	if got := ListBuilderInsert(ctx, listBuilder, 0, intValue); got != NixOk {
		t.Fatalf("ListBuilderInsert(0) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := ListBuilderInsert(ctx, listBuilder, 1, stringValue); got != NixOk {
		t.Fatalf("ListBuilderInsert(1) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := MakeList(ctx, listBuilder, listOut); got != NixOk {
		t.Fatalf("MakeList = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := GetListSize(ctx, listOut); got != 2 {
		t.Fatalf("GetListSize(manual) = %d, want 2", got)
	}

	attrsOut := allocTestValue(t, ctx, state)
	attrsBuilder := MakeBindingsBuilder(ctx, state, 1)
	if attrsBuilder == nil {
		t.Fatalf("MakeBindingsBuilder returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		BindingsBuilderFree(attrsBuilder)
	})
	if got := BindingsBuilderInsert(ctx, attrsBuilder, "answer", intValue); got != NixOk {
		t.Fatalf("BindingsBuilderInsert = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := MakeAttrs(ctx, attrsOut, attrsBuilder); got != NixOk {
		t.Fatalf("MakeAttrs = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if !HasAttrByname(ctx, attrsOut, state, "answer") {
		t.Fatal("manual attrs missing answer")
	}

	fn := evalTestExpr(t, ctx, state, "x: y: x + y")
	arg1 := allocTestValue(t, ctx, state)
	if got := InitInt(ctx, arg1, 2); got != NixOk {
		t.Fatalf("InitInt(arg1) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	arg2 := allocTestValue(t, ctx, state)
	if got := InitInt(ctx, arg2, 3); got != NixOk {
		t.Fatalf("InitInt(arg2) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	callResult := allocTestValue(t, ctx, state)
	args := ValueArray{
		Items: []ValueItem{{Value: arg1}, {Value: arg2}},
		Len:   2,
	}
	if got := ValueCallMulti(ctx, state, fn, args, callResult); got != NixOk {
		t.Fatalf("ValueCallMulti = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := ValueForce(ctx, state, callResult); got != NixOk {
		t.Fatalf("ValueForce(callResult) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := GetInt(ctx, callResult); got != 5 {
		t.Fatalf("GetInt(callResult) = %d, want 5", got)
	}
}

func TestNixExprValueCallMultiRejectsNilArrayItem(t *testing.T) {
	ctx, _, state := newTestExprState(t)

	fn := evalTestExpr(t, ctx, state, "x: x")
	result := allocTestValue(t, ctx, state)
	args := ValueArray{
		Items: []ValueItem{{Value: nil}},
		Len:   1,
	}
	if got := ValueCallMulti(ctx, state, fn, args, result); got == NixOk {
		t.Fatal("ValueCallMulti accepted a nil ValueArray item")
	}
	if msg := errMsgString(t, ctx); strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after nil ValueArray item returned an empty string")
	}
	ClearErr(ctx)
}

func TestNixExprRealisedString(t *testing.T) {
	ctx, _, state := newTestExprState(t)

	value := evalTestExpr(t, ctx, state, `"plain"`)
	realised := StringRealise(ctx, state, value, false)
	if realised == nil {
		t.Fatalf("StringRealise returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		RealisedStringFree(realised)
	})
	if got := ownedCString(t, RealisedStringGetBuffer(realised)); got != "plain" {
		t.Fatalf("RealisedStringGetBuffer = %q, want plain", got)
	}
	if got := RealisedStringGetBufferSize(realised); got != 5 {
		t.Fatalf("RealisedStringGetBufferSize = %d, want 5", got)
	}
	if got := RealisedStringGetStorePathCount(realised); got != 0 {
		t.Fatalf("RealisedStringGetStorePathCount = %d, want 0", got)
	}
}

func TestNixExprInvalidExpressionSetsContextError(t *testing.T) {
	ctx, _, state := newTestExprState(t)
	value := allocTestValue(t, ctx, state)

	if got := ExprEvalFromString(ctx, state, "let =", ".", value); got == NixOk {
		t.Fatalf("ExprEvalFromString(invalid) = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after invalid ExprEvalFromString returned an empty string")
	}
	ClearErr(ctx)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after ClearErr = %v, want %v", got, NixOk)
	}
}
