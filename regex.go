// 改自 https://github.com/kingwkb/readability python版本
// 于2016-11-10，睡不着了，写起代码来就到零晨4点了
// by: ying32
package readability

import (
	"regexp"
)

var (
	unlikelyCandidates, _   = regexp.Compile(`(?is)combx|comment|community|disqus|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|tweet|twitter|location`)
	okMaybeItsACandidate, _ = regexp.Compile(`(?is)and|article|body|column|main|shadow`)
	positive, _             = regexp.Compile(`(?is)article|body|content|entry|hentry|main|page|pagination|post|text|blog|story`)
	negative, _             = regexp.Compile(`(?is)combx|comment|com|contact|foot|footer|footnote|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget`)
	extraneous, _           = regexp.Compile(`(?is)print|archive|comment|discuss|e[\-]?mail|share|reply|all|login|sign|single`)
	divToPElements, _       = regexp.Compile(`(?is)<(a|blockquote|dl|div|img|ol|p|pre|table|ul)`)
	replaceBrs, _           = regexp.Compile(`(?is)(<br[^>]*>[ \n\r\t]*){2,}`)
	replaceFonts, _         = regexp.Compile(`(?is)<(/?)font[^>]*>`)
	trim, _                 = regexp.Compile(`(?is)^\s+|\s+$`)
	normalize, _            = regexp.Compile(`(?is)\s{2,}`)
	killBreaks, _           = regexp.Compile(`(?is)(<br\s*/?>(\s|&nbsp;?)*)+`)
	videos, _               = regexp.Compile(`(?is)http://(www\.)?(youtube|vimeo)\.com`)
	skipFootnoteLink, _     = regexp.Compile(`(?is)^\s*(\[?[a-z0-9]{1,2}\]?|^|edit|citation needed)\s*$"`)
	nextLink, _             = regexp.Compile(`(?is)(next|weiter|continue|>([^\|]|$)|»([^\|]|$))`)
	prevLink, _             = regexp.Compile(`(?is)(prev|earl|old|new|<|«)`)

	unlikelyElements, _ = regexp.Compile(`(?is)(input|time|button)`)

	pageCodeReg, _ = regexp.Compile(`(?is)<meta.+?charset=[^\w]?([-\w]+)`)
)
