/*!include:re2c "base.re" */

// Original pattern: (?i)byline|author|dateline|writtenby|p-author
func IsByline(input string) bool {
	var cursor, marker int
	input += string(rune(0)) // add terminating null
	limit := len(input) - 1  // limit points at the terminating null
	_ = marker

	for { /*!use:re2c:base_template
		re2c:case-insensitive = 1;

		byline = byline|author|dateline|writtenby|p-author;

		{byline} { return true }
		*        { continue }
		$        { return false }
		*/
	}
}