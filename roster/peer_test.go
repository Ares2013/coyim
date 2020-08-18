package roster

import (
	"github.com/coyim/coyim/xmpp/data"
	"github.com/coyim/coyim/xmpp/jid"
	g "gopkg.in/check.v1"
)

type PeerSuite struct{}

var _ = g.Suite(&PeerSuite{})

func (s *PeerSuite) Test_PeerFrom_returnsANewPeerWithTheSameInformation(c *g.C) {
	re := data.RosterEntry{
		Jid:          "foo@bar.com",
		Subscription: "from",
		Name:         "someone",
		Group: []string{
			"onegroup",
			"twogroup",
		},
	}

	p := PeerFrom(re, "", "", nil)

	c.Assert(p.Jid, g.Equals, tj("foo@bar.com"))
	c.Assert(p.Subscription, g.Equals, "from")
	c.Assert(p.Name, g.Equals, "someone")
	c.Assert(p.Groups, g.DeepEquals, toSet("onegroup", "twogroup"))
}

func (s *PeerSuite) Test_fromSet_returnsTheKeysInTheSet(c *g.C) {
	c.Assert(fromSet(toSet("helo")), g.DeepEquals, []string{"helo"})
	c.Assert(fromSet(toSet()), g.DeepEquals, []string{})
	c.Assert(len(fromSet(toSet("helo1", "helo2", "helo1"))), g.DeepEquals, 2)
}

func (s *PeerSuite) Test_toEntry_ReturnsAnEntryWithTheInformation(c *g.C) {
	p := &Peer{
		Jid:          tj("foo@bar.com"),
		Name:         "something",
		Subscription: "from",
		Groups:       toSet("hello::bar"),
		resources:    make(map[string]Status),
	}

	c.Assert(p.ToEntry().Jid, g.Equals, "foo@bar.com")
	c.Assert(p.ToEntry().Name, g.Equals, "something")
	c.Assert(p.ToEntry().Subscription, g.Equals, "from")
	c.Assert(p.ToEntry().Group, g.DeepEquals, []string{"hello::bar"})
}

func (s *PeerSuite) Test_Dump_willDumpAllInfo(c *g.C) {
	p := &Peer{
		Jid:          tj("foo@bar.com"),
		Name:         "something",
		Subscription: "from",
		Groups:       toSet("hello::bar"),
		resources:    make(map[string]Status),
	}

	c.Assert(p.Dump(), g.Equals, "Peer{foo@bar.com[something ()], subscription='from', status=''('') online=false, asked=false, pendingSubscribe='', belongsTo='', resources=map[], lockedResource=''}")
}

func (s *PeerSuite) Test_PeerWithState_createsANewPeer(c *g.C) {
	p := PeerWithState(tj("bla@foo.com"), "hmm", "no", "", jid.NewResource("1234"))
	c.Assert(p.Jid, g.Equals, tj("bla@foo.com"))
	c.Assert(p.Name, g.Equals, "")
	c.Assert(p.MainStatus(), g.Equals, "hmm")
	c.Assert(p.MainStatusMsg(), g.Equals, "no")
}

func (s *PeerSuite) Test_PeerWithState_hasWorkingResources(c *g.C) {
	p := PeerWithState(tj("bla@foo.com"), "hmm", "no", "", jid.NewResource("1234"))
	c.Assert(p.Resources(), g.DeepEquals, []jid.Resource{jid.NewResource("1234")})
}

func (s *PeerSuite) Test_PeerWithPendingSubscribe_createsNewPeer(c *g.C) {
	p := peerWithPendingSubscribe(tj("bla@foo.com/1234"), "223434", "")
	c.Assert(p.Jid, g.Equals, tj("bla@foo.com"))
	c.Assert(p.Name, g.Equals, "")
	c.Assert(p.PendingSubscribeID, g.Equals, "223434")
}

func (s *PeerSuite) Test_NameForPresentation_returnsTheNameIfItExistsButJidOtherwise(c *g.C) {
	c.Assert((&Peer{Name: "foo", Jid: tj("foo@bar.com")}).NameForPresentation(), g.Equals, "foo")
	c.Assert((&Peer{Jid: tj("foo@bar.com")}).NameForPresentation(), g.Equals, "foo@bar.com")
	c.Assert((&Peer{Jid: tj("jid@coy.im"), Name: "Name", Nickname: "Nick"}).NameForPresentation(), g.Equals, "Nick")
}

func (s *PeerSuite) Test_MergeWith_takesTheFirstGroupsIfExists(c *g.C) {
	p1 := &Peer{
		Groups:    toSet("one"),
		resources: make(map[string]Status),
	}
	p2 := &Peer{}

	c.Assert(fromSet(p1.MergeWith(p2).Groups)[0], g.Equals, "one")
}

func (s *PeerSuite) Test_SetLatestError_setsLatestError(c *g.C) {
	p1 := &Peer{
		Groups:    toSet("one"),
		resources: make(map[string]Status),
	}
	p1.SetLatestError("oen", "tow", "there")

	c.Assert(p1.LatestError, g.DeepEquals, &PeerError{"oen", "tow", "there"})
}
