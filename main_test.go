package main

import (
	"bytes"
	"strings"
	"testing"
)

func Test_writeMinifiedDockerfile(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `FROM scratch`,
			expected: `FROM scratch`,
		},
		{
			input: `
# normal comment
FROM scratch
`,
			expected: `FROM scratch`,
		},
		{
			input: `
# syntax = docker/dockerfile:experimental
FROM scratch
`,
			expected: `
# syntax = docker/dockerfile:experimental
FROM scratch`,
		},
		{
			input: `
FROM scratch
COPY ./a ./foo/
COPY ./b ./foo/
`,
			expected: `
FROM scratch
COPY ./a ./b ./foo/
`,
		},
		{
			input: `
FROM scratch
COPY ./a ./foo/
COPY ./b ./foo
`,
			expected: `
FROM scratch
COPY ./a ./foo/
COPY ./b ./foo
`,
		},
		{
			input: `
FROM scratch
COPY ./a ./foo/
COPY ./b ./foo/
COPY ./c ./foo/
`,
			expected: `
FROM scratch
COPY ./a ./b ./c ./foo/
`,
		},
		{
			input: `
FROM scratch
COPY --from=bar ./a ./foo/
COPY --from=bar ./b ./foo/
COPY --from=foo ./c ./foo/
`,
			expected: `
FROM scratch
COPY --from=bar ./a ./b ./foo/
COPY --from=foo ./c ./foo/
`,
		},
		{
			input: `
FROM scratch
RUN echo "foo"
RUN echo "bar"
`,
			expected: `
FROM scratch
RUN echo "foo" && echo "bar"
`,
		},
		{
			input: `
FROM scratch
ENV A "foo"
ENV B "bar"
`,
			expected: `
FROM scratch
ENV A="foo" B="bar"
`,
		},
		{
			input: `
FROM scratch
ENV A "foo"
ENV B "${A}"
ENV C "bar"
`,
			expected: `
FROM scratch
ENV A="foo"
ENV B="${A}" C="bar"
`,
		},
		{
			input: `
FROM scratch
LABEL foo=bar
LABEL baz=foobar
`,
			expected: `
FROM scratch
LABEL foo=bar baz=foobar
`,
		},
	}

	for i, test := range tests {
		var b bytes.Buffer
		if err := writeMinifiedDockerfile(&b, []byte(strings.TrimSpace(test.input))); err != nil {
			t.Errorf("failed to minify on testcase %d: %v", i, err)
			continue
		}
		got := b.String()
		if strings.TrimSpace(got) != strings.TrimSpace(test.expected) {
			t.Errorf("unexpected minified Dockerfile on testcase %d. expected: %v, but got: %v", i, test.expected, got)
		}
	}
}
