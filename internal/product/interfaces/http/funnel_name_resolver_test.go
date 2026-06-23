package http

import "testing"

func TestResolveFunnelName_FromIndex(t *testing.T) {
	idx := funnelNameIndex([]FunnelInfo{
		{ID: "f-1", Name: "Campanha Previdência"},
		{ID: "f-2", Name: "Funil Aposentados"},
	})

	cases := []struct {
		funnelID, want string
	}{
		{"f-1", "Campanha Previdência"},
		{"f-2", "Funil Aposentados"},
		{"f-desconhecido", missingFunnelLabel},
		{"", missingFunnelLabel},
	}
	for _, c := range cases {
		if got := resolveFunnelName(c.funnelID, idx); got != c.want {
			t.Errorf("resolveFunnelName(%q) = %q; want %q", c.funnelID, got, c.want)
		}
	}
}

func TestResolveFunnelName_EmptyNameTreatedAsMissing(t *testing.T) {
	idx := funnelNameIndex([]FunnelInfo{{ID: "f-1", Name: ""}})
	if got := resolveFunnelName("f-1", idx); got != missingFunnelLabel {
		t.Errorf("got %q; want %q for funnel with empty name", got, missingFunnelLabel)
	}
}

func TestFunnelNameIndex_Empty(t *testing.T) {
	idx := funnelNameIndex(nil)
	if got := resolveFunnelName("anything", idx); got != missingFunnelLabel {
		t.Errorf("got %q; want %q with empty index", got, missingFunnelLabel)
	}
}
