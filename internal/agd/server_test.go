package agd_test

import (
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardDNS/internal/agd"
	"github.com/AdguardTeam/AdGuardDNS/internal/agdnet"
	"github.com/AdguardTeam/AdGuardDNS/internal/agdtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Common variables for tests.
var (
	bindDataAddrPortV4 = &agd.ServerBindData{
		AddrPort: netip.MustParseAddrPort("1.2.3.4:53"),
	}

	bindDataAddrPortV6 = &agd.ServerBindData{
		AddrPort: netip.MustParseAddrPort("[1234::cdef]:53"),
	}

	bindDataIface = &agd.ServerBindData{
		ListenConfig: &agdtest.ListenConfig{},
		PrefixAddr: &agdnet.PrefixNetAddr{
			Prefix: netip.MustParsePrefix("1.2.3.0/24"),
			Net:    "",
			Port:   53,
		},
	}

	bindDataIfaceSingleIP = &agd.ServerBindData{
		ListenConfig: &agdtest.ListenConfig{},
		PrefixAddr: &agdnet.PrefixNetAddr{
			Prefix: netip.MustParsePrefix("1.2.3.4/32"),
			Net:    "",
			Port:   53,
		},
	}
)

func TestServer_SetBindData(t *testing.T) {
	testCases := []struct {
		name         string
		wantPanicMsg string
		in           []*agd.ServerBindData
	}{{
		name:         "nil",
		wantPanicMsg: "empty bind data",
		in:           nil,
	}, {
		name:         "empty",
		wantPanicMsg: "empty bind data",
		in:           []*agd.ServerBindData{},
	}, {
		name:         "one",
		wantPanicMsg: "",
		in:           []*agd.ServerBindData{bindDataAddrPortV4},
	}, {
		name:         "two_same_type",
		wantPanicMsg: "",
		in: []*agd.ServerBindData{
			bindDataAddrPortV4,
			bindDataAddrPortV6,
		},
	}, {
		name:         "two_diff_type",
		wantPanicMsg: "at index 1: inconsistent type of bind data",
		in: []*agd.ServerBindData{
			bindDataAddrPortV4,
			bindDataIface,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := func() {
				s := &agd.Server{}
				s.SetBindData(tc.in)

				assert.NotEmpty(t, s.BindData())
			}

			if tc.wantPanicMsg == "" {
				assert.NotPanics(t, f)
			} else {
				assert.PanicsWithError(t, tc.wantPanicMsg, f)
			}
		})
	}
}

func TestServer_BindsToInterfaces(t *testing.T) {
	s := &agd.Server{}
	assert.False(t, s.BindsToInterfaces())

	s.SetBindData([]*agd.ServerBindData{bindDataAddrPortV4})
	assert.False(t, s.BindsToInterfaces())

	s.SetBindData([]*agd.ServerBindData{bindDataIface})
	assert.True(t, s.BindsToInterfaces())

	s.SetBindData([]*agd.ServerBindData{bindDataIfaceSingleIP})
	assert.True(t, s.BindsToInterfaces())
}

func TestServer_HasAddr(t *testing.T) {
	testCases := []struct {
		bindData *agd.ServerBindData
		want     assert.BoolAssertionFunc
		name     string
	}{{
		bindData: bindDataAddrPortV4,
		want:     assert.True,
		name:     "addr_has",
	}, {
		bindData: bindDataAddrPortV6,
		want:     assert.False,
		name:     "addr_missing",
	}, {
		bindData: bindDataIfaceSingleIP,
		want:     assert.True,
		name:     "prefix_has",
	}, {
		bindData: bindDataIface,
		want:     assert.False,
		name:     "prefix_missing",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &agd.Server{}
			require.NotPanics(t, func() {
				s.SetBindData([]*agd.ServerBindData{tc.bindData})
			})

			tc.want(t, s.HasAddr(bindDataAddrPortV4.AddrPort))
		})
	}
}

func TestServer_HasIPv6(t *testing.T) {
	s := &agd.Server{}
	assert.False(t, s.HasIPv6())

	s.SetBindData([]*agd.ServerBindData{bindDataAddrPortV4})
	assert.False(t, s.HasIPv6())

	s.SetBindData([]*agd.ServerBindData{bindDataAddrPortV6})
	assert.True(t, s.HasIPv6())
}
