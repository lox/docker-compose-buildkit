package compose

import (
	"reflect"
	"testing"
)

const basicConfigV3 = `---
version: '3'
services:
  webapp:
    build: .
`

const nestedConfigV3 = `---
version: '3'
services:
  webapp:
    build:
      context: ..
      dockerfile: Dockerfile.llamas
      args:
       - BLAH=true
      labels:
        llamas: always
        alpacas: sometimes
`

func TestParseBasicConfigString(t *testing.T) {
	c, err := ParseString(basicConfigV3)
	if err != nil {
		t.Fatal(err)
	}

	if l := len(c.Services); l != 1 {
		t.Fatalf("Expected 1 service, got %d", l)
	}

	service, ok := c.Services["webapp"]
	if !ok {
		t.Fatalf("Expected a service of webapp")
	}

	if service.Build.Context != "." {
		t.Fatalf("Expected webapp.build.context to be '.', got %q", service.Build.Context)
	}
}

func TestParseNestedConfigString(t *testing.T) {
	c, err := ParseString(nestedConfigV3)
	if err != nil {
		t.Fatal(err)
	}

	if l := len(c.Services); l != 1 {
		t.Fatalf("Expected 1 service, got %d", l)
	}

	if l := len(c.Services); l != 1 {
		t.Fatalf("Expected 1 service, got %d", l)
	}

	service, ok := c.Services["webapp"]
	if !ok {
		t.Fatalf("Expected a service of webapp")
	}

	if service.Build.Context != ".." {
		t.Fatalf("Expected webapp.build.context to be '..', got %q", service.Build.Context)
	}

	if !reflect.DeepEqual(service.Build.Args, mapOrSlice{`BLAH=true`}) {
		t.Fatalf("Bad args: %#v", service.Build.Args)
	}

	if !reflect.DeepEqual(service.Build.Labels, mapOrSlice{`llamas=always`, `alpacas=sometimes`}) {
		t.Fatalf("Bad labels: %#v", service.Build.Labels)
	}

}
