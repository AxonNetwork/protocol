package util

func StringSetIndex(ss []string, elem string) int {
	for i := range ss {
		if ss[i] == elem {
			return i
		}
	}
	return -1
}

func StringSetAdd(ss []string, elem string) []string {
	if StringSetIndex(ss, elem) == -1 {
		ss = append(ss, elem)
	}
	return ss
}

func StringSetRemove(ss []string, elem string) []string {
	i := StringSetIndex(ss, elem)
	if i == -1 {
		return ss
	} else if i == 0 {
		return ss[1:]
	} else if i == len(ss)-1 {
		return ss[:len(ss)-1]
	} else {
		return append(ss[:i], ss[i+1:]...)
	}
}
