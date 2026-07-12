package main

// @sk-task arch-issues#T5.1: route table for code generator (AC-005, AC-006)
type routeEntry struct {
	Route         string // route constant name (e.g. "routeBasic")
	Name          string // human-readable name
	Pattern       string // "mk" for Pipeline method ref, "wrap" for handler function
	Retrieve      string // handler name or Pipeline method name
	Answer        string
	Cite          string
	InlineCite    string
	Stream        string
	StreamSources string
	StreamCite    string
}

// routeTable defines all routes and their handler mappings for every output type.
// Routes with Pattern="mk" use mk*(pipeline.Method) — the Target field stores the method name.
// Routes with Pattern="wrap" use wrap*(handlerFunc) — the Target field stores the function name.
// Order determines priority in the generated maps (must match pickRoute priority).
var routeTable = []routeEntry{
	{
		Route: "routeBasic", Name: "basic", Pattern: "mk",
		Retrieve: "Query", Answer: "Answer",
		Cite: "AnswerWithCitations", InlineCite: "AnswerWithInlineCitations",
		Stream: "AnswerStream", StreamSources: "AnswerStreamWithSources",
		StreamCite: "AnswerStreamWithInlineCitations",
	},
	{
		Route: "routeRewriter", Name: "rewriter", Pattern: "wrap",
		Retrieve: "rewriterRetrieve", Answer: "rewriterAnswer",
		Cite: "rewriterCite", InlineCite: "rewriterInlineCite",
		Stream: "rewriterStream", StreamSources: "rewriterStreamSources",
		StreamCite: "rewriterStreamCite",
	},
	{
		Route: "routeSubDecompose", Name: "subDecompose", Pattern: "wrap",
		Retrieve: "subDecomposeRetrieve", Answer: "subDecomposeAnswer",
		Cite: "subDecomposeCite", InlineCite: "subDecomposeInlineCite",
		Stream: "subDecomposeStream", StreamSources: "subDecomposeStreamSources",
		StreamCite: "subDecomposeStreamCite",
	},
	{
		Route: "routeHyDE", Name: "hyDE", Pattern: "mk",
		Retrieve: "QueryHyDE", Answer: "AnswerHyDE",
		Cite: "AnswerHyDEWithCitations", InlineCite: "AnswerHyDEWithInlineCitations",
		Stream: "AnswerHyDEStream", StreamSources: "AnswerHyDEStreamWithSources",
		StreamCite: "AnswerHyDEStreamWithInlineCitations",
	},
	{
		Route: "routeMultiQuery", Name: "multiQuery", Pattern: "wrap",
		Retrieve: "multiQueryRetrieve", Answer: "multiQueryAnswer",
		Cite: "multiQueryCite", InlineCite: "multiQueryInlineCite",
		Stream: "multiQueryStream", StreamSources: "multiQueryStreamSources",
		StreamCite: "multiQueryStreamCite",
	},
	{
		Route: "routeTools", Name: "tools", Pattern: "wrap",
		Retrieve: "toolsRetrieve", Answer: "toolsAnswer",
		Cite: "toolsCite", InlineCite: "toolsInlineCite",
		Stream: "toolsStream", StreamSources: "toolsStreamSources",
		StreamCite: "toolsStreamCite",
	},
	{
		Route: "routeHybrid", Name: "hybrid", Pattern: "wrap",
		Retrieve: "hybridRetrieve", Answer: "hybridAnswer",
		Cite: "hybridCite", InlineCite: "hybridInlineCite",
		Stream: "hybridStream", StreamSources: "hybridStreamSources",
		StreamCite: "hybridStreamCite",
	},
	{
		Route: "routeParentIDs", Name: "parentIDs", Pattern: "wrap",
		Retrieve: "parentIDsRetrieve", Answer: "parentIDsAnswer",
		Cite: "parentIDsCite", InlineCite: "parentIDsInlineCite",
		Stream: "parentIDsStream", StreamSources: "parentIDsStreamSources",
		StreamCite: "parentIDsStreamCite",
	},
	{
		Route: "routeFilter", Name: "filter", Pattern: "wrap",
		Retrieve: "filterRetrieve", Answer: "filterAnswer",
		Cite: "filterCite", InlineCite: "filterInlineCite",
		Stream: "filterStream", StreamSources: "filterStreamSources",
		StreamCite: "filterStreamCite",
	},
}

// outputColumns defines the 7 output types with their wrapper factory and result type.
type outputColumn struct {
	Name       string // "Retrieve", "Answer", etc.
	Wrapper    string // "Retrieve", "Answer", etc. (for mkWrapper/wrapWrapper naming)
	ResultType string // Go result struct name
}

var outputColumns = []outputColumn{
	{Name: "Retrieve", Wrapper: "Retrieve", ResultType: "rRetrieve"},
	{Name: "Answer", Wrapper: "Answer", ResultType: "rAnswer"},
	{Name: "Cite", Wrapper: "Cite", ResultType: "rCite"},
	{Name: "InlineCite", Wrapper: "InlineCite", ResultType: "rInlineCite"},
	{Name: "Stream", Wrapper: "Stream", ResultType: "rStream"},
	{Name: "StreamSources", Wrapper: "StreamSources", ResultType: "rStreamSources"},
	{Name: "StreamCite", Wrapper: "StreamCite", ResultType: "rStreamCite"},
}
