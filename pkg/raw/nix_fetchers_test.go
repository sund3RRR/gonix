package raw

import "testing"

func TestNixFetchersSettingsLifecycle(t *testing.T) {
	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	settings := FetchersSettingsNew(ctx)
	if settings == nil {
		t.Fatalf("FetchersSettingsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FetchersSettingsFree(settings)
	})

	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after FetchersSettingsNew = %v, want %v", got, NixOk)
	}
}
