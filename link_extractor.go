package downloader

import (
	"strings"
)

const (
	SPACE = ' '
	GT    = '>'
	QUOT  = '"'
	SQUOT = '\''
	NMARK = '#'
	QMARK = '?'
)

func ExtractDomain(path string) string {
	tks := strings.Split(path, "/")
	if len(tks) < 3 {
		return ""
	}
	ret := tks[0] + "//" + tks[2]
	return ret
}

func IsValidLink(link string) bool {
	if len(link) < 8 {
		return false
	}
	if len(link) > 255 {
		return false
	}
	for _, ch := range link {
		if ch == QMARK || ch == NMARK {
			return false
		}
		if uint8(ch) > 127 {
			return false
		}
	}
	return true
}

func NormalizeLink(link string) string {
	tks := strings.Split(link, "://")
	if len(tks) != 2 {
		return ""
	}
	return tks[0] + "://" + strings.Replace(tks[1], "//", "/", -1)
}

func ConcatLink(root0 string, link0 string) string {
	if len(link0) == 0 {
		return root0
	}
	root := root0
	if link0[0] == '/' {
		root = ExtractDomain(root0)
	}
	link := strings.ToLower(link0)
	if strings.Index(link, "http://") == 0 || strings.Index(link, "https://") == 0 {
		return link0
	} else {
		srcTks := strings.Split(root, "/")
		dstTks := strings.Split(link0, "/")
		n := 0
		k := -1
		for i, tk := range dstTks {
			if tk == ".." {
				n += 1
			} else if tk == "." {
				n += 0
			} else {
				k = i
				break
			}
		}
		ret := ""
		for _, tk := range srcTks[:len(srcTks)-n] {
			ret += tk
			ret += "/"
		}
		if k >= 0 {
			for _, tk := range dstTks[k:] {
				if len(tk) == 0 {
					continue
				}
				ret += tk
				ret += "/"
			}
		}
		if link0[len(link0)-1] != '/' {
			return strings.TrimRight(ret, "/")
		} else {
			return ret
		}
	}
}

func ExtractLinks(html []byte, root string) []string {
	ret := []string{}

	for k := 0; k < len(html); k++ {
		next := []byte{}
		for i := k; i < len(html) && i < k+5; i++ {
			next = append(next, html[i])
		}
		if strings.ToLower(string(next)) != "href=" {
			continue
		}
		tmp := []byte{}
		for i := k + 5; i < len(html); i++ {
			ch := html[i]
			if i == k+5 && (ch == SQUOT || ch == QUOT) {
				continue
			}
			if i > k+5 && (ch == SPACE || ch == QUOT || ch == SQUOT || ch == GT) {
				break
			}
			tmp = append(tmp, ch)
		}
		ret = append(ret, ConcatLink(root, string(tmp)))
	}
	return ret
}
