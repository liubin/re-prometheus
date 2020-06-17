package main

import (
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
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

func (oldComponent *Component) updateComponent(newComponent *Component) {
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
func (oldDoc *Document) updateYamlCocument(newDoc *Document) {

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
			ci.updateComponent(cj)
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
