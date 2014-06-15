// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nntp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestSanityChecks(t *testing.T) {
	if _, err := Dial("", ""); err == nil {
		t.Fatal("Dial should require at least a destination address.")
	}
}

type faker struct {
	io.Writer
}

func (f faker) Read([]byte) (int, error) {
	panic("reads should be served by the bufio")
}

func (f faker) Close() error {
	return nil
}

func TestBasics(t *testing.T) {
	basicServer = strings.Join(strings.Split(basicServer, "\n"), "\r\n")
	basicClient = strings.Join(strings.Split(basicClient, "\n"), "\r\n")

	var cmdbuf bytes.Buffer
	var fake faker
	fake.Writer = &cmdbuf

	conn := &Conn{conn: fake, w: fake, r: bufio.NewReader(strings.NewReader(basicServer))}

	// Test some global commands that don't take arguments
	if _, err := conn.Capabilities(); err != nil {
		t.Fatal("should be able to request CAPABILITIES after connecting: " + err.Error())
	}

	if exts, err := conn.ListExtensions(); err != nil {
		t.Fatal("should be able to request LIST EXTENSIONS after connecting: " + err.Error())
	} else if strings.Join(exts, " + ") != "HDR + OVER" {
		t.Fatal("should return HDR + OVER")
	}

	_, err := conn.Date()
	if err != nil {
		t.Fatal("should be able to send DATE: " + err.Error())
	}

	/*
		 Test broken until time.Parse adds this format.
		cdate := time.UTC()
		if sdate.Year != cdate.Year || sdate.Month != cdate.Month || sdate.Day != cdate.Day {
			t.Fatal("DATE seems off, probably erroneous: " + sdate.String())
		}
	*/

	// Test LIST (implicit ACTIVE)
	if _, err = conn.List(); err != nil {
		t.Fatal("LIST should work: " + err.Error())
	}

	tt := time.Date(2010, time.March, 01, 00, 0, 0, 0, time.UTC)

	const grp = "gmane.comp.lang.go.general"
	group, err := conn.Group(grp)
	l, h := group.Low, group.High
	if err != nil {
		t.Fatal("Group shouldn't error: " + err.Error())
	}

	// test STAT, NEXT, and LAST
	if _, _, err = conn.Stat(""); err != nil {
		t.Fatal("should be able to STAT after selecting a group: " + err.Error())
	}
	if _, _, err = conn.Next(); err != nil {
		t.Fatal("should be able to NEXT after selecting a group: " + err.Error())
	}
	if _, _, err = conn.Last(); err != nil {
		t.Fatal("should be able to LAST after a NEXT selecting a group: " + err.Error())
	}

	// Can we grab articles?
	a, err := conn.Article(fmt.Sprintf("%d", l))
	if err != nil {
		t.Fatal("should be able to fetch the low article: " + err.Error())
	}
	body, err := ioutil.ReadAll(a.Body)
	if err != nil {
		t.Fatal("error reading reader: " + err.Error())
	}

	// Test that the article body doesn't get mangled.
	expectedbody := `Blah, blah.
.A single leading .
Fin.
`
	if !bytes.Equal([]byte(expectedbody), body) {
		t.Fatalf("article body read incorrectly; got:\n%s\nExpected:\n%s", body, expectedbody)
	}

	// Test articleReader
	expectedart := `Message-Id: <b@c.d>

Body.
`
	a, err = conn.Article(fmt.Sprintf("%d", l+1))
	if err != nil {
		t.Fatal("shouldn't error reading article low+1: " + err.Error())
	}
	var abuf bytes.Buffer
	_, err = a.WriteTo(&abuf)
	if err != nil {
		t.Fatal("shouldn't error writing out article: " + err.Error())
	}
	actualart := abuf.String()
	if actualart != expectedart {
		t.Fatalf("articleReader broke; got:\n%s\nExpected\n%s", actualart, expectedart)
	}

	// Just headers?
	if _, err = conn.Head(fmt.Sprintf("%d", h)); err != nil {
		t.Fatal("should be able to fetch the high article: " + err.Error())
	}

	// Without an id?
	if _, err = conn.Head(""); err != nil {
		t.Fatal("should be able to fetch the selected article without specifying an id: " + err.Error())
	}

	// How about bad articles? Do they error?
	if _, err = conn.Head(fmt.Sprintf("%d", l-1)); err == nil {
		t.Fatal("shouldn't be able to fetch articles lower than low")
	}
	if _, err = conn.Head(fmt.Sprintf("%d", h+1)); err == nil {
		t.Fatal("shouldn't be able to fetch articles higher than high")
	}

	// Just the body?
	r, err := conn.Body(fmt.Sprintf("%d", l))
	if err != nil {
		t.Fatal("should be able to fetch the low article body" + err.Error())
	}
	if _, err = ioutil.ReadAll(r); err != nil {
		t.Fatal("error reading reader: " + err.Error())
	}

	if _, err = conn.NewNews(grp, tt); err != nil {
		t.Fatal("newnews should work: " + err.Error())
	}

	// NewGroups
	if _, err = conn.NewGroups(tt); err != nil {
		t.Fatal("newgroups shouldn't error " + err.Error())
	}

	// Overview
	overviews, err := conn.Overview(10, 11)
	if err != nil {
		t.Fatal("overview shouldn't error: " + err.Error())
	}
	expectedOverviews := []MessageOverview{
		MessageOverview{10, "Subject10", "Author <author@server>", time.Date(2003, 10, 18, 18, 0, 0, 0, time.FixedZone("", 1800)), "<d@e.f>", []string{}, 1000, 9, []string{}},
		MessageOverview{11, "Subject11", "", time.Date(2003, 10, 18, 19, 0, 0, 0, time.FixedZone("", 1800)), "<e@f.g>", []string{"<d@e.f>", "<a@b.c>"}, 2000, 18, []string{"Extra stuff"}},
	}

	if len(overviews) != len(expectedOverviews) {
		t.Fatalf("returned %d overviews, expected %d", len(overviews), len(expectedOverviews))
	}

	for i, o := range overviews {
		if fmt.Sprint(o) != fmt.Sprint(expectedOverviews[i]) {
			t.Fatalf("in place of %dth overview expected %+v, got %+v", i, expectedOverviews[i], o)
		}
	}

	if err = conn.Quit(); err != nil {
		t.Fatal("Quit shouldn't error: " + err.Error())
	}

	actualcmds := cmdbuf.String()
	if basicClient != actualcmds {
		t.Fatalf("Got:\n%s\nExpected\n%s", actualcmds, basicClient)
	}
}

var basicServer = `101 Capability list:
VERSION 2
.
202 Extensions supported:
HDR
OVER
.
111 20100329034158
215 Blah blah
foo 7 3 y
bar 000008 02 m
.
211 100 1 100 gmane.comp.lang.go.general
223 1 <a@b.c> status
223 2 <b@c.d> Article retrieved
223 1 <a@b.c> Article retrieved
220 1 <a@b.c> article
Path: fake!not-for-mail
From: Someone
Newsgroups: gmane.comp.lang.go.general
Subject: [go-nuts] What about base members?
Message-ID: <a@b.c>

Blah, blah.
..A single leading .
Fin.
.
220 2 <b@c.d> article
Message-ID: <b@c.d>

Body.
.
221 100 <c@d.e> head
Path: fake!not-for-mail
Message-ID: <c@d.e>
.
221 100 <c@d.e> head
Path: fake!not-for-mail
Message-ID: <c@d.e>
.
423 Bad article number
423 Bad article number
222 1 <a@b.c> body
Blah, blah.
..A single leading .
Fin.
.
230 list of new articles by message-id follows
<d@e.c>
.
231 New newsgroups follow
.
500 Not supported
224 Overview information for 10-11 follows
10	Subject10	Author <author@server>	Sat, 18 Oct 2003 18:00:00 +0030	<d@e.f>		1000	9
11	Subject11		18 Oct 2003 19:00:00 +0030	<e@f.g>	<d@e.f> <a@b.c>	2000	18	Extra stuff
.
205 Bye!
`

var basicClient = `CAPABILITIES
LIST EXTENSIONS
DATE
LIST
GROUP gmane.comp.lang.go.general
STAT
NEXT
LAST
ARTICLE 1
ARTICLE 2
HEAD 100
HEAD
HEAD 0
HEAD 101
BODY 1
NEWNEWS gmane.comp.lang.go.general 20100301 000000 GMT
NEWGROUPS 20100301 000000 GMT
XZVER 10-11
OVER 10-11
QUIT
`

func TestHacks(t *testing.T) {
	hackServer = strings.Join(strings.Split(hackServer, "\n"), "\r\n")
	hackClient = strings.Join(strings.Split(hackClient, "\n"), "\r\n")

	var cmdbuf bytes.Buffer
	var fake faker
	fake.Writer = &cmdbuf

	conn := &Conn{conn: fake, w: fake, r: bufio.NewReader(strings.NewReader(hackServer))}

	_, err := conn.Group("uk.politics.drugs")
	if err != nil {
		t.Fatal("Group shouldn't error: " + err.Error())
	}

	expectedReferences := []string{
		"<10ne25lo0a67kjvj3p3juj4e6fo12ci90o@4ax.com>",
		"<h08prb$fv7$1@frank-exchange-of-views.oucs.ox.ac.uk>",
		"<A6OdnZqzDpzeabrXnZ2dnUVZ8qKdnZ2d@bt.com>",
		"<h08stk$gu3$1@frank-exchange-of-views.oucs.ox.ac.uk>",
		"<3pWdnaRmM9h1ZbrXnZ2dneKdnZydnZ2d@bt.com>",
		"<h0gorg$st2$1@localhost.localdomain>",
		"<Y7qdnV1ox7f7TLHXnZ2dnUVZ8gKdnZ2d@bt.com>",
		"<1fop259hut1lguvpi1crbtadtpjeaq0ttg@4ax.com>",
		"<J6Odnd0x_MSkd7HXnZ2dnUVZ8q2dnZ2d@bt.com>",
		"<a03bfa3d-9d7b-46f8-a4e4-8d279612a157@y7g2000yqa.googlegroups.com>",
		"<rvadnaNGnI_ulrDXnZ2dnUVZ8nqdnZ2d@bt.com>",
	}

	if articles, err := conn.Overview(55010, 55010); err != nil {
		t.Fatal("Overview shouldn't error: " + err.Error())
	} else if len(articles) != 1 {
		t.Fatalf("Expected 1 article, got: %+v", articles)
	} else if articles[0].Subject != "Re: Supermarket staff stab customer to death for complaining" {
		t.Fatal("Article has wrong subject")
	} else if articles[0].Bytes != 1935 {
		t.Fatal("Article has wrong byte count")
	} else if strings.Join(articles[0].References, "\n") != strings.Join(expectedReferences, "\n") {
		t.Fatal("Article has wrong references")
	}

	if _, err := conn.Overview(53102, 53102); err != nil {
		t.Fatal("Second overview shouldn't error: " + err.Error())
	}

	if _, err := conn.Overview(59034, 59034); err != nil {
		t.Fatal("Third overview shouldn't error: " + err.Error())
	}

	actualcmds := cmdbuf.String()
	if hackClient != actualcmds {
		t.Fatalf("Got: %q\nExpected: %q", actualcmds, hackClient)
	}
}

var hackServer = `211 6117 53009 59125 uk.politics.drugs
500 Not supported
500 What?
224 data follows 
55010	Re: Supermarket staff stab customer to death for complaining	Jethro <jethro_uk@hotmail.com>	Mon, 8 Jun 2009 06:27:41 -0700 (PDT)	<162bacd2-b4d5-490b-b98f-944b1ae27d28@h28g2000yqd.googlegroups.com>	<10ne25lo0a67kjvj3p3juj4e6fo12ci90o@4ax.com> <h08prb$fv7$1@frank-exchange-of-views.oucs.ox.ac.uk> 	<A6OdnZqzDpzeabrXnZ2dnUVZ8qKdnZ2d@bt.com> <h08stk$gu3$1@frank-exchange-of-views.oucs.ox.ac.uk> 	<3pWdnaRmM9h1ZbrXnZ2dneKdnZydnZ2d@bt.com> <h0gorg$st2$1@localhost.localdomain> 	<Y7qdnV1ox7f7TLHXnZ2dnUVZ8gKdnZ2d@bt.com> <1fop259hut1lguvpi1crbtadtpjeaq0ttg@4ax.com> 	<J6Odnd0x_MSkd7HXnZ2dnUVZ8q2dnZ2d@bt.com> <a03bfa3d-9d7b-46f8-a4e4-8d279612a157@y7g2000yqa.googlegroups.com> 	<rvadnaNGnI_ulrDXnZ2dnUVZ8nqdnZ2d@bt.com>	1935	55	Xref: news-big.astraweb.com uk.legal:1033432 uk.politics.misc:1593178 uk.politics.drugs:55010
.
500 What?
224 data follows 
53102	Re: Two men to be hanged for trafficking in cannabis	johannes <johs@sizef3367786864itter.com>	Sat, 11 Oct 2008 00:10:24 +0100	<48EFE0E0.229481B@sizef3367786864itter.com>	<6l64q5FaqjquU1@mid.individual.net> <fNadnZpn3tWKYXDVRVnytQA@pipex.net> 		<6l6f92FavfkiU1@mid.individual.net> <d0eecf41-9924-4955-96a5-a53c8d3ddca8@a2g2000prm.googlegroups.com> 		<329te4dpc4sgvluren79qn4s38ubasea5m@4ax.com> <6l8guvFb321kU2@mid.individual.net> 		<7u2dneydodp0F3LVnZ2dnUVZ8t3inZ2d@bt.com> <62a152c0-f62c-4786-b454-fd9e8d05e68a@m74g2000hsh.googlegroups.com>	1558	27	Xref: news-big.astraweb.com talk.politics.drugs:181567 uk.legal:937888 uk.politics.drugs:53102 rec.drugs.cannabis:25907
.
500 What?
224 data follows 
59034	Campaigning against the 'war on drugs'	tomcosta43@gmail.com	Sat, 1 Sep 2012 05:08:18 -0700 (PDT)	<d6575f22-6b6a-42d2-9d40-08e808f0ee78@googlegroups.com>	<s46dndUc54vjTkLTnZ2dnUVZ8kadnZ2d@bt.com>	1390		Xref: news-big.astraweb.com uk.politics.drugs:59034
.
`

var hackClient = `GROUP uk.politics.drugs
XZVER 55010-55010
OVER 55010-55010
XOVER 55010-55010
OVER 53102-53102
XOVER 53102-53102
OVER 59034-59034
XOVER 59034-59034
`
