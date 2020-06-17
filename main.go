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

type LabelValue struct {
	Value string `yaml:"value"`
	Desc  string `yaml:"desc"`
}

type Label struct {
	Name string `yaml:"name"`
	Desc string `yaml:"desc"`

	// if ManuallyEdit = true, this field can only be edited manually
	ManuallyEdit bool `yaml:"manually_edit"`

	// Fixed = true means thie label value's candidate is fixed.
	// for item like container IDs, `Fixed` will be false
	Fixed  bool          `yaml:"fixed"`
	Values []*LabelValue `yaml:"values"`
}

type Row struct {
	Name   string   `yaml:"name"`
	Type   string   `yaml:"type"`
	Unit   string   `yaml:"unit"`
	Help   string   `yaml:"help"`
	Labels []*Label `yaml:"labels"`
	Since  string   `yaml:"since"`
}

type Component struct {
	Prefix           string `yaml:"prefix"`
	Title            string `yaml:"title"`
	Desc             string `yaml:"desc"`
	Rows             []*Row `yaml:"metrics"`
	metricFamilyList []*dto.MetricFamily
}

type Document struct {
	Version    string       `yaml:"version"`
	Components []*Component `yaml:"components"`
}

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

	updateYamlCocument(oldObj, yamlObj)

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

func updateLabelValues(oldLabelValues, newLabelValues []*LabelValue) []*LabelValue {

	if len(oldLabelValues) == 0 {
		return newLabelValues
	}

	labelValues := make([]*LabelValue, 0)
	i := 0
	j := 0
	ok := (i < len(oldLabelValues) && j < len(newLabelValues))
	for ; ok; ok = (i < len(oldLabelValues) && j < len(newLabelValues)) {
		lvi := oldLabelValues[i]
		lvj := newLabelValues[j]

		c := strings.Compare(lvi.Value, lvj.Value)
		switch c {
		case -1:
			// only in old, may be deleted
			i++
		case 0:
			labelValues = append(labelValues, lvi)
			i++
			j++
		case 1:
			// not in old, should insert into old
			labelValues = append(labelValues, lvj)
			j++
		}
	}

	return labelValues

}

func updateLabels(oldLabels, newLabels []*Label) []*Label {

	sort.SliceStable(oldLabels, func(i, j int) bool {
		b := strings.Compare(oldLabels[i].Name, oldLabels[j].Name)
		return b < 0
	})

	sort.SliceStable(newLabels, func(i, j int) bool {
		b := strings.Compare(newLabels[i].Name, newLabels[j].Name)
		return b < 0
	})

	if len(oldLabels) == 0 {
		return newLabels
	}

	labels := make([]*Label, 0)
	i := 0
	j := 0
	ok := (i < len(oldLabels) && j < len(newLabels))
	for ; ok; ok = (i < len(oldLabels) && j < len(newLabels)) {
		li := oldLabels[i]
		lj := newLabels[j]

		sort.SliceStable(li.Values, func(i, j int) bool {
			b := strings.Compare(li.Values[i].Value, li.Values[j].Value)
			return b < 0
		})
		sort.SliceStable(lj.Values, func(i, j int) bool {
			b := strings.Compare(lj.Values[i].Value, lj.Values[j].Value)
			return b < 0
		})

		c := strings.Compare(li.Name, lj.Name)
		switch c {
		case -1:
			// only in old, may be deleted
			i++
		case 0:
			// the same label
			if !li.ManuallyEdit {
				// update label values
				li.Values = updateLabelValues(li.Values, lj.Values)
			}
			li.Fixed = lj.Fixed
			labels = append(labels, li)

			i++
			j++
		case 1:
			// not in old, should insert into old
			labels = append(labels, lj)
			j++
		}
	}

	return labels
}

func updateComponent(oldComponent, newComponent *Component) {
	rows := make([]*Row, 0)

	sort.SliceStable(oldComponent.Rows, func(i, j int) bool {
		b := strings.Compare(oldComponent.Rows[i].Name, oldComponent.Rows[j].Name)
		return b < 0
	})

	sort.SliceStable(newComponent.Rows, func(i, j int) bool {
		b := strings.Compare(newComponent.Rows[i].Name, newComponent.Rows[j].Name)
		return b < 0
	})

	if len(oldComponent.Rows) == 0 {
		oldComponent.Rows = newComponent.Rows
		return
	}

	i := 0
	j := 0
	ok := (i < len(oldComponent.Rows) && j < len(newComponent.Rows))

	for ; ok; ok = (i < len(oldComponent.Rows) && j < len(newComponent.Rows)) {
		ri := oldComponent.Rows[i]
		rj := newComponent.Rows[j]

		c := strings.Compare(ri.Name, rj.Name)
		switch c {
		case -1:
			// only in old, may be deleted
			i++
		case 0:
			// the same metrics
			ri.Help = rj.Help
			ri.Type = rj.Type
			ri.Labels = updateLabels(ri.Labels, rj.Labels)
			rows = append(rows, ri)
			i++
			j++
		case 1:
			// not in old, should insert into old
			rows = append(rows, rj)
			j++
		}
	}

	oldComponent.Rows = rows
}

// updateYamlCocument will update oldDoc useing newDoc
func updateYamlCocument(oldDoc, newDoc *Document) {

	sort.SliceStable(oldDoc.Components, func(i, j int) bool {
		b := strings.Compare(oldDoc.Components[i].Prefix, oldDoc.Components[j].Prefix)
		return b < 0
	})
	sort.SliceStable(newDoc.Components, func(i, j int) bool {
		b := strings.Compare(newDoc.Components[i].Prefix, newDoc.Components[j].Prefix)
		return b < 0
	})

	if len(oldDoc.Components) == 0 {
		oldDoc.Components = newDoc.Components
		return
	}
	components := make([]*Component, 0)

	i := 0
	j := 0
	ok := (i < len(oldDoc.Components) && j < len(newDoc.Components))

	for ; ok; ok = (i < len(oldDoc.Components) && j < len(newDoc.Components)) {
		ci := oldDoc.Components[i]
		cj := newDoc.Components[j]

		c := strings.Compare(ci.Prefix, cj.Prefix)
		switch c {
		case -1:
			// only in old, may be deleted
			i++
		case 0:
			// the same component
			updateComponent(ci, cj)
			components = append(components, ci)
			i++
			j++
		case 1:
			// not in old, should insert into old
			components = append(components, cj)
			j++
		}
	}

	oldDoc.Components = components
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
