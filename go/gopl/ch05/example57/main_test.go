package main

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestOutline(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "void element uses self-closing shorthand",
			input: `<img src="x.png">`,
			want: "<html>\n" +
				"  <head/>\n" +
				"  <body>\n" +
				"    <img src='x.png'/>\n" +
				"  </body>\n" +
				"</html>\n",
		},
		{
			name:  "comment, text and attributes are rendered",
			input: `<body><!--hi--><p class="a">Text</p></body>`,
			want: "<html>\n" +
				"  <head/>\n" +
				"  <body>\n" +
				"    <!--hi-->\n" +
				"    <p class='a'>\n" +
				"      Text\n" +
				"    </p>\n" +
				"  </body>\n" +
				"</html>\n",
		},
		{
			name:  "nested siblings keep open and close tags aligned",
			input: `<div><span>a</span><span>b</span></div>`,
			want: "<html>\n" +
				"  <head/>\n" +
				"  <body>\n" +
				"    <div>\n" +
				"      <span>\n" +
				"        a\n" +
				"      </span>\n" +
				"      <span>\n" +
				"        b\n" +
				"      </span>\n" +
				"    </div>\n" +
				"  </body>\n" +
				"</html>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depth = 0
			var buf bytes.Buffer
			out = &buf

			doc, err := html.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("html.Parse: %v", err)
			}
			forEachNode(doc, startElement, endElement)

			if got := buf.String(); got != tt.want {
				t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
