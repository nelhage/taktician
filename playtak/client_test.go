package playtak

import "testing"

func TestParseShout(t *testing.T) {
	cases := []struct {
		in  string
		who string
		msg string
	}{
		{"zzzz", "", ""},
		{"Shout zzzz", "", ""},
		{"Shout <nelhage> hi there", "nelhage", "hi there"},
		{"Shout <IRC> <nelhage> hi there", "IRC", "<nelhage> hi there"},
	}
	for i, tc := range cases {
		who, msg := ParseShout(tc.in)
		if who != tc.who {
			t.Errorf("[%d] got who=%q!=%q",
				i, who, tc.who)
		}
		if msg != tc.msg {
			t.Errorf("[%d] got msg=%q!=%q",
				i, msg, tc.msg)
		}
	}
}

func TestParseShoutRoom(t *testing.T) {
	cases := []struct {
		in   string
		room string
		who  string
		msg  string
	}{
		{"ShoutRoom zzzz", "", "", ""},
		{"ShoutRoom Game1 <nelhage> hi there", "Game1", "nelhage", "hi there"},
		{"ShoutRoom <nelhage> hi there", "", "", ""},
	}
	for i, tc := range cases {
		room, who, msg := ParseShoutRoom(tc.in)
		if room != tc.room {
			t.Errorf("[%d] got room=%q!=%q",
				i, room, tc.room)
		}
		if who != tc.who {
			t.Errorf("[%d] got who=%q!=%q",
				i, who, tc.who)
		}
		if msg != tc.msg {
			t.Errorf("[%d] got msg=%q!=%q",
				i, msg, tc.msg)
		}
	}
}
