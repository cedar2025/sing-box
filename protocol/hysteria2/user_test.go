package hysteria2

import (
	"context"
	"net"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/listener"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	hy2 "github.com/sagernet/sing-quic/hysteria2"
	"github.com/sagernet/sing/common/auth"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type recordingRouter struct {
	adapter.Router
	tcp []string
	udp []string
}

func (r *recordingRouter) RouteConnectionEx(_ context.Context, _ net.Conn, m adapter.InboundContext, _ N.CloseHandlerFunc) {
	r.tcp = append(r.tcp, m.User)
}
func (r *recordingRouter) RoutePacketConnectionEx(_ context.Context, _ N.PacketConn, m adapter.InboundContext, _ N.CloseHandlerFunc) {
	r.udp = append(r.udp, m.User)
}

func TestAuthenticatedNameSurvivesUserRefresh(t *testing.T) {
	router := new(recordingRouter)
	h := &Inbound{service: new(hy2.Service[string]), router: router, listener: new(listener.Listener), logger: log.NewNOPFactory().NewLogger("test")}
	alice := option.Hysteria2User{Name: "alice", Password: "fixture-a"}
	bob := option.Hysteria2User{Name: "bob", Password: "fixture-b"}
	unnamed := option.Hysteria2User{Password: "fixture-unnamed"}
	if err := h.UpdateUsers([]option.Hysteria2User{alice, bob, unnamed}); err != nil {
		t.Fatal(err)
	}
	bobSession := auth.ContextWithUser(t.Context(), bob.Name)
	anonymousSession := auth.ContextWithUser(t.Context(), unnamed.Name)
	if err := h.UpdateUsers([]option.Hysteria2User{unnamed, bob}); err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []context.Context{bobSession, anonymousSession} {
		h.NewConnectionEx(ctx, nil, M.Socksaddr{}, M.Socksaddr{}, nil)
		h.NewPacketConnectionEx(ctx, nil, M.Socksaddr{}, M.Socksaddr{}, nil)
	}
	for _, users := range [][]string{router.tcp, router.udp} {
		if len(users) != 2 || users[0] != "bob" || users[1] != "" {
			t.Errorf("routed users = %q, want [bob <unnamed>]", users)
		}
	}
}
