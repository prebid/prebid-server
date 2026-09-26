package vastlint

// looksLikeVAST reports whether adm carries a VAST document rather than a
// tag URL, an empty markup, or a non-VAST creative.
func looksLikeVAST(adm string) bool {
	for i := 0; i+5 <= len(adm); i++ {
		if adm[i] != '<' {
			continue
		}
		if eqFoldVAST(adm[i+1 : i+5]) {
			return true
		}
	}
	return false
}

func eqFoldVAST(s string) bool {
	return (s[0] == 'V' || s[0] == 'v') &&
		(s[1] == 'A' || s[1] == 'a') &&
		(s[2] == 'S' || s[2] == 's') &&
		(s[3] == 'T' || s[3] == 't')
}
