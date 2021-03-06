package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
)

func write2Markdown(fileName string, doc *Document) error {
	if err := backupFile(fileName); err != nil {
		return err
	}

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	fmt.Fprintln(w, "## Metrics list")
	fmt.Fprintln(w, "Last updated:", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "Version: %s\n\n", doc.Version)

	for _, c := range doc.Components {
		fmt.Fprintf(w, "### %s\n\n%s\n\n", c.Title, c.Desc)

		fmt.Fprintf(w, "| Metric name | Type | Units | Labels | Introduced in Kata version |\n")
		fmt.Fprintf(w, "|---|---|---|---|---|\n")

		for i := range c.Rows {
			row := c.Rows[i]
			mfName := fmt.Sprintf("`%s`: <br> %s", row.Name, fmtHelp(row.Help))

			labelString := generateHTMLList(row.Labels)

			unit := ""
			if row.Unit != "" {
				unit = fmt.Sprintf("`%s`", row.Unit)
			}
			fmt.Fprintf(w, "| %s | `%s` | %s | %s | 2.0.0 |\n", mfName, row.Type, unit, labelString)
		}

		fmt.Fprintf(w, "\n")
	}

	return w.Flush()
}

func generateHTMLList(labels []*Label) string {
	if len(labels) == 0 {
		return ""
	}

	sort.SliceStable(labels, func(i, j int) bool {
		b := strings.Compare(labels[i].Name, labels[j].Name)
		return b < 0
	})

	s := "<ul>"
	for _, label := range labels {
		s = s + "<li>`" + label.Name + "`"
		if label.Desc != "" {
			s = s + " (" + label.Desc + ")"
		}

		if len(label.Values) > 0 {
			sort.SliceStable(label.Values, func(i, j int) bool {
				b := strings.Compare(label.Values[i].Value, label.Values[j].Value)
				return b < 0
			})

			s = s + "<ul>"
			for _, lv := range label.Values {
				vv := "`" + lv.Value + "`"
				if lv.Desc != "" {
					vv = vv + " (" + lv.Desc + ")"
				}
				s = s + "<li>" + vv + "</li>"
			}
			s = s + "</ul>"
		}
		s = s + "</li>"
	}

	s = s + "</ul>"
	return s
}

var (
	escapeWordList []string
)

func init() {
	file := ".escape_words.txt"

	fi, err := os.Stat(file)
	if err != nil || fi.IsDir() {
		return
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(b), "\n")

	for _, line := range lines {
		words := strings.Split(line, ",")
		for _, word := range words {
			if word != "" {
				escapeWordList = append(escapeWordList, word)
			}
		}
	}
	fmt.Printf("Escape words: %+v\n", escapeWordList)
}

func fmtHelp(s string) string {

	f1 := strings.Split(s, " ")
	r := make([]string, len(f1))
	i := 0
	for i < len(f1) {
		found := false
		t := f1[i]

		for _, ew := range escapeWordList {
			if t == ew {
				found = true
				break
			}
		}

		if found {
			t = "`" + t + "`"
		}
		r[i] = t
		i++
	}

	return strings.Join(r, " ")
}
