package testlab

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strings"
)

/*
 * JUnit XML is the one thing every test framework agrees on. Parsing it here
 * means Applad never has to know what Jest, pytest, go test or flutter test
 * are — a suite says where its report lands, and the report says what happened.
 *
 * The format is a convention rather than a standard, so this reads the union
 * of what runners actually emit: a bare <testsuite>, a <testsuites> wrapper,
 * failures as elements or as attributes, and time in seconds as either a float
 * or an integer.
 */

// CaseStatus is the outcome of a single test case.
type CaseStatus string

const (
	CasePassed  CaseStatus = "passed"
	CaseFailed  CaseStatus = "failed"
	CaseSkipped CaseStatus = "skipped"
	CaseErrored CaseStatus = "errored"
)

// Case is one test case parsed from a report.
type Case struct {
	SuiteName      string
	Name           string
	Status         CaseStatus
	DurationMs     int64
	FailureMessage string
	FailureDetails string
}

// Summary counts the outcomes of a parsed report.
type Summary struct {
	Total      int
	Passed     int
	Failed     int
	Skipped    int
	DurationMs int64
}

type xmlSuites struct {
	XMLName xml.Name   `xml:"testsuites"`
	Suites  []xmlSuite `xml:"testsuite"`
	Cases   []xmlCase  `xml:"testcase"` // some runners nest cases directly
}

type xmlSuite struct {
	XMLName xml.Name   `xml:"testsuite"`
	Name    string     `xml:"name,attr"`
	Time    string     `xml:"time,attr"`
	Suites  []xmlSuite `xml:"testsuite"` // nested suites are legal
	Cases   []xmlCase  `xml:"testcase"`
}

type xmlCase struct {
	Name      string      `xml:"name,attr"`
	ClassName string      `xml:"classname,attr"`
	Time      string      `xml:"time,attr"`
	Failures  []xmlDetail `xml:"failure"`
	Errors    []xmlDetail `xml:"error"`
	Skipped   *xmlDetail  `xml:"skipped"`
}

type xmlDetail struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// ParseJUnit reads a JUnit XML report into cases.
func ParseJUnit(r io.Reader) ([]Case, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("testlab: read report: %w", err)
	}
	return parseJUnitBytes(data)
}

func parseJUnitBytes(data []byte) ([]Case, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("testlab: report is empty")
	}

	// A <testsuites> wrapper is the common case; a bare <testsuite> is equally
	// common, so try both rather than requiring one.
	var wrapper xmlSuites
	if err := xml.Unmarshal(data, &wrapper); err == nil && wrapper.XMLName.Local == "testsuites" {
		var out []Case
		for _, s := range wrapper.Suites {
			out = append(out, flatten(s, "")...)
		}
		for _, c := range wrapper.Cases {
			out = append(out, convert(c, ""))
		}
		return out, nil
	}

	var single xmlSuite
	if err := xml.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("testlab: report is not JUnit XML: %w", err)
	}
	if single.XMLName.Local != "testsuite" {
		return nil, fmt.Errorf("testlab: report is not JUnit XML: unexpected root <%s>", single.XMLName.Local)
	}
	return flatten(single, ""), nil
}

// flatten walks nested suites into a single list, keeping the suite path as
// the case's suite name so nesting is visible without being structural.
func flatten(s xmlSuite, prefix string) []Case {
	name := s.Name
	if prefix != "" && name != "" {
		name = prefix + " › " + name
	} else if name == "" {
		name = prefix
	}

	var out []Case
	for _, c := range s.Cases {
		out = append(out, convert(c, name))
	}
	for _, nested := range s.Suites {
		out = append(out, flatten(nested, name)...)
	}
	return out
}

func convert(c xmlCase, suiteName string) Case {
	// classname is more specific than the suite when both are present, and is
	// what most runners put the meaningful grouping in.
	suite := suiteName
	if c.ClassName != "" {
		suite = c.ClassName
	}
	if suite == "" {
		suite = "(root)"
	}

	out := Case{
		SuiteName:  suite,
		Name:       strings.TrimSpace(c.Name),
		Status:     CasePassed,
		DurationMs: secondsToMs(c.Time),
	}

	switch {
	case len(c.Errors) > 0:
		// An error is the framework failing to run the test, not the test
		// disagreeing with the code. Keeping them apart matters when reading
		// a red run.
		out.Status = CaseErrored
		out.FailureMessage, out.FailureDetails = detail(c.Errors[0])
	case len(c.Failures) > 0:
		out.Status = CaseFailed
		out.FailureMessage, out.FailureDetails = detail(c.Failures[0])
	case c.Skipped != nil:
		out.Status = CaseSkipped
		out.FailureMessage, _ = detail(*c.Skipped)
	}
	return out
}

func detail(d xmlDetail) (message, details string) {
	message = strings.TrimSpace(d.Message)
	details = strings.TrimSpace(d.Body)
	if message == "" {
		// Some runners put everything in the body and leave the attribute off.
		if line, _, ok := strings.Cut(details, "\n"); ok {
			message = strings.TrimSpace(line)
		} else {
			message = details
		}
	}
	if message == "" && d.Type != "" {
		message = d.Type
	}
	return message, details
}

// secondsToMs converts JUnit's seconds-as-string into milliseconds. A missing
// or unparseable time is not an error: plenty of runners omit it.
func secondsToMs(v string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%g", &f); err != nil || math.IsNaN(f) || f < 0 {
		return 0
	}
	return int64(f * 1000)
}

// Summarise counts outcomes for a run.
func Summarise(cases []Case) Summary {
	s := Summary{Total: len(cases)}
	for _, c := range cases {
		s.DurationMs += c.DurationMs
		switch c.Status {
		case CasePassed:
			s.Passed++
		case CaseSkipped:
			s.Skipped++
		default:
			// An errored case is a failing case as far as the run's verdict
			// goes; the distinction is kept on the case itself.
			s.Failed++
		}
	}
	return s
}
