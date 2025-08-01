package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

type DebugAdapter struct {
	reader   *bufio.Reader
	writer   io.Writer
	seq      int
	seqMutex sync.Mutex

	// Goja runtime
	vm          *goja.Runtime
	debugger    *goja.Debugger
	program     string
	sourceCode  string
	sourceLines []string

	// Debug state
	running     bool
	terminated  bool
	breakpoints map[string][]int // filename -> line numbers
	bpIDCounter int
	bpMap       map[int]*Breakpoint // breakpoint ID -> breakpoint

	// Thread simulation (goja is single-threaded)
	threadID int

	// Variable references
	varRefCounter int
	varRefMap     map[int]interface{} // reference -> variable data

	// Stack frames
	currentFrameID int
	frameMap       map[int]*goja.StackFrame

	// Synchronization
	debugStateMutex sync.Mutex
	waitingForCmd   bool
	nextCommand     goja.DebugCommand
	commandReady    chan struct{}
}

func NewDebugAdapter(reader io.Reader, writer io.Writer) *DebugAdapter {
	return &DebugAdapter{
		reader:       bufio.NewReader(reader),
		writer:       writer,
		seq:          1,
		breakpoints:  make(map[string][]int),
		bpMap:        make(map[int]*Breakpoint),
		varRefMap:    make(map[int]interface{}),
		frameMap:     make(map[int]*goja.StackFrame),
		threadID:     1,
		commandReady: make(chan struct{}),
		nextCommand:  goja.DebugContinue,
	}
}

func (da *DebugAdapter) nextSeq() int {
	da.seqMutex.Lock()
	defer da.seqMutex.Unlock()
	seq := da.seq
	da.seq++
	return seq
}

func (da *DebugAdapter) sendResponse(requestSeq int, command string, success bool, body interface{}) {
	response := Response{
		Message: Message{
			Seq:  da.nextSeq(),
			Type: "response",
		},
		RequestSeq: requestSeq,
		Success:    success,
		Command:    command,
		Body:       body,
	}

	if !success && body == nil {
		response.ErrorMessage = "Unknown error"
	}

	log.Printf("<== Sending response: %s (success=%v, req_seq=%d)", command, success, requestSeq)
	da.sendMessage(response)
}

func (da *DebugAdapter) sendEvent(event string, body interface{}) {
	evt := Event{
		Message: Message{
			Seq:  da.nextSeq(),
			Type: "event",
		},
		Event: event,
		Body:  body,
	}

	// Log important events
	switch event {
	case "stopped":
		log.Printf("<== EVENT: stopped (reason: %v)", body)
	case "initialized", "terminated", "exited":
		log.Printf("<== EVENT: %s", event)
	case "output":
		// Don't log output events as they're too verbose
	default:
		log.Printf("<== EVENT: %s", event)
	}

	da.sendMessage(evt)
}

func (da *DebugAdapter) sendMessage(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	// DAP uses Content-Length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	da.writer.Write([]byte(header))
	da.writer.Write(data)
}

func (da *DebugAdapter) readMessage() (*Request, error) {
	// Read Content-Length header
	headerLine, err := da.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(headerLine, "Content-Length:") {
		return nil, fmt.Errorf("invalid header: %s", headerLine)
	}

	lengthStr := strings.TrimSpace(strings.TrimPrefix(headerLine, "Content-Length:"))
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, err
	}

	// Read empty line
	da.reader.ReadString('\n')

	// Read message body
	body := make([]byte, length)
	_, err = io.ReadFull(da.reader, body)
	if err != nil {
		return nil, err
	}

	var req Request
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}

	return &req, nil
}

func (da *DebugAdapter) Run() {
	for {
		req, err := da.readMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading message: %v", err)
			continue
		}

		da.handleRequest(req)

		if da.terminated {
			break
		}
	}
}

func (da *DebugAdapter) handleRequest(req *Request) {
	log.Printf("==> Received request: %s (seq=%d)", req.Command, req.Seq)

	switch req.Command {
	case "initialize":
		da.handleInitialize(req)
	case "launch":
		da.handleLaunch(req)
	case "setBreakpoints":
		da.handleSetBreakpoints(req)
	case "configurationDone":
		da.handleConfigurationDone(req)
	case "threads":
		da.handleThreads(req)
	case "stackTrace":
		da.handleStackTrace(req)
	case "scopes":
		da.handleScopes(req)
	case "variables":
		da.handleVariables(req)
	case "evaluate":
		da.handleEvaluate(req)
	case "continue":
		da.handleContinue(req)
	case "next":
		da.handleNext(req)
	case "stepIn":
		da.handleStepIn(req)
	case "stepOut":
		da.handleStepOut(req)
	case "pause":
		da.handlePause(req)
	case "disconnect":
		da.handleDisconnect(req)
	case "terminate":
		da.handleTerminate(req)
	default:
		log.Printf("!!! Unknown command: %s", req.Command)
		da.sendResponse(req.Seq, req.Command, false, nil)
	}
}

func (da *DebugAdapter) handleInitialize(req *Request) {
	log.Printf(">>> Initialize request - setting up debug session")

	capabilities := Capabilities{
		SupportsConfigurationDoneRequest: true,
		SupportsConditionalBreakpoints:   false,
		SupportsEvaluateForHovers:        true,
		SupportsSetVariable:              false,
		SupportsTerminateRequest:         true,
	}

	da.sendResponse(req.Seq, req.Command, true, capabilities)
	da.sendEvent("initialized", InitializedEvent{})

	log.Printf(">>> Initialize complete - ready for configuration")
}

func (da *DebugAdapter) handleLaunch(req *Request) {
	log.Printf(">>> Launch request - preparing to run program")

	var args LaunchRequestArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	da.program = args.Program
	log.Printf(">>> Program to debug: %s", da.program)

	// Read the program file
	content, err := os.ReadFile(da.program)
	if err != nil {
		da.sendResponse(req.Seq, req.Command, false, map[string]string{
			"error": fmt.Sprintf("Failed to read program: %v", err),
		})
		return
	}

	da.sourceCode = string(content)
	da.sourceLines = strings.Split(da.sourceCode, "\n")

	log.Printf("Loaded program %s with %d lines", da.program, len(da.sourceLines))

	// Create runtime and enable debugger
	da.vm = goja.New()
	da.debugger = da.vm.EnableDebugger()

	// Set up console.log
	console := da.vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		// Format the output properly with spaces between arguments
		var output string
		for i, arg := range call.Arguments {
			if i > 0 {
				output += " "
			}
			// Convert goja.Value to string
			if arg != nil && !goja.IsUndefined(arg) && !goja.IsNull(arg) {
				output += arg.String()
			} else if goja.IsNull(arg) {
				output += "null"
			} else {
				output += "undefined"
			}
		}
		da.sendEvent("output", map[string]interface{}{
			"category": "console",
			"output":   output + "\n",
		})
		return goja.Undefined()
	})
	da.vm.Set("console", console)

	// Set up debug handler
	da.debugger.SetHandler(da.debugHandler)

	// Only enable step mode if explicitly requested
	if args.StopOnEntry {
		da.nextCommand = goja.DebugStepInto
		da.debugger.SetStepMode(true)
	} else {
		da.nextCommand = goja.DebugContinue
		da.debugger.SetStepMode(false)
	}

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleSetBreakpoints(req *Request) {
	var args SetBreakpointsArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	// Clear existing breakpoints for this file
	filename := args.Source.Path
	if filename == "" {
		filename = args.Source.Name
	}

	// Make sure we're using the same filename as the program
	if filepath.Base(filename) == filepath.Base(da.program) {
		filename = da.program
		log.Printf("Normalized filename from %s to %s", args.Source.Path, filename)
	}

	log.Printf("SetBreakpoints request for file: %s, breakpoints: %d", filename, len(args.Breakpoints))

	// Remove ALL old breakpoints for this file
	if da.debugger != nil {
		existingBPs := da.debugger.GetBreakpoints()
		for _, ebp := range existingBPs {
			if ebp.SourcePos.Filename == filename {
				da.debugger.RemoveBreakpoint(ebp.ID())
			}
		}
	}

	// Clear our tracking maps
	for id, bp := range da.bpMap {
		if bp.Source.Path == filename {
			delete(da.bpMap, id)
		}
	}
	da.breakpoints[filename] = []int{}

	// Add new breakpoints
	var breakpoints []Breakpoint

	for _, sbp := range args.Breakpoints {
		// Add breakpoint to debugger
		gojaID := da.debugger.AddBreakpoint(filename, sbp.Line, sbp.Column)

		da.bpIDCounter++
		bpID := da.bpIDCounter

		bp := Breakpoint{
			ID:       bpID,
			Verified: true,
			Source:   args.Source,
			Line:     sbp.Line,
			Column:   sbp.Column,
		}

		da.bpMap[bpID] = &bp
		da.breakpoints[filename] = append(da.breakpoints[filename], sbp.Line)
		breakpoints = append(breakpoints, bp)

		log.Printf("Added breakpoint: file=%s, line=%d, column=%d, gojaID=%d",
			filename, sbp.Line, sbp.Column, gojaID)
	}

	da.sendResponse(req.Seq, req.Command, true, SetBreakpointsResponseBody{
		Breakpoints: breakpoints,
	})
}

func (da *DebugAdapter) handleConfigurationDone(req *Request) {
	log.Printf(">>> ConfigurationDone - starting execution")

	da.sendResponse(req.Seq, req.Command, true, nil)

	// Start execution in a goroutine
	go da.startExecution()
}

func (da *DebugAdapter) handleThreads(req *Request) {
	threads := []Thread{
		{
			ID:   da.threadID,
			Name: "main",
		},
	}

	da.sendResponse(req.Seq, req.Command, true, ThreadsResponseBody{
		Threads: threads,
	})
}

func (da *DebugAdapter) handleStackTrace(req *Request) {
	var args StackTraceArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	// Get call stack
	stack := da.vm.CaptureCallStack(10, nil)

	var frames []StackFrame
	da.frameMap = make(map[int]*goja.StackFrame)

	for i, frame := range stack {
		frameID := i + 1
		da.frameMap[frameID] = &frame

		funcName := frame.FuncName()
		if funcName == "" {
			funcName = "(anonymous)"
		}

		pos := frame.Position()
		sf := StackFrame{
			ID:     frameID,
			Name:   funcName,
			Line:   pos.Line,
			Column: 1,
			Source: Source{
				Name: filepath.Base(pos.Filename),
				Path: pos.Filename,
			},
		}

		frames = append(frames, sf)
	}

	da.sendResponse(req.Seq, req.Command, true, StackTraceResponseBody{
		StackFrames: frames,
		TotalFrames: len(frames),
	})
}

func (da *DebugAdapter) handleScopes(req *Request) {
	var args ScopesArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	// Create both Local and Global scopes
	scopes := []Scope{}
	
	// Local scope
	da.varRefCounter++
	localRef := da.varRefCounter
	da.varRefMap[localRef] = map[string]interface{}{
		"type":    "local",
		"frameID": args.FrameID,
	}
	
	scopes = append(scopes, Scope{
		Name:               "Local",
		VariablesReference: localRef,
		Expensive:          false,
	})
	
	// Global scope
	da.varRefCounter++
	globalRef := da.varRefCounter
	da.varRefMap[globalRef] = map[string]interface{}{
		"type": "global",
	}
	
	scopes = append(scopes, Scope{
		Name:               "Global",
		VariablesReference: globalRef,
		Expensive:          false,
	})

	da.sendResponse(req.Seq, req.Command, true, ScopesResponseBody{
		Scopes: scopes,
	})
}

func (da *DebugAdapter) handleVariables(req *Request) {
	var args VariablesArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}
	
	log.Printf("handleVariables: variablesReference=%d", args.VariablesReference)

	var variables []Variable

	// Get scope info
	if scopeInfo, ok := da.varRefMap[args.VariablesReference]; ok {
		if info, ok := scopeInfo.(map[string]interface{}); ok {
			scopeType := info["type"].(string)
			log.Printf("Scope type: %s", scopeType)
			
			if scopeType == "local" {
				frameID := info["frameID"].(int)
				log.Printf("Getting local variables for frame %d", frameID)
				variables = da.getLocalVariables(frameID)
			} else if scopeType == "global" {
				log.Printf("Getting global variables")
				variables = da.getGlobalVariables()
			}
		} else if val, ok := scopeInfo.(goja.Value); ok {
			// It's an object to expand
			log.Printf("Expanding object properties")
			variables = da.getObjectProperties(val)
		}
	} else {
		log.Printf("WARNING: variablesReference %d not found in map", args.VariablesReference)
	}

	log.Printf("Returning %d variables", len(variables))
	da.sendResponse(req.Seq, req.Command, true, VariablesResponseBody{
		Variables: variables,
	})
}

func (da *DebugAdapter) getLocalVariables(frameID int) []Variable {
	var variables []Variable
	
	// Get frame from our map
	frame, ok := da.frameMap[frameID]
	if !ok {
		log.Printf("Frame %d not found", frameID)
		return variables
	}
	
	funcName := frame.FuncName()
	log.Printf("Getting variables for function '%s'", funcName)
	
	// Get local variables using the new Goja API
	localVars := frame.GetLocalVariables()
	log.Printf("Found %d local variables", len(localVars))
	
	for name, value := range localVars {
		variable := da.valueToVariable(name, value)
		variables = append(variables, variable)
	}
	
	// Get arguments using the new Goja API
	args := frame.GetArguments()
	log.Printf("Found %d arguments", len(args))
	
	for i, arg := range args {
		name := fmt.Sprintf("argument[%d]", i)
		// Try to get the actual parameter name if possible
		variable := da.valueToVariable(name, arg)
		variables = append(variables, variable)
	}
	
	// Get 'this' value
	thisValue := frame.GetThis()
	if thisValue != nil && !goja.IsUndefined(thisValue) {
		variable := da.valueToVariable("this", thisValue)
		variables = append(variables, variable)
	}
	
	return variables
}

func (da *DebugAdapter) getGlobalVariables() []Variable {
	var variables []Variable
	
	globalObj := da.vm.GlobalObject()
	if globalObj != nil {
		for _, key := range globalObj.Keys() {
			// Skip built-ins
			if da.isBuiltIn(key) {
				continue
			}

			val := globalObj.Get(key)
			if val != nil {
				variable := da.valueToVariable(key, val)
				variables = append(variables, variable)
			}
		}
	}
	
	return variables
}

func (da *DebugAdapter) valueToVariable(name string, val goja.Value) Variable {
	value := "undefined"
	varType := "undefined"
	varRef := 0
	
	if !goja.IsUndefined(val) && !goja.IsNull(val) {
		value = val.String()
		varType = da.getValueType(val)
		
		// Create reference for complex types
		if obj, ok := val.(*goja.Object); ok {
			if _, isFunc := goja.AssertFunction(obj); !isFunc {
				da.varRefCounter++
				varRef = da.varRefCounter
				da.varRefMap[varRef] = val
				value = da.formatComplexValue(val)
			} else {
				value = fmt.Sprintf("[Function: %s]", name)
			}
		}
		
		// Add quotes for strings
		if varType == "string" {
			value = fmt.Sprintf("%q", val.String())
		}
	} else if goja.IsNull(val) {
		value = "null"
		varType = "null"
	}
	
	return Variable{
		Name:               name,
		Value:              value,
		Type:               varType,
		VariablesReference: varRef,
	}
}

func (da *DebugAdapter) getObjectProperties(val goja.Value) []Variable {
	var variables []Variable
	
	obj, ok := val.(*goja.Object)
	if !ok {
		return variables
	}
	
	for _, key := range obj.Keys() {
		propVal := obj.Get(key)
		if propVal != nil {
			variable := da.valueToVariable(key, propVal)
			variables = append(variables, variable)
		}
	}
	
	// For arrays, add length property at the beginning
	if arr := obj.Export(); arr != nil {
		if reflect.TypeOf(arr).Kind() == reflect.Slice {
			s := reflect.ValueOf(arr)
			variables = append([]Variable{{
				Name:               "length",
				Value:              strconv.Itoa(s.Len()),
				Type:               "number",
				VariablesReference: 0,
			}}, variables...)
		}
	}
	
	return variables
}

func (da *DebugAdapter) getValueType(val goja.Value) string {
	if goja.IsUndefined(val) {
		return "undefined"
	}
	if goja.IsNull(val) {
		return "null"
	}
	
	switch val.Export().(type) {
	case string:
		return "string"
	case int, int64, float64:
		return "number"
	case bool:
		return "boolean"
	default:
		if obj, ok := val.(*goja.Object); ok {
			if _, ok := goja.AssertFunction(obj); ok {
				return "function"
			}
			// Check if it's an array
			if arr := obj.Export(); arr != nil && reflect.TypeOf(arr).Kind() == reflect.Slice {
				return "array"
			}
			return "object"
		}
		return "unknown"
	}
}

func (da *DebugAdapter) formatComplexValue(val goja.Value) string {
	obj, ok := val.(*goja.Object)
	if !ok {
		return val.String()
	}
	
	// Check if it's an array
	if arr := obj.Export(); arr != nil && reflect.TypeOf(arr).Kind() == reflect.Slice {
		s := reflect.ValueOf(arr)
		return fmt.Sprintf("Array[%d]", s.Len())
	}
	
	// For objects, show a preview
	keys := obj.Keys()
	if len(keys) == 0 {
		return "{}"
	} else if len(keys) <= 3 {
		return fmt.Sprintf("{%s}", strings.Join(keys, ", "))
	} else {
		return fmt.Sprintf("{%s, ...}", strings.Join(keys[:3], ", "))
	}
}

func (da *DebugAdapter) isBuiltIn(name string) bool {
	builtins := []string{
		"console", "Object", "Function", "Array", "String", "Number",
		"Boolean", "Date", "JSON", "Math", "RegExp", "Error",
		"undefined", "Infinity", "NaN", "parseInt", "parseFloat",
		"isNaN", "isFinite", "eval", "decodeURI", "decodeURIComponent",
		"encodeURI", "encodeURIComponent", "escape", "unescape",
	}
	
	for _, b := range builtins {
		if name == b {
			return true
		}
	}
	return false
}

func (da *DebugAdapter) handleEvaluate(req *Request) {
	var args EvaluateArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	var result goja.Value
	var err error
	
	// Check if we have a frame context
	if args.FrameID > 0 {
		// Use the new EvaluateInFrame API
		log.Printf("Evaluating '%s' in frame %d", args.Expression, args.FrameID)
		result, err = da.vm.EvaluateInFrame(args.Expression, args.FrameID-1) // Convert to 0-based
	} else {
		// Evaluate in global context
		log.Printf("Evaluating '%s' in global context", args.Expression)
		result, err = da.vm.RunString(args.Expression)
	}
	
	if err != nil {
		da.sendResponse(req.Seq, req.Command, false, map[string]string{
			"error": err.Error(),
		})
		return
	}

	value := "(undefined)"
	varRef := 0
	varType := "undefined"
	
	if result != nil && !goja.IsUndefined(result) {
		value = result.String()
		varType = da.getValueType(result)
		
		// Create reference for complex types
		if obj, ok := result.(*goja.Object); ok {
			if _, isFunc := goja.AssertFunction(obj); !isFunc {
				da.varRefCounter++
				varRef = da.varRefCounter
				da.varRefMap[varRef] = result
				value = da.formatComplexValue(result)
			}
		}
		
		// Add quotes for strings in the result
		if varType == "string" {
			value = fmt.Sprintf("%q", result.String())
		}
	}

	da.sendResponse(req.Seq, req.Command, true, EvaluateResponseBody{
		Result:             value,
		Type:               varType,
		VariablesReference: varRef,
	})
}

func (da *DebugAdapter) handleContinue(req *Request) {
	log.Printf("=== CONTINUE: Setting next command to Continue")
	
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugContinue
	da.debugger.SetStepMode(false)
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, ContinueResponseBody{
		AllThreadsContinued: true,
	})
}

func (da *DebugAdapter) handleNext(req *Request) {
	log.Printf("=== NEXT: Setting next command to StepOver")
	
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugStepOver
	da.debugger.SetStepMode(true)
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleStepIn(req *Request) {
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugStepInto
	da.debugger.SetStepMode(true)
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleStepOut(req *Request) {
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugStepOut
	da.debugger.SetStepMode(true)
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handlePause(req *Request) {
	da.debugger.SetStepMode(true)
	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleDisconnect(req *Request) {
	da.sendResponse(req.Seq, req.Command, true, nil)
	da.terminated = true
}

func (da *DebugAdapter) handleTerminate(req *Request) {
	da.sendResponse(req.Seq, req.Command, true, nil)
	da.sendEvent("terminated", TerminatedEventBody{})
	da.terminated = true
}

func (da *DebugAdapter) debugHandler(state *goja.DebuggerState) goja.DebugCommand {
	log.Printf("\n=== DEBUG HANDLER ===")
	log.Printf("Position: %s:%d:%d", state.SourcePos.Filename, state.SourcePos.Line, state.SourcePos.Column)
	log.Printf("Has Breakpoint: %v", state.Breakpoint != nil)
	
	// Get current line content for context
	if state.SourcePos.Line > 0 && state.SourcePos.Line <= len(da.sourceLines) {
		line := da.sourceLines[state.SourcePos.Line-1]
		log.Printf("Current line: %s", strings.TrimSpace(line))
	}

	// Check if we're at the end of the script
	if state.SourcePos.Filename == "" && state.SourcePos.Line == 0 {
		log.Printf("End of script - continuing")
		return goja.DebugContinue
	}

	// Determine stop reason
	reason := "step"
	if state.Breakpoint != nil {
		reason = "breakpoint"
		log.Printf("Hit breakpoint at line %d", state.SourcePos.Line)
	}
	
	// Get the current command mode
	da.debugStateMutex.Lock()
	currentCmd := da.nextCommand
	da.debugStateMutex.Unlock()
	
	// If we're in continue mode and there's no breakpoint, just continue
	if currentCmd == goja.DebugContinue && state.Breakpoint == nil {
		log.Printf("In continue mode with no breakpoint - continuing")
		return goja.DebugContinue
	}

	// Send stopped event
	da.sendEvent("stopped", StoppedEventBody{
		Reason:            reason,
		ThreadID:          da.threadID,
		AllThreadsStopped: true,
	})

	// Wait for command
	da.debugStateMutex.Lock()
	da.waitingForCmd = true
	da.debugStateMutex.Unlock()

	log.Printf("Waiting for debugger command...")
	<-da.commandReady

	da.debugStateMutex.Lock()
	cmd := da.nextCommand
	da.debugStateMutex.Unlock()

	log.Printf("Returning command: %v", cmd)
	return cmd
}

func (da *DebugAdapter) startExecution() {
	da.running = true

	log.Printf("Starting script execution...")
	_, err := da.vm.RunScript(da.program, da.sourceCode)
	
	da.running = false
	
	if err != nil {
		log.Printf("Script error: %v", err)
		da.sendEvent("output", map[string]interface{}{
			"category": "stderr",
			"output":   fmt.Sprintf("Error: %v\n", err),
		})
	}
	
	// Send exit event
	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	
	da.sendEvent("exited", ExitedEventBody{
		ExitCode: exitCode,
	})
	da.sendEvent("terminated", TerminatedEventBody{})
}