// Copyright 2017 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package sanitizer

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nkanaev/yarr/src/content/htmlutil"
	"github.com/nkanaev/yarr/src/content/silo"
	"golang.org/x/net/html"
)

var splitSrcsetRegex = regexp.MustCompile(`,\s+`)

// Sanitize returns safe HTML.
func Sanitize(baseURL, input string) string {
	// https://html.spec.whatwg.org/multipage/syntax.html#void-elements
	voidElements := map[string]bool{
		"area":   true,
		"base":   true,
		"br":     true,
		"col":    true,
		"embed":  true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}
	var buffer bytes.Buffer
	var tagStack, blacklistStack []string
	var inSvg bool = false

	tokenizer := html.NewTokenizer(bytes.NewBufferString(input))
	for {
		if tokenizer.Next() == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				if len(tagStack) > 0 {
					fmt.Printf("Malformed HTML at URL '%s', the tag stack has %d elements left: %s\n", baseURL, len(tagStack), tagStack)
				}
				return buffer.String()
			}

			return ""
		}

		token := tokenizer.Token()
		switch token.Type {
		case html.TextToken:
			if len(blacklistStack) > 0 {
				continue
			}

			// An iframe element never has fallback content.
			// See https://www.w3.org/TR/2010/WD-html5-20101019/the-iframe-element.html#the-iframe-element
			if match, _ := popStack("iframe", tagStack); match {
				continue
			}

			buffer.WriteString(html.EscapeString(token.Data))
		case html.StartTagToken:
			tagName := token.Data
			if tagName == "svg" {
				inSvg = true
			}

			if isValidTag(inSvg, tagName) {
				attrNames, htmlAttributes := sanitizeAttributes(baseURL, inSvg, tagName, token.Attr)

				if hasRequiredAttributes(tagName, attrNames) {
					if res, src, thumbnail := isVideoIframe(token); res {
						buffer.WriteString(`<media-player :media="{url:'` + src + `', thumbnail:'` + thumbnail + `', type:'video'}"/>`)
						continue
					}

					if len(attrNames) > 0 {
						buffer.WriteString("<" + tagName + " " + htmlAttributes + ">")
					} else {
						buffer.WriteString("<" + tagName + ">")
					}

					if tagName == "iframe" {
						// autoclose iframes
						buffer.WriteString("</iframe>")
					} else if !voidElements[tagName] {
						tagStack = append(tagStack, tagName)
					}
				}
			} else if isBlockedTag(tagName) {
				blacklistStack = append(blacklistStack, tagName)
			}
		case html.EndTagToken:
			tagName := token.Data
			// iframes are autoclosed. see above
			if tagName == "iframe" {
				continue
			}
			if isValidTag(inSvg, tagName) && inList(tagName, tagStack) {
				buffer.WriteString(fmt.Sprintf("</%s>", tagName))
				_, tagStack = popStack(tagName, tagStack)
				if tagName == "svg" {
					inSvg = false
				}
			} else if isBlockedTag(tagName) {
				_, blacklistStack = popStack(tagName, blacklistStack)
			}
		case html.SelfClosingTagToken:
			tagName := token.Data
			svg := inSvg || tagName == "svg"
			if isValidTag(svg, tagName) {
				attrNames, htmlAttributes := sanitizeAttributes(baseURL, svg, tagName, token.Attr)

				if hasRequiredAttributes(tagName, attrNames) {
					if len(attrNames) > 0 {
						buffer.WriteString("<" + tagName + " " + htmlAttributes + "/>")
					} else {
						buffer.WriteString("<" + tagName + "/>")
					}
				}
			}
		}
	}
}

func sanitizeAttributes(baseURL string, svg bool, tagName string, attributes []html.Attribute) ([]string, string) {
	var htmlAttrs, attrNames []string

	for _, attribute := range attributes {
		value := attribute.Val

		if !isValidAttribute(svg, tagName, attribute.Key) {
			continue
		}

		if (tagName == "img" || tagName == "source") && attribute.Key == "srcset" {
			value = sanitizeSrcsetAttr(baseURL, value)
		}

		if isExternalResourceAttribute(attribute.Key) {
			if tagName == "iframe" {
				if isValidIframeSource(baseURL, attribute.Val) {
					value = attribute.Val
				} else {
					continue
				}
			} else if tagName == "img" && attribute.Key == "src" && isValidDataAttribute(attribute.Val) {
				value = attribute.Val
			} else {
				value = htmlutil.AbsoluteUrl(value, baseURL)
				if value == "" {
					continue
				}

				if !hasValidURIScheme(value) || isBlockedResource(value) {
					continue
				}
			}
		}

		attrNames = append(attrNames, attribute.Key)
		htmlAttrs = append(
			htmlAttrs,
			fmt.Sprintf(`%s="%s"`, attribute.Key, html.EscapeString(value)),
		)
	}

	extraAttrNames, extraHTMLAttributes := getExtraAttributes(tagName)
	if len(extraAttrNames) > 0 {
		attrNames = append(attrNames, extraAttrNames...)
		htmlAttrs = append(htmlAttrs, extraHTMLAttributes...)
	}

	return attrNames, strings.Join(htmlAttrs, " ")
}

func getExtraAttributes(tagName string) ([]string, []string) {
	switch tagName {
	case "a":
		return []string{
				"rel",
				"target",
				"referrerpolicy",
			}, []string{
				`rel="noopener noreferrer"`,
				`target="_blank"`,
				`referrerpolicy="no-referrer"`,
			}
	case "video", "audio":
		return []string{"controls"}, []string{"controls"}
	case "iframe":
		return []string{
				"sandbox",
				"loading",
			}, []string{
				`sandbox="allow-scripts allow-same-origin allow-popups"`,
				`loading="lazy"`,
			}
	case "img":
		return []string{"loading"}, []string{`loading="lazy"`, `referrerpolicy="no-referrer"`}
	default:
		return nil, nil
	}
}

func isValidTag(svg bool, tagName string) bool {
	var x bool
	if svg {
		x = allowedSvgTags.has(tagName) || allowedSvgFilters.has(tagName)
	} else {
		x = allowedTags.has(tagName)
	}
	//fmt.Println(tagName, x)
	return x
}

func isValidAttribute(svg bool, tagName, attributeName string) bool {
	if svg {
		if allowedSvgTags.has(tagName) {
			return allowedSvgAttrs.has(attributeName)
		}
	} else {
		if attrs, ok := allowedAttrs[tagName]; ok {
			return attrs.has(attributeName)
		}
	}
	return false
}

func isExternalResourceAttribute(attribute string) bool {
	switch attribute {
	case "src", "href", "poster", "cite":
		return true
	default:
		return false
	}
}

func hasRequiredAttributes(tagName string, attributes []string) bool {
	elements := make(map[string][]string)
	elements["a"] = []string{"href"}
	elements["iframe"] = []string{"src"}
	elements["img"] = []string{"src"}
	elements["source"] = []string{"src", "srcset"}

	for element, attrs := range elements {
		if tagName == element {
			for _, attribute := range attributes {
				if slices.Contains(attrs, attribute) {
					return true
				}
			}

			return false
		}
	}

	return true
}

// See https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml
func hasValidURIScheme(src string) bool {
	scheme := strings.SplitN(src, ":", 2)[0]
	return allowedURISchemes.has(scheme)
}

func isBlockedResource(src string) bool {
	blacklist := []string{
		"feedsportal.com",
		"api.flattr.com",
		"stats.wordpress.com",
		"plus.google.com/share",
		"twitter.com/share",
		"feeds.feedburner.com",
	}

	for _, element := range blacklist {
		if strings.Contains(src, element) {
			return true
		}
	}

	return false
}

func isValidIframeSource(baseURL, src string) bool {
	whitelist := []string{
		"bandcamp.com",
		"cdn.embedly.com",
		"invidio.us",
		"old.bitchute.com",
		"player.bilibili.com",
		"player.vimeo.com",
		"soundcloud.com",
		"vk.com",
		"w.soundcloud.com",
		"www.bitchute.com",
		"www.dailymotion.com",
		"www.youtube-nocookie.com",
		"www.youtube.com",
	}

	domain := htmlutil.URLDomain(src)
	// allow iframe from same origin
	if htmlutil.URLDomain(baseURL) == domain {
		return true
	}

	return slices.Contains(whitelist, domain)
}

func popStack(name string, stack []string) (bool, []string) {
	l := len(stack)
	if l < 1 || stack[l - 1] != name {
		return false, stack
	}

	return true, stack[:l - 1]
}

func inList(needle string, haystack []string) bool {
	return slices.Contains(haystack, needle)
}

func isBlockedTag(tagName string) bool {
	blacklist := []string{
		"noscript",
		"script",
		"style",
		"head",
	}

	return slices.Contains(blacklist, tagName)
}

/*
One or more strings separated by commas, indicating possible image sources for the user agent to use.

Each string is composed of:
- A URL to an image
- Optionally, whitespace followed by one of:
- A width descriptor (a positive integer directly followed by w). The width descriptor is divided by the source size given in the sizes attribute to calculate the effective pixel density.
- A pixel density descriptor (a positive floating point number directly followed by x).
*/
func sanitizeSrcsetAttr(baseURL, value string) string {
	var sanitizedSources []string
	rawSources := splitSrcsetRegex.Split(value, -1)
	for _, rawSource := range rawSources {
		parts := strings.Split(strings.TrimSpace(rawSource), " ")
		nbParts := len(parts)

		if nbParts > 0 {
			sanitizedSource := parts[0]
			if !strings.HasPrefix(parts[0], "data:") {
				sanitizedSource = htmlutil.AbsoluteUrl(parts[0], baseURL)
				if sanitizedSource == "" {
					continue
				}
			}

			if nbParts == 2 && isValidWidthOrDensityDescriptor(parts[1]) {
				sanitizedSource += " " + parts[1]
			}

			sanitizedSources = append(sanitizedSources, sanitizedSource)
		}
	}
	return strings.Join(sanitizedSources, ", ")
}

func isValidWidthOrDensityDescriptor(value string) bool {
	if value == "" {
		return false
	}

	lastChar := value[len(value)-1:]
	if lastChar != "w" && lastChar != "x" {
		return false
	}

	_, err := strconv.ParseFloat(value[0:len(value)-1], 32)
	return err == nil
}

func isValidDataAttribute(value string) bool {
	var dataAttributeAllowList = []string{
		"data:image/avif",
		"data:image/apng",
		"data:image/png",
		"data:image/svg",
		"data:image/svg+xml",
		"data:image/jpg",
		"data:image/jpeg",
		"data:image/gif",
		"data:image/webp",
	}

	for _, prefix := range dataAttributeAllowList {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func isVideoIframe(token html.Token) (bool, string, string) {
	videoWhitelist := map[string]bool{
		"player.bilibili.com":      true,
		"player.vimeo.com":         true,
		"www.dailymotion.com":      true,
	}
	if token.Data == "iframe" {
		for _, attr := range token.Attr {
			if attr.Key == "src" {
				domain := htmlutil.URLDomain(attr.Val)
				url, thumbnail := silo.VideoIFrameURL(attr.Val)
				if url != "" {
					return true, url, thumbnail
				}
				return videoWhitelist[domain], attr.Val, ""
			}
		}
	}
	return false, "", ""
}
