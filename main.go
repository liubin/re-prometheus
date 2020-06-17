package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/prometheus/common/expfmt"

	"gopkg.in/yaml.v3"

	dto "github.com/prometheus/client_model/go"
)

const (
	unitBytes         = "bytes"
	unitSeconds       = "seconds"
	unitMilliseconds  = "milliseconds"
	prefixFieldLength = 2
)

func main() {

	if len(os.Args) != 2 {
		panic("please use: CMD <metric-endpoint>")
	}

	var (
		body []byte
		err  error
	)

	metricEndpoint := os.Args[1]

	if strings.HasPrefix(metricEndpoint, "http") {
		resp, err := http.Get(metricEndpoint)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err = ioutil.ReadAll(resp.Body)
	} else {
		body, err = ioutil.ReadFile(metricEndpoint)
	}

	if err != nil {
		panic(err)
	}

	yamlObj, err := txt2YamlObj(body)
	if err != nil {
		panic(err)
	}

	oldObj, err := loadFromYamlFile("tmp/metrics.yaml")
	if err != nil {
		panic(err)
	}

	oldObj.updateYamlCocument(yamlObj)

	err = write2File("tmp/metrics.yaml", oldObj)
	if err != nil {
		panic(err)
	}

	err = write2Markdown("tmp/metrics.md", oldObj)
	if err != nil {
		panic(err)
	}
}

func loadFromYamlFile(file string) (*Document, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		d := &Document{
			Components: []*Component{},
		}
		return d, nil
	}

	body, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var doc Document

	err = yaml.Unmarshal(body, &doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func txt2YamlObj(body []byte) (*Document, error) {
	unfixedLabels := strings.Split(os.Getenv("UNFIXED_LABELS"), ",")

	componentsMap := map[string]*Component{}

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

	for _, mf := range list {
		prefix := getPrefix(*mf.Name)
		c, found := componentsMap[prefix]
		if !found {
			c = &Component{
				Prefix:           prefix,
				Title:            "FIXME",
				Desc:             "FIXME",
				Rows:             make([]*Row, 0),
				metricFamilyList: make([]*dto.MetricFamily, 0),
			}
			componentsMap[prefix] = c
		}
		c.metricFamilyList = append(c.metricFamilyList, mf)
	}

	for _, c := range componentsMap {
		// printout mfs
		for i := range c.metricFamilyList {
			mf := c.metricFamilyList[i]
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

			processedLabelValues := make(map[string][]string)
			for k, v := range labelValues {
				vvl := false
				for ili := range unfixedLabels {
					if k == unfixedLabels[ili] {
						vvl = true
						break
					}
				}

				if vvl {
					processedLabelValues[k] = []string{""}
				} else {
					vl := make([]string, len(v))
					i := 0
					for k := range v {
						vl[i] = k
						i++
					}
					processedLabelValues[k] = vl
				}
			}

			labels := make([]*Label, len(processedLabelValues))
			i := 0
			for k, v := range processedLabelValues {
				l := &Label{
					Name: k,
				}

				if len(v) == 1 && v[0] == "" {
					l.Fixed = false
				} else {
					l.Fixed = true
					for _, rlv := range v {
						lv := LabelValue{
							Value: rlv,
							// Desc:  rlv,
						}
						l.Values = append(l.Values, &lv)
					}
				}
				labels[i] = l
				i++
			}

			row := &Row{
				Name:   *mf.Name,
				Type:   mf.Type.String(),
				Unit:   guestUnit(*mf.Name, *mf.Help),
				Help:   *mf.Help,
				Labels: labels,
				Since:  "2.0.0",
			}

			c.Rows = append(c.Rows, row)
		}

	}

	components := make([]*Component, len(componentsMap))
	i := 0
	for _, v := range componentsMap {
		components[i] = v
		i++
	}

	d := &Document{
		Components: components,
	}

	return d, nil
}

func write2File(fileName string, data interface{}) error {
	if err := backupFile(fileName); err != nil {
		return err
	}

	buf, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, buf, 0666)
	if err != nil {
		return err
	}
	return nil
}
