nntp.go [![Build Status](https://drone.io/github.com/willglynn/nntp/status.png)](https://drone.io/github.com/willglynn/nntp/latest)
=======

An NNTP (news) Client package for go (golang). Forked from [nntp-go](http://code.google.com/p/nntp-go/) and forked again from [chrisfarms/nntp](https://github.com/chrisfarms/nntp).

This fork contains support for:

* Parsing broken overview responses
* `XOVER` (attempted automatically when `OVER` fails)
* `XZVER` (compressed headers, Astraweb style)
* `XFEATURE COMPRESS GZIP` (compressed headers, Giganews style)

Example
-------

```go
	// connect to news server
	conn, err := nntp.Dial("tcp", "news.example.com:119")
	if err != nil {
		log.Fatalf("connection failed: %v", err)
	}

	// auth
	if err := conn.Authenticate("user", "pass"); err != nil {
		log.Fatalf("Could not authenticate")
	}

	// connect to a news group
	grp := "alt.binaries.pictures"
	_, l, _, err := conn.Group(grp)
	if err != nil {
		log.Fatalf("Could not connect to group %s: %v %d", grp, err, l)
	}

	// fetch an article
	id := "<4c1c18ec$0$8490$c3e8da3@news.astraweb.com>"
	article, err := conn.Article(id)
	if err != nil {
		log.Fatalf("Could not fetch article %s: %v", id, err)
	}

	// read the article contents
	body, err := ioutil.ReadAll(article.Body)
	if err != nil {
		log.Fatalf("error reading reader: %v", err)
	}
```
