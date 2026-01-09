package silo

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	frame   = `<iframe src="%s" width="640" height="360" frameborder="0" allowfullscreen></iframe>`
	vimeoRegex   = regexp.MustCompile(`\/(\d+)$`)
)

func VideoIFrameURL(link string) (string, string) {
	l, err := url.Parse(link)
	if err != nil {
		return "", ""
	}

	host := strings.TrimPrefix(l.Host, "www.")
	id := ""
	found := false
	if host == "youtube.com" || host == "youtube-nocookie.com"{
		if l.Path == "/watch" {
			id = l.Query().Get("v")
			found = true
		} else if id, found = strings.CutPrefix(l.Path, "/v/"); found {
		} else if id, found = strings.CutPrefix(l.Path, "/shorts/"); found {
		} else if id, found = strings.CutPrefix(l.Path, "/embed/"); found {
		}
	} else if host == "youtu.be" {
		id = strings.TrimLeft(l.Path, "/")
		found = true
	}
	if found {
		return "https://www.youtube-nocookie.com/embed/" + id, "https://img.youtube.com/vi/" + id + "/hqdefault.jpg"
	}

	if host == "vimeo.com" {
		if matches := vimeoRegex.FindStringSubmatch(l.Path); len(matches) > 0 {
			return "https://player.vimeo.com/video/" + matches[1], ""
		}
	}

	if host == "bitchute.com" || host == "old.bitchute.com" {
		if id, found = strings.CutPrefix(l.Path, "/video/"); found {
		} else if id, found = strings.CutPrefix(l.Path, "/torrent/"); found {
		} else if id, found = strings.CutPrefix(l.Path, "/embed/"); found {
		}
		if found {
			return "https://www.bitchute.com/embed/" + id, ""
		}
	}

	return "", ""
}

func VideoIFrame(link string) string {
	if l, _ := VideoIFrameURL(link); l != "" {
		return fmt.Sprintf(frame, l)
	}

	return ""
}
