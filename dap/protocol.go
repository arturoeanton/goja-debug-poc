package main

// DAP Protocol types based on Debug Adapter Protocol specification

type Message struct {
	Seq  int    `json:"seq"`
	Type string `json:"type"` // "request", "response", "event"
}

type Request struct {
	Message
	Command   string      `json:"command"`
	Arguments interface{} `json:"arguments,omitempty"`
}

type Response struct {
	Message
	RequestSeq   int         `json:"request_seq"`
	Success      bool        `json:"success"`
	Command      string      `json:"command"`
	ErrorMessage string      `json:"message,omitempty"`
	Body         interface{} `json:"body,omitempty"`
}

type Event struct {
	Message
	Event string      `json:"event"`
	Body  interface{} `json:"body,omitempty"`
}

// Initialize request/response
type InitializeRequestArguments struct {
	ClientID                     string `json:"clientID,omitempty"`
	ClientName                   string `json:"clientName,omitempty"`
	AdapterID                    string `json:"adapterID"`
	Locale                       string `json:"locale,omitempty"`
	LinesStartAt1                bool   `json:"linesStartAt1"`
	ColumnsStartAt1              bool   `json:"columnsStartAt1"`
	PathFormat                   string `json:"pathFormat,omitempty"`
	SupportsVariableType         bool   `json:"supportsVariableType,omitempty"`
	SupportsVariablePaging       bool   `json:"supportsVariablePaging,omitempty"`
	SupportsRunInTerminalRequest bool   `json:"supportsRunInTerminalRequest,omitempty"`
}

type Capabilities struct {
	SupportsConfigurationDoneRequest bool `json:"supportsConfigurationDoneRequest"`
	SupportsFunctionBreakpoints      bool `json:"supportsFunctionBreakpoints"`
	SupportsConditionalBreakpoints   bool `json:"supportsConditionalBreakpoints"`
	SupportsEvaluateForHovers        bool `json:"supportsEvaluateForHovers"`
	SupportsStepBack                 bool `json:"supportsStepBack"`
	SupportsSetVariable              bool `json:"supportsSetVariable"`
	SupportsRestartFrame             bool `json:"supportsRestartFrame"`
	SupportsStepInTargetsRequest     bool `json:"supportsStepInTargetsRequest"`
	SupportsDelayedStackTraceLoading bool `json:"supportsDelayedStackTraceLoading"`
	SupportsTerminateRequest         bool `json:"supportsTerminateRequest"`
}

// Launch request
type LaunchRequestArguments struct {
	NoDebug     bool     `json:"noDebug,omitempty"`
	Program     string   `json:"program"`
	Args        []string `json:"args,omitempty"`
	StopOnEntry bool     `json:"stopOnEntry,omitempty"`
}

// Breakpoint types
type Source struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
}

type SourceBreakpoint struct {
	Line      int    `json:"line"`
	Column    int    `json:"column,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type Breakpoint struct {
	ID        int    `json:"id"`
	Verified  bool   `json:"verified"`
	Message   string `json:"message,omitempty"`
	Source    Source `json:"source,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
}

type SetBreakpointsArguments struct {
	Source      Source             `json:"source"`
	Breakpoints []SourceBreakpoint `json:"breakpoints,omitempty"`
}

type SetBreakpointsResponseBody struct {
	Breakpoints []Breakpoint `json:"breakpoints"`
}

// Thread types
type Thread struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ThreadsResponseBody struct {
	Threads []Thread `json:"threads"`
}

// Stack trace types
type StackFrame struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Source    Source `json:"source,omitempty"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
}

type StackTraceArguments struct {
	ThreadID   int `json:"threadId"`
	StartFrame int `json:"startFrame,omitempty"`
	Levels     int `json:"levels,omitempty"`
}

type StackTraceResponseBody struct {
	StackFrames []StackFrame `json:"stackFrames"`
	TotalFrames int          `json:"totalFrames,omitempty"`
}

// Scope types
type Scope struct {
	Name               string `json:"name"`
	VariablesReference int    `json:"variablesReference"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
	Expensive          bool   `json:"expensive"`
}

type ScopesArguments struct {
	FrameID int `json:"frameId"`
}

type ScopesResponseBody struct {
	Scopes []Scope `json:"scopes"`
}

// Variable types
type Variable struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	Type               string `json:"type,omitempty"`
	VariablesReference int    `json:"variablesReference"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
}

type VariablesArguments struct {
	VariablesReference int `json:"variablesReference"`
	Start              int `json:"start,omitempty"`
	Count              int `json:"count,omitempty"`
}

type VariablesResponseBody struct {
	Variables []Variable `json:"variables"`
}

// Evaluate types
type EvaluateArguments struct {
	Expression string `json:"expression"`
	FrameID    int    `json:"frameId,omitempty"`
	Context    string `json:"context,omitempty"`
}

type EvaluateResponseBody struct {
	Result             string `json:"result"`
	Type               string `json:"type,omitempty"`
	VariablesReference int    `json:"variablesReference"`
}

// Continue/Step types
type ContinueArguments struct {
	ThreadID int `json:"threadId"`
}

type ContinueResponseBody struct {
	AllThreadsContinued bool `json:"allThreadsContinued"`
}

type NextArguments struct {
	ThreadID int `json:"threadId"`
}

type StepInArguments struct {
	ThreadID int `json:"threadId"`
}

type StepOutArguments struct {
	ThreadID int `json:"threadId"`
}

// Pause types
type PauseArguments struct {
	ThreadID int `json:"threadId"`
}

// Events
type StoppedEventBody struct {
	Reason            string `json:"reason"`
	Description       string `json:"description,omitempty"`
	ThreadID          int    `json:"threadId,omitempty"`
	AllThreadsStopped bool   `json:"allThreadsStopped"`
}

type InitializedEvent struct {
}

type TerminatedEventBody struct {
	Restart interface{} `json:"restart,omitempty"`
}

type ExitedEventBody struct {
	ExitCode int `json:"exitCode"`
}
