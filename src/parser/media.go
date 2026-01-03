package parser

import (
	"strings"
)

type media struct {
	MediaGroups       []mediaGroup       `xml:"http://search.yahoo.com/mrss/ group"`
	MediaContents     []mediaContent     `xml:"http://search.yahoo.com/mrss/ content"`
	MediaThumbnails   []mediaThumbnail   `xml:"http://search.yahoo.com/mrss/ thumbnail"`
	MediaDescriptions []mediaDescription `xml:"http://search.yahoo.com/mrss/ description"`
}

type mediaGroup struct {
	MediaTitle        string             `xml:"http://search.yahoo.com/mrss/ title"`
	MediaContents     []mediaContent     `xml:"http://search.yahoo.com/mrss/ content"`
	MediaThumbnails   []mediaThumbnail   `xml:"http://search.yahoo.com/mrss/ thumbnail"`
	MediaDescriptions []mediaDescription `xml:"http://search.yahoo.com/mrss/ description"`
	MediaCommunities  []mediaCommunity   `xml:"http://search.yahoo.com/mrss/ community"`
}

type mediaContent struct {
	MediaThumbnails  []mediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"`
	MediaType        string           `xml:"type,attr"`
	MediaMedium      string           `xml:"medium,attr"`
	MediaURL         string           `xml:"url,attr"`
	MediaDescription mediaDescription `xml:"http://search.yahoo.com/mrss/ description"`
}

type mediaThumbnail struct {
	URL string `xml:"url,attr"`
}

type mediaDescription struct {
	Type string `xml:"type,attr"`
	Text string `xml:",chardata"`
}

type mediaCommunity struct {
	MediaStatistics []mediaStatistics `xml:"http://search.yahoo.com/mrss/ statistics"`
}

type mediaStatistics struct {
	Views int64 `xml:"views,attr"`
}

func firstMediaThumbnail(vals ...[]mediaThumbnail) string {
	for _, val := range vals {
		if len(val) > 0 {
			return val[0].URL
		}
	}

	return ""
}

func (m *media) firstMediaDescription() string {
	for _, d := range m.MediaDescriptions {
		return plain2html(d.Text)
	}
	for _, g := range m.MediaGroups {
		for _, d := range g.MediaDescriptions {
			return plain2html(d.Text)
		}
	}
	return ""
}

func (m *media) mediaLinks() []MediaLink {
	links := make([]MediaLink, 0)
	for _, thumbnail := range m.MediaThumbnails {
		links = append(links, MediaLink{URL: thumbnail.URL, Type: "image"})
	}
	for _, group := range m.MediaGroups {
		for _, content := range group.MediaContents {
			if content.MediaURL != "" {
				url := content.MediaURL
				description := firstNonEmpty(content.MediaDescription.Text, group.MediaTitle)
				thumbnail := firstMediaThumbnail(content.MediaThumbnails, group.MediaThumbnails, m.MediaThumbnails)

				if strings.HasPrefix(content.MediaType, "image/") {
					links = append(links, MediaLink{URL: url, Type: "image", Description: description, Thumbnail: thumbnail})
				} else if strings.HasPrefix(content.MediaType, "audio/") {
					links = append(links, MediaLink{URL: url, Type: "audio", Description: description})
				} else if strings.HasPrefix(content.MediaType, "video/") || content.MediaType == "application/x-shockwave-flash" {
					links = append(links, MediaLink{URL: url, Type: "video", Description: description, Thumbnail: thumbnail})
				} else if content.MediaMedium == "image" || content.MediaMedium == "audio" || content.MediaMedium == "video" {
					links = append(links, MediaLink{URL: url, Type: content.MediaMedium, Description: description, Thumbnail: thumbnail})
				} else {
					if thumbnail != "" {
						links = append(links, MediaLink{
							URL:  thumbnail,
							Type: "image",
						})
					}
				}
			}
		}
	}
	for _, content := range m.MediaContents {
		if content.MediaURL != "" {
			url := content.MediaURL
			description := content.MediaDescription.Text
			thumbnail := firstMediaThumbnail(content.MediaThumbnails, m.MediaThumbnails)

			if strings.HasPrefix(content.MediaType, "image/") {
				links = append(links, MediaLink{URL: url, Type: "image", Description: description, Thumbnail: thumbnail})
			} else if strings.HasPrefix(content.MediaType, "audio/") {
				links = append(links, MediaLink{URL: url, Type: "audio", Description: description, Thumbnail: thumbnail})
			} else if strings.HasPrefix(content.MediaType, "video/") || content.MediaType == "application/x-shockwave-flash" {
				links = append(links, MediaLink{URL: url, Type: "video", Description: description, Thumbnail: thumbnail})
			} else if content.MediaMedium == "image" || content.MediaMedium == "audio" || content.MediaMedium == "video" {
				links = append(links, MediaLink{URL: url, Type: content.MediaMedium, Description: description, Thumbnail: thumbnail})
			} else {
				if thumbnail != "" {
					links = append(links, MediaLink{
						URL:  thumbnail,
						Type: "image",
					})
				}
			}
		}
	}
	if len(links) == 0 {
		return nil
	}
	return links
}

func (m *media) hasMediaViews() bool {
	for _, g := range m.MediaGroups {
		for _, c := range g.MediaCommunities {
			for _, s := range c.MediaStatistics {
				return s.Views > 0
			}
		}
	}
	return true
}
