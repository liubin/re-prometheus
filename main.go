package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/prometheus/common/expfmt"

	dto "github.com/prometheus/client_model/go"
)

func main() {

	tmp := os.Getenv("IGNORE_LABELS")
	// this labels will not show detail label values in label list
	ignoreLabels := strings.Split(tmp, ",")

	if len(os.Args) != 2 {
		panic("please use: CMD <metric-endpoint>")
	}
	metricEndpoint := os.Args[1]
	resp, err := http.Get(metricEndpoint)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	reader := bytes.NewReader(body)
	decoder := expfmt.NewDecoder(reader, expfmt.FmtText)

	// get mf list
	list := make([]*dto.MetricFamily, 0)
	for {
		mf := &dto.MetricFamily{}
		if err := decoder.Decode(mf); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		list = append(list, mf)
	}

	sort.SliceStable(list, func(i, j int) bool {
		b := strings.Compare(*list[i].Name, *list[j].Name)
		return b < 0
	})

	// printout mfs
	for i := range list {
		mf := list[i]
		// key is label name, value is label value list
		labelValues := make(map[string]map[string]int)
		for j := range mf.Metric {
			m := mf.Metric[j]
			labels := m.Label
			for k := range labels {
				lk := *labels[k].Name
				lv := *labels[k].Value
				if _, found := labelValues[lk]; found {
					labelValues[lk][lv] = 1
				} else {
					labelValues[lk] = map[string]int{lv: 1}
				}
			}
		}

		fmt.Printf("\n#### %s (%s)\n\n", *mf.Name, mf.Type.String())
		fmt.Println(*mf.Help)

		if len(labelValues) == 0 {
			continue
		}

		fmt.Printf("\nLabels:\n\n")
		for k, v := range labelValues {
			fmt.Printf("  - %s\n", k)
			ignore := false
			for ili := range ignoreLabels {
				if k == ignoreLabels[ili] {
					ignore = true
					break
				}
			}

			if ignore {
				fmt.Printf("    - (depend on env)\n")
				continue
			}

			for vk := range v {
				fmt.Printf("    - %s\n", vk)
			}
		}
	}
}
