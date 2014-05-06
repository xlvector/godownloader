package downloader

import (
	"net"
	"strings"
)

const (
	SPACE     = ' '
	GT        = '>'
	QUOT      = '"'
	SEMICOLON = ';'
	SQUOT     = '\''
	NMARK     = '#'
	COLON     = ':'
	QMARK     = '?'
	AND       = '&'
)

func ExtractDomain(path string) string {
	tks := strings.Split(path, "/")
	if len(tks) < 3 {
		return ""
	}
	ret := tks[0] + "//" + tks[2]
	return ret
}

func ExtractDomainOnly(path string) string {
	tks := strings.Split(path, "/")
	if len(tks) < 3 {
		return ""
	}
	ret := strings.Replace(tks[2], ".", "_", -1)
	return ret
}

func LoopUpHost(path string) string {
	host := ExtractDomain(path)
	addrs, _ := net.LookupHost(host)
	if len(addrs) > 0 {
		return addrs[0]
	} else {
		return ExtractMainDomain(path)
	}
}

func checkDigit(sub string) bool {
	for _, v := range sub {
		if v < '0' || v > '9' {
			return false
		}
	}
	return true
}

func ExtractMainDomain(path string) string {
	tks := strings.Split(path, "/")
	if len(tks) < 3 {
		return ""
	}
	tks2 := strings.Split(tks[2], ".")
	tks3 := strings.Split(tks2[len(tks2)-1], ":")
	if len(tks2) < 3 {
		return tks[2]
	} else if len(tks2) == 4 && checkDigit(tks2[2]) && checkDigit(tks3[0]) {
		return tks[2]
	} else {
		ret := tks2[len(tks2)-2] + "." + tks3[0]
		if ret == "com.cn" || ret == "org.cn" || ret == "net.cn" || ret == "edu.cn" || ret == "gov.cn" {
			ret = tks2[len(tks2)-3] + "." + ret
		}
		return ret
	}
}

func IsValidLink(link string) bool {
	if len(link) < 8 {
		return false
	}
	if len(link) > 200 {
		return false
	}
	for _, ch := range link {
		if ch == NMARK || ch == SEMICOLON {
			return false
		}
		if uint8(ch) > 127 {
			return false
		}
	}
	if strings.Contains(link, "void(") {
		return false
	}
	if strings.Contains(link, "(") {
		return false
	}
	tks := strings.Split(link, ".")
	if len(tks) > 0 && (tks[len(tks)-1] == "jpg" || tks[len(tks)-1] == "css" || tks[len(tks)-1] == "js" || tks[len(tks)-1] == "png" || tks[len(tks)-1] == "gif") {
		return false
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
		if len(srcTks) < n {
			return ""
		}
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
			if i > k+5 && (ch == SPACE || ch == QUOT || ch == SQUOT || ch == GT || ch == NMARK) {
				break
			}
			tmp = append(tmp, ch)
		}
		absLink := ConcatLink(root, string(tmp))
		if IsValidLink(absLink) {
			ret = append(ret, absLink)
		}
	}
	return ret
}
