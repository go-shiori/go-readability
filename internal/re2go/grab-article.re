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
