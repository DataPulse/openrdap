// OpenRDAP
// Copyright 2017 Tom Harwood
// MIT License, see the LICENSE file.

package rdap

import (
	"reflect"
	"testing"

	"github.com/DataPulse/openrdap/test"
)

func TestVCardErrors(t *testing.T) {
	filenames := []string{
		"jcard/error_invalid_json.json",
		"jcard/error_bad_top_type.json",
		"jcard/error_bad_vcard_label.json",
		"jcard/error_bad_properties_array.json",
		"jcard/error_bad_property_size.json",
		"jcard/error_bad_property_name.json",
		"jcard/error_bad_property_type.json",
		"jcard/error_bad_property_parameters.json",
		"jcard/error_bad_property_parameters_2.json",
		"jcard/error_bad_property_nest_depth.json",
	}

	for _, filename := range filenames {
		j, err := NewVCard(test.LoadFile(filename))

		if j != nil || err == nil {
			t.Errorf("jCard with error unexpectedly parsed %s %v %s\n", filename, j, err)
		}
	}
}

func TestVCardIgnoreInvalidProperties(t *testing.T) {
	json := test.LoadFile("jcard/error_invalid_properties.json")

	j1, err1 := NewVCardWithOptions(json, VCardOptions{IgnoreInvalidProperties: true})
	if j1 == nil || len(j1.Properties) != 4 || err1 != nil {
		t.Errorf("jCard with ignored errors not parsed correctly\n")
	}

	j2, err2 := NewVCardWithOptions(json, VCardOptions{IgnoreInvalidProperties: false})
	if j2 != nil || err2 == nil {
		t.Errorf("jCard with errors unexpectedly parsed\n")
	}
}

func TestVCardExample(t *testing.T) {
	j, err := NewVCard(test.LoadFile("jcard/example.json"))
	if j == nil || err != nil {
		t.Errorf("jCard parse failed %v %s\n", j, err)
	}

	numProperties := 17
	if len(j.Properties) != numProperties {
		t.Errorf("Got %d properties expected %d", len(j.Properties), numProperties)
	}

	expectedVersion := &VCardProperty{
		Name:       "version",
		Parameters: make(map[string][]string),
		Type:       "text",
		Value:      "4.0",
	}

	if !reflect.DeepEqual(j.Get("version")[0], expectedVersion) {
		t.Errorf("version field incorrect")
	}

	expectedN := &VCardProperty{
		Name:       "n",
		Parameters: make(map[string][]string),
		Type:       "text",
		Value:      []interface{}{"Perreault", "Simon", "", "", []interface{}{"ing. jr", "M.Sc."}},
	}

	// One entry per "n" component (RFC 6350: family, given, additional,
	// prefixes, suffixes). The suffix component is itself an array and is
	// joined, not spread, so the five positions stay put.
	expectedFlatN := []string{
		"Perreault",
		"Simon",
		"",
		"",
		"ing. jr, M.Sc.",
	}

	if !reflect.DeepEqual(j.Get("n")[0], expectedN) {
		t.Errorf("n field incorrect")
	}

	if !reflect.DeepEqual(j.Get("n")[0].Values(), expectedFlatN) {
		t.Errorf("n flat value incorrect")
	}

	expectedTel0 := &VCardProperty{
		Name:       "tel",
		Parameters: map[string][]string{"type": {"work", "voice"}, "pref": {"1"}},
		Type:       "uri",
		Value:      "tel:+1-418-656-9254;ext=102",
	}

	if !reflect.DeepEqual(j.Get("tel")[0], expectedTel0) {
		t.Errorf("tel[0] field incorrect")
	}
}

func TestVCardMixedDatatypes(t *testing.T) {
	j, err := NewVCard(test.LoadFile("jcard/mixed.json"))
	if j == nil || err != nil {
		t.Errorf("jCard parse failed %v %s\n", j, err)
	}

	expectedMixed := &VCardProperty{
		Name:       "mixed",
		Parameters: make(map[string][]string),
		Type:       "text",
		Value:      []interface{}{"abc", true, float64(42), nil, []interface{}{"def", false, float64(43)}},
	}

	// The nested array is one component, joined with ", ". Its members are
	// not all strings, which used to panic in the join.
	expectedFlatMixed := []string{
		"abc",
		"true",
		"42",
		"",
		"def, false, 43",
	}

	if !reflect.DeepEqual(j.Get("mixed")[0], expectedMixed) {
		t.Errorf("mixed field incorrect")
	}

	flattened := j.Get("mixed")[0].Values()
	if !reflect.DeepEqual(flattened, expectedFlatMixed) {
		t.Errorf("mixed flat value incorrect %v", flattened)
	}
}

// An "adr" component may be an array (RFC 7095 §3.3.1.3: several street
// lines). Values() must keep one entry per component so the positional
// accessors keep pointing at the right field; this is why nested arrays are
// joined rather than spread.
func TestVCardValuesKeepsAddressPositions(t *testing.T) {
	adr := &VCardProperty{
		Name:       "adr",
		Parameters: make(map[string][]string),
		Type:       "text",
		Value: []interface{}{
			"",
			"Suite D2-630",
			[]interface{}{"2875 Laurier", "Building B"},
			"Quebec",
			"QC",
			"G1V 2M2",
			"Canada",
		},
	}
	expected := []string{"", "Suite D2-630", "2875 Laurier, Building B", "Quebec", "QC", "G1V 2M2", "Canada"}
	if got := adr.Values(); !reflect.DeepEqual(got, expected) {
		t.Errorf("Values() = %v, want %v", got, expected)
	}

	v := &VCard{Properties: []*VCardProperty{adr}}
	if got := v.StreetAddress(); got != "2875 Laurier, Building B" {
		t.Errorf("StreetAddress() = %q, want the joined street lines", got)
	}
	if got := v.Country(); got != "Canada" {
		t.Errorf("Country() = %q, want %q: an array component shifted the positions", got, "Canada")
	}
}

// Values() on scalar and deeply nested values: a scalar is one entry, and
// nesting below the first level flattens fully into the component's entry
// whatever the member types.
func TestVCardValuesScalarsAndDeepNesting(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
		want  []string
	}{
		{"string", "4.0", []string{"4.0"}},
		{"bool", true, []string{"true"}},
		{"number", float64(42), []string{"42"}},
		{"nil", nil, []string{""}},
		{"empty array", []interface{}{}, []string{}},
		{"deep", []interface{}{"a", []interface{}{"b", []interface{}{"c", float64(1), nil, false}}}, []string{"a", "b, c, 1, , false"}},
	}
	for _, c := range cases {
		p := &VCardProperty{Name: c.name, Type: "text", Value: c.value}
		if got := p.Values(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Values() = %#v, want %#v", c.name, got, c.want)
		}
	}
}

func TestVCardQuickAccessors(t *testing.T) {
	j, err := NewVCard(test.LoadFile("jcard/example.json"))
	if j == nil || err != nil {
		t.Errorf("jCard parse failed %v %s\n", j, err)
	}

	got := []string{
		j.Name(),
		j.POBox(),
		j.ExtendedAddress(),
		j.StreetAddress(),
		j.Locality(),
		j.Region(),
		j.PostalCode(),
		j.Country(),
		j.Tel(),
		j.Fax(),
		j.Email(),
		j.Org(),
	}

	expected := []string{
		"Simon Perreault",
		"",
		"Suite D2-630",
		"2875 Laurier",
		"Quebec",
		"QC",
		"G1V 2M2",
		"Canada",
		"+1-418-656-9254;ext=102",
		"",
		"simon.perreault@viagenie.ca",
		"Viagenie",
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got %v expected %v\n", got, expected)
	}
}
