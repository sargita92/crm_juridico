package infrastructure

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestInvitesTotal_Increments(t *testing.T) {
	before := testutil.ToFloat64(InvitesTotal.WithLabelValues("sent"))
	InvitesTotal.WithLabelValues("sent").Inc()
	after := testutil.ToFloat64(InvitesTotal.WithLabelValues("sent"))
	assert.Equal(t, before+1, after)
}

func TestInvitesTotal_AcceptsAllOutcomes(t *testing.T) {
	outcomes := []string{"sent", "accepted", "revoked"}
	for _, o := range outcomes {
		before := testutil.ToFloat64(InvitesTotal.WithLabelValues(o))
		InvitesTotal.WithLabelValues(o).Inc()
		after := testutil.ToFloat64(InvitesTotal.WithLabelValues(o))
		assert.GreaterOrEqual(t, after, before+1, "outcome %s should increment", o)
	}
}

func TestInvitesTotal_Registered(t *testing.T) {
	InvitesTotal.WithLabelValues("sent").Inc()
	mfs, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	names := make([]string, 0, len(mfs))
	for _, mf := range mfs {
		names = append(names, mf.GetName())
	}
	joined := strings.Join(names, ",")
	assert.Contains(t, joined, "crm_auth_invites_total")
}
