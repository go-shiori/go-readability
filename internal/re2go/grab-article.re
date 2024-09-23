/*!include:re2c "base.re" */

// Original pattern: (?i)-ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote
func IsUnlikelyCandidates(input string) bool {
	var cursor, marker int
	input += string(rune(0)) // add terminating null
	limit := len(input) - 1  // limit points at the terminating null
	_ = marker

	for { /*!use:re2c:base_template
		re2c:case-insensitive = 1;

		unlikely = -ad-|ai2html|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|footer|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote;

		{unlikely} { return true }
		*          { continue }
		$          { return false }
		*/
	}
}

// Original pattern: (?i)and|article|body|column|content|main|shadow
func MaybeItsACandidate(input string) bool {
	var cursor, marker int
	input += string(rune(0)) // add terminating null
	limit := len(input) - 1  // limit points at the terminating null
	_ = marker

	for { /*!use:re2c:base_template
		re2c:case-insensitive = 1;

		maybe = and|article|body|column|content|main|shadow;

		{maybe} { return true }
		*       { continue }
		$       { return false }
		*/
	}
}

// Commas as used in Latin, Sindhi, Chinese and various other scripts.
// see: https://en.wikipedia.org/wiki/Comma#Comma_variants
// Original pattern: \u002C|\u060C|\uFE50|\uFE10|\uFE11|\u2E41|\u2E34|\u2E32|\uFF0C
func CountCommas(input string) int {
	var count int
	var cursor, marker int
	input += string(rune(0)) // add terminating null
	limit := len(input) - 1  // limit points at the terminating null
	_ = marker

	for { /*!use:re2c:base_template
		re2c:case-insensitive = 1;

		commas = [\u002C\u060C\uFE50\uFE10\uFE11\u2E41\u2E34\u2E32\uFF0C];

		{commas} { count++; continue }
		*        { continue }
		$        { return count }
		*/
	}
}