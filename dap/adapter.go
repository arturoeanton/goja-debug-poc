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

	// Pause state
	pauseChan chan struct{}
	stepMode  goja.DebugCommand

	// Synchronization
	initMutex sync.Mutex
}

func NewDebugAdapter(reader io.Reader, writer io.Writer) *DebugAdapter {
	return &DebugAdapter{
		reader:      bufio.NewReader(reader),
		writer:      writer,
		seq:         1,
		breakpoints: make(map[string][]int),
		bpMap:       make(map[int]*Breakpoint),
		varRefMap:   make(map[int]interface{}),
		frameMap:    make(map[int]*goja.StackFrame),
		threadID:    1,
		pauseChan:   make(chan struct{}, 1),
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
	console.Set("log", func(args ...interface{}) {
		// Format the output properly with spaces between arguments
		var output string
		for i, arg := range args {
			if i > 0 {
				output += " "
			}
			output += fmt.Sprint(arg)
		}
		da.sendEvent("output", map[string]interface{}{
			"category": "console",
			"output":   output + "\n",
		})
	})
	da.vm.Set("console", console)

	// Set up debug handler
	da.debugger.SetHandler(da.debugHandler)

	// Only enable step mode if explicitly requested
	if args.StopOnEntry {
		da.debugger.SetStepMode(true)
		da.stepMode = goja.DebugStepInto
	} else {
		// Make sure we start in continue mode
		da.debugger.SetStepMode(false)
		da.stepMode = goja.DebugContinue
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
	// VS Code might send absolute path, but we loaded with relative path
	if filepath.Base(filename) == filepath.Base(da.program) {
		filename = da.program
		log.Printf("Normalized filename from %s to %s", args.Source.Path, filename)
	}

	log.Printf("SetBreakpoints request for file: %s, breakpoints: %d", filename, len(args.Breakpoints))

	// Remove ALL old breakpoints for this file
	if da.debugger != nil {
		// Get all existing breakpoints
		existingBPs := da.debugger.GetBreakpoints()
		log.Printf("Found %d existing breakpoints", len(existingBPs))
		for _, ebp := range existingBPs {
			if ebp.SourcePos.Filename == filename {
				log.Printf("Removing breakpoint ID=%d at line %d", ebp.ID(), ebp.SourcePos.Line)
				da.debugger.RemoveBreakpoint(ebp.ID())
			}
		}
	} else {
		log.Printf("Warning: debugger not initialized yet, deferring breakpoint setup")
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
		// Note: Goja uses 1-based line numbers, same as VS Code
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

		log.Printf("Added breakpoint: file=%s, line=%d, column=%d, gojaID=%d, dapID=%d",
			filename, sbp.Line, sbp.Column, gojaID, bpID)
	}

	da.sendResponse(req.Seq, req.Command, true, SetBreakpointsResponseBody{
		Breakpoints: breakpoints,
	})
}

func (da *DebugAdapter) handleConfigurationDone(req *Request) {
	log.Printf(">>> ConfigurationDone - all breakpoints set, starting execution")

	da.sendResponse(req.Seq, req.Command, true, nil)

	// Set initial step mode to continue (not step)
	da.stepMode = goja.DebugContinue
	log.Printf(">>> Initial step mode: Continue")

	// Start execution in a goroutine
	log.Printf(">>> Launching script execution in goroutine")
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
			Column: 1, // VS Code espera 1-based, usar 1 para inicio de línea
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

	da.varRefCounter++
	localRef := da.varRefCounter

	// Store frame ID for variable lookup
	da.varRefMap[localRef] = args.FrameID

	scopes := []Scope{
		{
			Name:               "Local",
			VariablesReference: localRef,
			Expensive:          false,
		},
	}

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

	var variables []Variable

	// Get the frame ID from the variable reference
	if frameID, ok := da.varRefMap[args.VariablesReference]; ok {
		if fid, ok := frameID.(int); ok {
			// Get variables for this frame
			// This is a simplified version - in production you'd get actual scope variables
			variables = da.getFrameVariables(fid)
		}
	}

	da.sendResponse(req.Seq, req.Command, true, VariablesResponseBody{
		Variables: variables,
	})
}

func (da *DebugAdapter) getFrameVariables(frameID int) []Variable {
	var variables []Variable

	// Get the current global object to access variables
	if da.vm == nil {
		log.Printf("Warning: VM is nil when getting variables")
		return variables
	}

	// Get global variables
	globalObj := da.vm.GlobalObject()
	if globalObj == nil {
		log.Printf("Warning: Global object is nil")
		return variables
	}

	// Get all property names from the global object
	for _, key := range globalObj.Keys() {
		val := globalObj.Get(key)
		if val != nil {
			// Skip built-in objects and functions for now
			if key == "console" || key == "Object" || key == "Function" || key == "Array" {
				continue
			}

			varType := "undefined"
			varValue := "undefined"
			varRef := 0

			// Get the actual value and type
			if !goja.IsUndefined(val) && !goja.IsNull(val) {
				varValue = val.String()

				// Determine type
				switch val.ExportType().Kind() {
				case reflect.String:
					varType = "string"
				case reflect.Int, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
					varType = "number"
				case reflect.Bool:
					varType = "boolean"
				case reflect.Map, reflect.Struct:
					varType = "object"
					// Create a variable reference for objects
					da.varRefCounter++
					varRef = da.varRefCounter
					da.varRefMap[varRef] = val
				case reflect.Slice, reflect.Array:
					varType = "array"
					// Create a variable reference for arrays
					da.varRefCounter++
					varRef = da.varRefCounter
					da.varRefMap[varRef] = val
				default:
					if _, ok := val.(*goja.Object); ok {
						varType = "object"
						da.varRefCounter++
						varRef = da.varRefCounter
						da.varRefMap[varRef] = val
					}
				}
			}

			variables = append(variables, Variable{
				Name:               key,
				Value:              varValue,
				Type:               varType,
				VariablesReference: varRef,
			})
		}
	}

	log.Printf("Found %d variables in frame %d", len(variables), frameID)
	return variables
}

func (da *DebugAdapter) formatValue(val goja.Value) string {
	if val == nil || goja.IsUndefined(val) {
		return "undefined"
	}
	if goja.IsNull(val) {
		return "null"
	}
	return val.String()
}

func (da *DebugAdapter) handleEvaluate(req *Request) {
	var args EvaluateArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	// Try to evaluate the expression
	result, err := da.vm.RunString(args.Expression)
	if err != nil {
		da.sendResponse(req.Seq, req.Command, false, map[string]string{
			"error": err.Error(),
		})
		return
	}

	value := "(undefined)"
	if result != nil {
		value = result.String()
	}

	da.sendResponse(req.Seq, req.Command, true, EvaluateResponseBody{
		Result:             value,
		VariablesReference: 0,
	})
}

func (da *DebugAdapter) handleContinue(req *Request) {
	log.Printf("Continue request received")
	da.stepMode = goja.DebugContinue
	da.debugger.SetStepMode(false)

	// Signal to continue
	select {
	case da.pauseChan <- struct{}{}:
		log.Printf("Sent continue signal")
	default:
		log.Printf("Warning: pause channel was full")
	}

	da.sendResponse(req.Seq, req.Command, true, ContinueResponseBody{
		AllThreadsContinued: true,
	})
}

func (da *DebugAdapter) handleNext(req *Request) {
	log.Printf("Next (step over) request received")
	da.stepMode = goja.DebugStepOver
	da.debugger.SetStepMode(true)

	// Signal to continue
	select {
	case da.pauseChan <- struct{}{}:
		log.Printf("Sent step over signal")
	default:
		log.Printf("Warning: pause channel was full")
	}

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleStepIn(req *Request) {
	da.stepMode = goja.DebugStepInto
	da.debugger.SetStepMode(true)

	// Signal to continue
	select {
	case da.pauseChan <- struct{}{}:
	default:
	}

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleStepOut(req *Request) {
	da.stepMode = goja.DebugStepOut
	da.debugger.SetStepMode(true)

	// Signal to continue
	select {
	case da.pauseChan <- struct{}{}:
	default:
	}

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

// Track last line to avoid multiple stops on same line
var lastStoppedLine int

func (da *DebugAdapter) debugHandler(state *goja.DebuggerState) goja.DebugCommand {
	// Track last position to avoid duplicate stops
	currentPos := fmt.Sprintf("%s:%d:%d", state.SourcePos.Filename, state.SourcePos.Line, state.SourcePos.Column)

	log.Printf("\n=== DEBUG HANDLER CALLED ===")
	log.Printf("Position: %s", currentPos)
	log.Printf("Has Breakpoint: %v", state.Breakpoint != nil)
	log.Printf("Current step mode: %v", da.stepMode)
	log.Printf("Last stopped line: %d", lastStoppedLine)

	// Check if we're at the end of the script (position :0:0)
	if state.SourcePos.Filename == "" && state.SourcePos.Line == 0 {
		log.Printf(">>> END OF SCRIPT: Reached position :0:0, continuing to finish")
		return goja.DebugContinue
	}
	
	// REMOVED: Skip same line logic - let it stop at each column
	// This is more predictable and avoids issues with loops

	// Send stopped event
	reason := "step"
	description := ""

	// Check if we should have hit a breakpoint
	if state.Breakpoint == nil {
		if bps, ok := da.breakpoints[state.SourcePos.Filename]; ok {
			for _, bpLine := range bps {
				if bpLine == state.SourcePos.Line {
					log.Printf("WARNING: At breakpoint line %d but state.Breakpoint is nil! This might be a Goja issue.", bpLine)
					// Force it to act like a breakpoint
					reason = "breakpoint"
					description = fmt.Sprintf("Breakpoint at line %d", state.SourcePos.Line)
				}
			}
		}
	}

	if state.Breakpoint != nil {
		reason = "breakpoint"
		description = fmt.Sprintf("Breakpoint at line %d", state.SourcePos.Line)
		log.Printf("Hit breakpoint ID=%d at %s:%d:%d (BP was set for line %d)",
			state.Breakpoint.ID(), state.SourcePos.Filename, state.SourcePos.Line, state.SourcePos.Column,
			state.Breakpoint.SourcePos.Line)

		// Check if we're at the wrong line
		if state.SourcePos.Line != state.Breakpoint.SourcePos.Line {
			log.Printf("WARNING: Breakpoint line mismatch! Stopped at line %d but breakpoint is for line %d",
				state.SourcePos.Line, state.Breakpoint.SourcePos.Line)
		}
	} else {
		// Get the source line for context
		lineText := ""
		if state.SourcePos.Line > 0 && state.SourcePos.Line <= len(da.sourceLines) {
			lineText = strings.TrimSpace(da.sourceLines[state.SourcePos.Line-1])
			if len(lineText) > 50 {
				lineText = lineText[:47] + "..."
			}
		}
		log.Printf("Stepped to %s:%d:%d (PC=%d) | %s",
			state.SourcePos.Filename, state.SourcePos.Line, state.SourcePos.Column, state.PC, lineText)

		// If we're in step over mode and this looks like we're entering a branch we shouldn't
		// (e.g., else branch when if was true), we might want to continue
		// This is a heuristic - proper fix would be in the Goja VM itself
		if da.stepMode == goja.DebugStepOver {
			// Check if this is likely a branch we shouldn't enter
			// For now, we'll trust Goja's stepping logic
		}
	}

	da.sendEvent("stopped", StoppedEventBody{
		Reason:            reason,
		Description:       description,
		ThreadID:          da.threadID,
		AllThreadsStopped: true,
	})

	// Update last stopped line
	lastStoppedLine = state.SourcePos.Line

	// Wait for continue signal
	log.Printf(">>> PAUSED: Waiting for continue signal at %s", currentPos)
	log.Printf(">>> To resume, VS Code needs to send: continue, next, stepIn, or stepOut")

	// This blocks until we receive a signal
	<-da.pauseChan

	log.Printf(">>> RESUMING: Received signal, step mode = %v", da.stepMode)

	// Reset last line if we're continuing (not stepping)
	if da.stepMode == goja.DebugContinue {
		lastStoppedLine = -1
		log.Printf(">>> Reset last stopped line for continue mode")
	}

	log.Printf(">>> Returning command: %v", da.stepMode)
	return da.stepMode
}

func (da *DebugAdapter) startExecution() {
	da.running = true

	log.Printf("Starting execution of script: %s", da.program)

	// Don't prime the channel - let the debugger pause naturally
	log.Printf("Starting script execution without priming pause channel")

	log.Printf(">>> Script execution starting...")
	result, err := da.vm.RunScript(da.program, da.sourceCode)
	log.Printf(">>> Script execution completed. Error: %v", err)
	
	da.running = false
	
	if err != nil {
		log.Printf(">>> Script error: %v", err)
		da.sendEvent("output", map[string]interface{}{
			"category": "stderr",
			"output":   fmt.Sprintf("Error: %v\n", err),
		})
	} else if result != nil {
		log.Printf(">>> Script result: %v", result)
	}
	
	// Send exit event
	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	
	log.Printf(">>> Sending exit events (code=%d)", exitCode)
	da.sendEvent("exited", ExitedEventBody{
		ExitCode: exitCode,
	})
	da.sendEvent("terminated", TerminatedEventBody{})
}
