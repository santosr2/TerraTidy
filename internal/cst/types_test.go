package cst

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time assertions that the four sum-type members satisfy BodyItem.
// If a new BodyItem is added, add it here so the seal stays exhaustive.
var (
	_ BodyItem = (*Attribute)(nil)
	_ BodyItem = (*Block)(nil)
	_ BodyItem = (*BlankLine)(nil)
	_ BodyItem = (*StandaloneComment)(nil)
)

func TestRawBytes_ReturnsStoredBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		item BodyItem
		want []byte
	}{
		{
			name: "Attribute populated",
			item: &Attribute{raw: []byte("name = \"value\"\n")},
			want: []byte("name = \"value\"\n"),
		},
		{
			name: "Block populated",
			item: &Block{raw: []byte("resource \"x\" \"y\" {}\n")},
			want: []byte("resource \"x\" \"y\" {}\n"),
		},
		{
			name: "BlankLine populated",
			item: &BlankLine{Count: 2, raw: []byte("\n\n")},
			want: []byte("\n\n"),
		},
		{
			name: "StandaloneComment populated",
			item: &StandaloneComment{raw: []byte("# header\n")},
			want: []byte("# header\n"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.item.RawBytes())
		})
	}
}

func TestRawBytes_NilWhenMutated(t *testing.T) {
	t.Parallel()

	// Zero-value items have nil raw, simulating the "mutated since Build"
	// signal that Serialize uses to switch to the regeneration path.
	tests := []struct {
		name string
		item BodyItem
	}{
		{"Attribute zero-value", &Attribute{}},
		{"Block zero-value", &Block{}},
		{"BlankLine zero-value", &BlankLine{}},
		{"StandaloneComment zero-value", &StandaloneComment{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, tc.item.RawBytes())
		})
	}
}

// TestAncillaryTypes_FieldsArePresent pins the shape of Comment, Label,
// and the CommentStyle constants so a silent drift (renaming a field,
// dropping a constant) is caught here rather than in the Build tests.
func TestAncillaryTypes_FieldsArePresent(t *testing.T) {
	t.Parallel()

	c := Comment{Style: CommentHash, Text: "# x", Raw: []byte("# x")}
	assert.Equal(t, CommentHash, c.Style)
	assert.Equal(t, "# x", c.Text)
	assert.Equal(t, []byte("# x"), c.Raw)

	l := Label{Text: "foo", Raw: []byte(`"foo"`)}
	assert.Equal(t, "foo", l.Text)
	assert.Equal(t, []byte(`"foo"`), l.Raw)

	assert.NotEqual(t, CommentHash, CommentSlash)
	assert.NotEqual(t, CommentSlash, CommentBlock)

	f := File{Source: []byte("x = 1\n"), Body: &Body{OpenByte: -1, CloseByte: -1}}
	assert.Equal(t, []byte("x = 1\n"), f.Source)
	assert.Equal(t, -1, f.Body.OpenByte)
	assert.Equal(t, -1, f.Body.CloseByte)
}
