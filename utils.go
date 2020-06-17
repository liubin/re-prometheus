package main

import (
	"os"
	"strings"
)

func backupFile(fileName string) error {
	_, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	bak := fileName + ".bak"
	return os.Rename(fileName, bak)
}

func getPrefix(mn string) string {
	fields := strings.Split(mn, "_")
	if len(fields) <= prefixFieldLength {
		return mn
	}

	return strings.Join(fields[:prefixFieldLength], "_")
}

// func generateHTMLList2(labels map[string][]string) string {
// 	if len(labels) == 0 {
// 		return ""
// 	}

// 	keys := make([]string, 0)

// 	for k := range labels {
// 		keys = append(keys, k)
// 	}

// 	sort.SliceStable(keys, func(i, j int) bool {
// 		b := strings.Compare(keys[i], keys[j])
// 		return b < 0
// 	})

// 	s := "<ul>"
// 	for _, k := range keys {
// 		v := labels[k]
// 		s = s + "<li>`" + k + "`"
// 		if len(v) > 0 {

// 			sort.SliceStable(v, func(i, j int) bool {
// 				b := strings.Compare(v[i], v[j])
// 				return b < 0
// 			})

// 			s = s + "<ul>"
// 			for _, vv := range v {
// 				s = s + "<li>`" + vv + "`</li>"
// 			}
// 			s = s + "</ul>"
// 		}
// 		s = s + "</li>"
// 	}

// 	s = s + "</ul>"
// 	return s
// }

func guestUnit(name, help string) string {

	if strings.Index(name, "milliseconds") > 0 {
		return unitMilliseconds
	} else if strings.Index(name, "bytes") > 0 {
		return unitBytes
	} else if strings.Index(name, "seconds") > 0 {
		return unitSeconds
	} else if strings.Index(help, "milliseconds") > 0 {
		return unitMilliseconds
	} else if strings.Index(help, "bytes") > 0 {
		return unitBytes
	} else if strings.Index(help, "seconds") > 0 {
		return unitSeconds
	}
	return ""
}
