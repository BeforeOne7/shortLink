package main

type Storage interface {
	Shorten(url string, exp int64) (string, error)
	ShortLinkInfo(eid string) (URLDetail, error)
	UnShorten(eid string) (string, error)
}
