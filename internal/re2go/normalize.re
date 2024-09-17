/*!include:re2c "base.re" */

import "strings"

// Original pattern: (?i)\s{2,}
func NormalizeSpaces(input string) string {
	var cursor, marker int
	input += string(rune(0)) // add terminating null
	limit := len(input) - 1  // limit points at the terminating null
	_ = marker

	// Variable for capturing parentheses (twice the number of groups).
	/*!maxnmatch:re2c*/
	yypmatch := make([]int, YYMAXNMATCH*2)
	var yynmatch int
	_ = yynmatch

	// Autogenerated tag variables used by the lexer to track tag values.
	/*!stags:re2c format = 'var @@ int; _ = @@\n'; */

	var start int
	var sb strings.Builder

	for { /*!use:re2c:base_template
		re2c:posix-captures = 1;

		[\t\n\f\r ]{2,} {
			sb.WriteString(input[start:yypmatch[0]])
			sb.WriteString(" ")
			start = yypmatch[1]
			continue
		}

		$ {
			sb.WriteString(input[start:limit])
			return sb.String()
		}

		* { continue }
		*/
	}
}