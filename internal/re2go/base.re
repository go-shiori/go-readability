package re2go

/*!rules:re2c:base_template
re2c:eof              = 0;
re2c:yyfill:enable    = 0;
re2c:posix-captures   = 0;
re2c:case-insensitive = 0;

re2c:define:YYCTYPE     = byte;
re2c:define:YYPEEK      = "input[cursor]";
re2c:define:YYSKIP      = "cursor++";
re2c:define:YYBACKUP    = "marker = cursor";
re2c:define:YYRESTORE   = "cursor = marker";
re2c:define:YYLESSTHAN  = "limit <= cursor";
re2c:define:YYSTAGP     = "@@{tag} = cursor";
re2c:define:YYSTAGN     = "@@{tag} = -1";
re2c:define:YYSHIFTSTAG = "@@{tag} += @@{shift}";
*/
