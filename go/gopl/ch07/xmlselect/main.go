// Xmlselect prints the text of selected elements of an XML document.
package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {

	dec := xml.NewDecoder(getXML())
	var stack []string // stack of element names
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "xmlselect: %v\n", err)
			os.Exit(1)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			stack = append(stack, tok.Name.Local) // push
		case xml.EndElement:
			stack = stack[:len(stack)-1] // pop
		case xml.CharData:
			if containsAll(stack, os.Args[1:]) {
				fmt.Printf("%s: %s\n", strings.Join(stack, " "), tok)
			}
		}
	}
}

// containsAll reports whether x contains the elements of y, in order.
func containsAll(x, y []string) bool {
	for len(y) <= len(x) {
		if len(y) == 0 {
			return true
		}
		if x[0] == y[0] {
			y = y[1:]
		}
		x = x[1:]
	}
	return false
}

func getXML() io.Reader {
	return strings.NewReader(`<html>
  <head>
    <title>XML 1.1 (Second Edition)</title>
  </head>
  <body>
    <div class="book">
      <h2>Extensible Markup Language (XML) 1.1</h2>
      <div class="chapter">
        <h2>1 Introduction</h2>
        <p>Extensible Markup Language, abbreviated XML, describes a class of
        data objects called XML documents.</p>
        <div class="section">
          <h2>1.1 Origin and Goals</h2>
          <p>XML was developed by an XML Working Group formed under the
          auspices of the World Wide Web Consortium in 1996.</p>
        </div>
        <div class="section">
          <h2>1.2 Terminology</h2>
          <p>The terminology used to describe XML documents is defined in the
          body of this specification.</p>
        </div>
      </div>
      <div class="chapter">
        <h2>2 Documents</h2>
        <div class="section">
          <h2>2.1 Well-Formed XML Documents</h2>
          <p>A textual object is a well-formed XML document if it matches the
          production labeled document.</p>
        </div>
      </div>
    </div>
  </body>
</html>
`)
}
