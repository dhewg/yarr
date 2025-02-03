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

func VideoIFrameURL(link string) string {
	l, err := url.Parse(link)
	if err != nil {
		return ""
	}

	youtubeID := ""
	if l.Host == "www.youtube.com" && l.Path == "/watch" {
		youtubeID = l.Query().Get("v")
	} else if l.Host == "youtu.be" {
		youtubeID = strings.TrimLeft(l.Path, "/")
	}
	if youtubeID != "" {
		return "https://www.youtube-nocookie.com/embed/" + youtubeID
	}

	if l.Host == "vimeo.com" {
		if matches := vimeoRegex.FindStringSubmatch(l.Path); len(matches) > 0 {
			return "https://player.vimeo.com/video/" + matches[1]
		}
	}
	return ""
}

func VideoIFrame(link string) string {
	l := VideoIFrameURL(link)
	if l != "" {
		return fmt.Sprintf(frame, l)
	}

	return ""
}
