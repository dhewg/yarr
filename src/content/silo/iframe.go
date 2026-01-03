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

	youtubeID := ""
	if l.Host == "www.youtube.com" && l.Path == "/watch" {
		youtubeID = l.Query().Get("v")
	} else if l.Host == "www.youtube.com" && strings.HasPrefix(l.Path, "/v/") {
		youtubeID = strings.TrimPrefix(l.Path, "/v/")
	} else if l.Host == "www.youtube.com" && strings.HasPrefix(l.Path, "/shorts/") {
		youtubeID = strings.TrimPrefix(l.Path, "/shorts/")
	} else if l.Host == "www.youtube.com" && strings.HasPrefix(l.Path, "/embed/") {
		youtubeID = strings.TrimPrefix(l.Path, "/embed/")
	} else if l.Host == "youtu.be" {
		youtubeID = strings.TrimLeft(l.Path, "/")
	}
	if youtubeID != "" {
		return "https://www.youtube-nocookie.com/embed/" + youtubeID, "https://img.youtube.com/vi/" + youtubeID + "/hqdefault.jpg"
	}

	if l.Host == "vimeo.com" {
		if matches := vimeoRegex.FindStringSubmatch(l.Path); len(matches) > 0 {
			return "https://player.vimeo.com/video/" + matches[1], ""
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
