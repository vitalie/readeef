package processor

import (
	"bytes"
	"net/url"
	"strings"
	"text/template"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/urandom/readeef/parser"
	"github.com/urandom/webfw"
	"github.com/urandom/webfw/util"
)

type ProxyHTTP struct {
	logger      webfw.Logger
	urlTemplate *template.Template
}

func NewProxyHTTP(l webfw.Logger, urlTemplate string) (ProxyHTTP, error) {
	l.Infof("URL Template: %s\n", urlTemplate)
	t, err := template.New("proxy-http-url-template").Parse(urlTemplate)
	if err != nil {
		return ProxyHTTP{}, err
	}

	return ProxyHTTP{logger: l, urlTemplate: t}, nil
}

func (p ProxyHTTP) Process(f parser.Feed) parser.Feed {
	p.logger.Infof("Proxying urls of feed '%s'\n", f.Title)

	for i := range f.Articles {
		if d, err := goquery.NewDocumentFromReader(strings.NewReader(f.Articles[i].Description)); err == nil {
			if ProxyArticleLinks(d, p.urlTemplate) {
				if content, err := d.Html(); err == nil {
					// net/http tries to provide valid html, adding html, head and body tags
					content = content[strings.Index(content, "<body>")+6 : strings.LastIndex(content, "</body>")]

					f.Articles[i].Description = content
				}
			}
		}
	}

	return f
}

func ProxyArticleLinks(d *goquery.Document, urlTemplate *template.Template) bool {
	changed := false
	d.Find("[src]").Each(func(i int, s *goquery.Selection) {
		var val string
		var ok bool

		if val, ok = s.Attr("src"); !ok {
			return
		}

		if link, err := processUrl(val, urlTemplate); err == nil {
			s.SetAttr("src", link)
		} else {
			return
		}

		changed = true
		return
	})

	d.Find("[srcset]").Each(func(i int, s *goquery.Selection) {
		var val string
		var ok bool

		if val, ok = s.Attr("srcset"); !ok {
			return
		}

		var res, buf bytes.Buffer

		expectUrl := true
		for _, r := range val {
			if unicode.IsSpace(r) {
				if buf.Len() != 0 {
					// From here on, only descriptors follow, until the end, or the comma
					expectUrl = false
					if link, err := processUrl(buf.String(), urlTemplate); err == nil {
						res.WriteString(link)
					} else {
						return
					}
					buf.Reset()
				} // Else, whitespace right before the link
				res.WriteRune(r)
			} else if r == ',' {
				// End of the current image candidate string
				expectUrl = true
				res.WriteRune(r)
			} else if expectUrl {
				// The link
				buf.WriteRune(r)
			} else {
				// The actual descriptor text
				res.WriteRune(r)
			}
		}

		if buf.Len() > 0 {
			if link, err := processUrl(buf.String(), urlTemplate); err == nil {
				res.WriteString(link)
			} else {
				return
			}
			buf.Reset()
		}

		s.SetAttr("srcset", res.String())

		changed = true
		return
	})

	return changed
}

func processUrl(link string, urlTemplate *template.Template) (string, error) {
	u, err := url.Parse(link)
	if err != nil || !u.IsAbs() || u.Scheme != "http" {
		return "", err
	}

	buf := util.BufferPool.GetBuffer()
	defer util.BufferPool.Put(buf)

	if err := urlTemplate.Execute(buf, url.QueryEscape(u.String())); err != nil {
		return "", err
	}

	return buf.String(), nil
}
