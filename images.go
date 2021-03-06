package goose

import (
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strconv"
	"strings"
)

type candidate struct {
	url     string
	surface int
	score   int
}

var largebig = regexp.MustCompile("(large|big)")

var rules = map[*regexp.Regexp]int{
	regexp.MustCompile("(large|big)"):          1,
	regexp.MustCompile("upload"):               1,
	regexp.MustCompile("media"):                1,
	regexp.MustCompile("gravatar.com"):         -1,
	regexp.MustCompile("feeds.feedburner.com"): -1,
	regexp.MustCompile("(?i)icon"):             -1,
	regexp.MustCompile("(?i)logo"):             -1,
	regexp.MustCompile("(?i)spinner"):          -1,
	regexp.MustCompile("(?i)loading"):          -1,
	regexp.MustCompile("(?i)ads"):              -1,
	regexp.MustCompile("badge"):                -1,
	regexp.MustCompile("1x1"):                  -1,
	regexp.MustCompile("pixel"):                -1,
	regexp.MustCompile("thumbnail[s]*"):        -1,
	regexp.MustCompile(".html|.gif|.ico|button|twitter.jpg|facebook.jpg|ap_buy_photo|digg.jpg|digg.png|delicious.png|facebook.png|reddit.jpg|doubleclick|diggthis|diggThis|adserver|/ads/|ec.atdmt.com|mediaplex.com|adsatt|view.atdmt"): -1,
}

func score(tag *goquery.Selection) int {
	src, _ := tag.Attr("src")
	if src == "" {
		src, _ = tag.Attr("data-src")
	}
	if src == "" {
		src, _ = tag.Attr("data-lazy-src")
	}
	if src == "" {
		return -1
	}
	tagScore := 0
	for rule, score := range rules {
		if rule.MatchString(src) {
			tagScore += score
		}
	}

	alt, exists := tag.Attr("alt")
	if exists {
		if strings.Contains(alt, "thumbnail") {
			tagScore--
		}
	}
	return tagScore
}

func WebPageResolver(article *Article) string {
	doc := article.Doc
	imgs := doc.Find("img")
	topImage := ""
	candidates := make([]candidate, 0)
	significantSurface := 320 * 200
	significantSurfaceCount := 0
	src := ""
	imgs.Each(func(i int, tag *goquery.Selection) {
		surface := 0
		src, _ = tag.Attr("src")
		if src == "" {
			src, _ = tag.Attr("data-src")
		}
		if src == "" {
			src, _ = tag.Attr("data-lazy-src")
		}
		if src == "" {
			return
		}

		width, _ := tag.Attr("width")
		height, _ := tag.Attr("height")
		if width != "" {
			w, _ := strconv.Atoi(width)
			if height != "" {
				h, _ := strconv.Atoi(height)
				surface = w * h
			} else {
				surface = w
			}
		} else {
			if height != "" {
				surface, _ = strconv.Atoi(height)
			} else {
				surface = 0
			}
		}

		if surface > significantSurface {
			significantSurfaceCount++
		}

		tagscore := score(tag)
		if tagscore >= 0 {
			c := candidate{
				url:     src,
				surface: surface,
				score:   score(tag),
			}
			candidates = append(candidates, c)
		}
	})

	if len(candidates) == 0 {
		return ""
	}

	if significantSurfaceCount > 0 {
		bestCandidate := findBestCandidateFromSurface(candidates)
		topImage = bestCandidate.url
	} else {
		bestCandidate := findBestCandidateFromScore(candidates)
		topImage = bestCandidate.url
	}

	if topImage != "" && !strings.HasPrefix(topImage, "http") {
		topImage = "http://" + topImage
	}

	return topImage
}

func findBestCandidateFromSurface(candidates []candidate) candidate {
	max := 0
	var bestCandidate candidate
	for _, candidate := range candidates {
		surface := candidate.surface
		if surface >= max {
			max = surface
			bestCandidate = candidate
		}
	}

	return bestCandidate
}

func findBestCandidateFromScore(candidates []candidate) candidate {
	max := 0
	var bestCandidate candidate
	for _, candidate := range candidates {
		score := candidate.score
		if score >= max {
			max = score
			bestCandidate = candidate
		}
	}

	return bestCandidate
}

type ogTag struct {
	tpe       string
	attribute string
	name      string
	value     string
}

var ogTags = [4]ogTag{
	ogTag{
		tpe:       "facebook",
		attribute: "property",
		name:      "og:image",
		value:     "content",
	},
	ogTag{
		tpe:       "facebook",
		attribute: "rel",
		name:      "image_src",
		value:     "href",
	},
	ogTag{
		tpe:       "twitter",
		attribute: "name",
		name:      "twitter:image",
		value:     "value",
	},
	ogTag{
		tpe:       "twitter",
		attribute: "name",
		name:      "twitter:image",
		value:     "content",
	},
}

type ogImage struct {
	url   string
	tpe   string
	score int
}

func OpenGraphResolver(article *Article) string {
	doc := article.Doc
	meta := doc.Find("meta")
	links := doc.Find("link")
	topImage := ""
	meta = meta.Union(links)
	ogImages := make([]ogImage, 0)
	meta.Each(func(i int, tag *goquery.Selection) {
		for _, ogTag := range ogTags {
			attr, exist := tag.Attr(ogTag.attribute)
			value, vexist := tag.Attr(ogTag.value)
			if exist && attr == ogTag.name && vexist {
				ogImage := ogImage{
					url:   value,
					tpe:   ogTag.tpe,
					score: 0,
				}

				ogImages = append(ogImages, ogImage)
			}
		}
	})

	if len(ogImages) == 1 {
		topImage = ogImages[0].url
	} else {
		for _, ogImage := range ogImages {
			if largebig.MatchString(ogImage.url) {
				ogImage.score++
			}
			if ogImage.tpe == "twitter" {
				ogImage.score++
			}
		}
		topImage = findBestImageFromScore(ogImages).url
	}

	if topImage != "" && !strings.HasPrefix(topImage, "http") {
		topImage = "http://" + topImage
	}

	return topImage
}

func findBestImageFromScore(ogImages []ogImage) ogImage {
	max := 0
	var bestOGImage ogImage
	for _, ogImage := range ogImages {
		score := ogImage.score
		//println("OG", ogImage.url, score)
		if score >= max {
			max = score
			bestOGImage = ogImage
		}
	}

	return bestOGImage
}
