# Guía de Implementación del Debugger de Goja - Core del Debugger

## Archivo: debugger.go (NUEVO)

Crear el archivo `debugger.go` en la raíz del proyecto Goja con el siguiente contenido:

### Estructura Completa del Archivo

```go
package goja

import (
	"fmt"
	"log"
	"sync"
	"github.com/dop251/goja/file"
)

// Debugger proporciona capacidades de debugging para el runtime de Goja
type Debugger struct {
	runtime  *Runtime
	mu       sync.Mutex
	
	// Estado del debugger
	enabled      bool
	paused       bool
	flags        uint32
	stepMode     DebugCommand
	stepDepth    int
	
	// Breakpoints
	breakpoints  map[int]*Breakpoint
	nextID       int
	
	// Handler para eventos de debugging
	handler      DebugHandler
	
	// Variables para inspección
	variableRefs map[int][]Variable
	nextVarRef   int
	
	// Logger opcional
	logger       *log.Logger
}

// Constantes para flags
const (
	FlagStepMode    uint32 = 1 << iota
	FlagPaused
	FlagBreakpoints
)

// DebugCommand representa los comandos de control del debugger
type DebugCommand int

const (
	DebugContinue DebugCommand = iota
	DebugStepOver
	DebugStepInto
	DebugStepOut
	DebugPause
)

// DebugHandler es la función callback para manejar eventos de debugging
type DebugHandler func(state *DebuggerState) DebugCommand

// Breakpoint representa un punto de interrupción
type Breakpoint struct {
	id       int
	file     string
	line     int
	column   int
	enabled  bool
	hits     int
	mu       sync.Mutex
}

// DebuggerState contiene el estado actual cuando el debugger se pausa
type DebuggerState struct {
	PC                 int
	SourcePos          file.Position
	Breakpoint         *Breakpoint
	StepMode           bool
	InNativeCall       bool
	NativeFunctionName string
	CallStack          []StackFrame
	DebugStack         []DebugFrame
}

// StackFrame representa un frame en el call stack
type StackFrame interface {
	FuncName() string
	Position() string
}

// DebugFrame contiene información detallada de un frame para debugging
type DebugFrame struct {
	ID         int
	Name       string
	SourcePos  file.Position
	This       Value
	Scopes     []Scope
	stackFrame // Embedded para implementar StackFrame
}

// Scope representa un ámbito de variables
type Scope struct {
	Name          string
	VariablesRef  int
	NamedVars     int
	IndexedVars   int
}

// Variable representa una variable en el debugger
type Variable struct {
	Name           string
	Value          Value
	Type           string
	VariablesRef   int
	IndexedVars    int
	NamedVars      int
}

// NewDebugger crea una nueva instancia del debugger
func NewDebugger(r *Runtime) *Debugger {
	return &Debugger{
		runtime:      r,
		breakpoints:  make(map[int]*Breakpoint),
		variableRefs: make(map[int][]Variable),
	}
}

// SetHandler establece el handler para eventos de debugging
func (d *Debugger) SetHandler(handler DebugHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = handler
	d.logger.Printf("SetHandler: Handler set\n")
}

// Enable activa el debugger
func (d *Debugger) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
	d.flags |= FlagBreakpoints
	d.logger.Printf("Enable: Debugger enabled\n")
}

// Disable desactiva el debugger
func (d *Debugger) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
	d.flags = 0
	d.logger.Printf("Disable: Debugger disabled\n")
}

// AddBreakpoint agrega un breakpoint en la posición especificada
func (d *Debugger) AddBreakpoint(file string, line, column int) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	bp := &Breakpoint{
		id:      d.nextID,
		file:    file,
		line:    line,
		column:  column,
		enabled: true,
	}
	
	d.breakpoints[d.nextID] = bp
	d.nextID++
	
	d.logger.Printf("AddBreakpoint: Added breakpoint #%d at %s:%d:%d\n", 
		bp.id, file, line, column)
	
	return bp.id
}

// RemoveBreakpoint elimina un breakpoint por ID
func (d *Debugger) RemoveBreakpoint(id int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if _, exists := d.breakpoints[id]; exists {
		delete(d.breakpoints, id)
		d.logger.Printf("RemoveBreakpoint: Removed breakpoint #%d\n", id)
		return true
	}
	return false
}

// ClearBreakpoints elimina todos los breakpoints
func (d *Debugger) ClearBreakpoints() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.breakpoints = make(map[int]*Breakpoint)
	d.logger.Printf("ClearBreakpoints: All breakpoints cleared\n")
}

// Continue reanuda la ejecución hasta el próximo breakpoint
func (d *Debugger) Continue() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags &^= FlagPaused | FlagStepMode
	d.stepMode = DebugContinue
	d.logger.Printf("Continue: flags=%b\n", d.flags)
}

// StepOver ejecuta la siguiente línea sin entrar en funciones
func (d *Debugger) StepOver() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepOver
	d.stepDepth = len(d.runtime.vm.callStack)
	d.logger.Printf("StepOver: flags=%b, stepMode=%v, stepDepth=%d\n", 
		d.flags, d.stepMode, d.stepDepth)
}

// StepInto ejecuta la siguiente línea entrando en funciones
func (d *Debugger) StepInto() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepInto
	d.logger.Printf("StepInto: flags=%b, stepMode=%v\n", d.flags, d.stepMode)
}

// StepOut continúa hasta que la función actual retorna
func (d *Debugger) StepOut() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepOut
	d.stepDepth = len(d.runtime.vm.callStack) - 1
	d.logger.Printf("StepOut: flags=%b, stepMode=%v, stepDepth=%d\n", 
		d.flags, d.stepMode, d.stepDepth)
}

// Pause pausa la ejecución en la siguiente oportunidad
func (d *Debugger) Pause() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagPaused
	d.logger.Printf("Pause: flags=%b\n", d.flags)
}

// checkBreakpoint es llamado por la VM para verificar si debe pausar
// IMPORTANTE: Este método es llamado frecuentemente, debe ser eficiente
func (d *Debugger) checkBreakpoint(vm *vm) bool {
	// Quick check sin lock
	if d.flags == 0 && len(d.breakpoints) == 0 {
		return false
	}
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Si ya estamos pausados, continuar pausados
	if d.flags&FlagPaused != 0 {
		d.logger.Printf("checkBreakpoint: Already paused, returning true\n")
		return true
	}
	
	// Obtener posición actual
	currentLine := -1
	if vm.prg != nil && vm.pc < len(vm.prg.src) {
		if pos := vm.prg.src[vm.pc]; pos.srcPos > 0 {
			position := vm.prg.Position(pos.srcPos)
			currentLine = position.Line
		}
	}
	
	// Verificar breakpoints
	if d.flags&FlagBreakpoints != 0 && currentLine > 0 {
		for _, bp := range d.breakpoints {
			if bp.enabled && bp.line == currentLine {
				bp.mu.Lock()
				bp.hits++
				bp.mu.Unlock()
				d.flags |= FlagPaused
				d.logger.Printf("checkBreakpoint: Hit breakpoint #%d at PC=%d, hits=%d\n",
					bp.id, vm.pc, bp.hits)
				return true
			}
		}
	}
	
	// Verificar step mode
	if d.flags&FlagStepMode != 0 {
		switch d.stepMode {
		case DebugStepInto:
			d.flags |= FlagPaused
			d.logger.Printf("checkBreakpoint: StepInto - pausing at PC=%d, Line=%d\n",
				vm.pc, currentLine)
			return true
		case DebugStepOver:
			if len(vm.callStack) <= d.stepDepth {
				d.flags |= FlagPaused
				d.logger.Printf("checkBreakpoint: StepOver - pausing at PC=%d, Line=%d\n",
					vm.pc, currentLine)
				return true
			}
		case DebugStepOut:
			if len(vm.callStack) < d.stepDepth {
				d.flags |= FlagPaused
				d.logger.Printf("checkBreakpoint: StepOut - pausing at PC=%d, Line=%d\n",
					vm.pc, currentLine)
				return true
			}
		}
	}
	
	return false
}

// handlePause es llamado cuando la VM se pausa
func (d *Debugger) handlePause(vm *vm) {
	d.logger.Printf("handlePause: Called, PC=%d, prg=%v\n", vm.pc, vm.prg != nil)
	
	if d.handler == nil {
		d.logger.Printf("handlePause: No handler set, continuing\n")
		d.Continue()
		return
	}
	
	// Construir el estado actual
	state := d.buildDebuggerState(vm)
	
	// Llamar al handler
	d.logger.Printf("handlePause: Calling handler at Line=%d, PC=%d, InNative=%v, NativeName=%s\n",
		state.SourcePos.Line, state.PC, state.InNativeCall, state.NativeFunctionName)
	
	cmd := d.handler(state)
	
	d.logger.Printf("handlePause: Handler returned command: %v\n", cmd)
	
	// Ejecutar el comando
	d.executeCommand(cmd)
}

// buildDebuggerState construye el estado actual del debugger
func (d *Debugger) buildDebuggerState(vm *vm) *DebuggerState {
	state := &DebuggerState{
		PC:       vm.pc,
		StepMode: d.flags&FlagStepMode != 0,
	}
	
	// Obtener posición en el código fuente
	if vm.prg != nil && vm.pc < len(vm.prg.src) {
		if pos := vm.prg.src[vm.pc]; pos.srcPos > 0 {
			state.SourcePos = vm.prg.Position(pos.srcPos)
		}
	}
	
	// Verificar si hit un breakpoint
	for _, bp := range d.breakpoints {
		if bp.enabled && bp.line == state.SourcePos.Line {
			state.Breakpoint = bp
			break
		}
	}
	
	// Construir call stack
	state.CallStack = d.buildCallStack(vm)
	
	// Construir debug stack con scopes
	state.DebugStack = d.buildDebugStack(vm)
	
	return state
}

// buildCallStack construye el call stack
func (d *Debugger) buildCallStack(vm *vm) []StackFrame {
	frames := make([]StackFrame, 0, len(vm.callStack))
	
	// Frame actual
	if vm.prg != nil {
		frames = append(frames, &stackFrame{
			prg: vm.prg,
			pc:  vm.pc,
		})
	}
	
	// Frames anteriores
	for i := len(vm.callStack) - 1; i >= 0; i-- {
		frames = append(frames, &vm.callStack[i])
	}
	
	return frames
}

// buildDebugStack construye el stack con información de debugging
func (d *Debugger) buildDebugStack(vm *vm) []DebugFrame {
	frames := make([]DebugFrame, 0)
	
	// Frame actual
	if vm.prg != nil {
		frame := DebugFrame{
			ID:   0,
			Name: vm.prg.funcName.String(),
			stackFrame: stackFrame{
				prg: vm.prg,
				pc:  vm.pc,
			},
		}
		
		if vm.pc < len(vm.prg.src) && vm.prg.src[vm.pc].srcPos > 0 {
			frame.SourcePos = vm.prg.Position(vm.prg.src[vm.pc].srcPos)
		}
		
		// Agregar scopes
		frame.Scopes = d.buildScopes(vm, 0)
		
		frames = append(frames, frame)
	}
	
	return frames
}

// buildScopes construye los scopes para un frame
func (d *Debugger) buildScopes(vm *vm, frameIndex int) []Scope {
	scopes := make([]Scope, 0)
	
	// Scope local
	localRef := d.createVariableReference(d.getLocalVariables(vm, frameIndex))
	scopes = append(scopes, Scope{
		Name:         "Local",
		VariablesRef: localRef,
		NamedVars:    len(d.variableRefs[localRef]),
	})
	
	// Scope closure (si existe)
	if vm.stash != nil {
		closureRef := d.createVariableReference(d.getClosureVariables(vm))
		if len(d.variableRefs[closureRef]) > 0 {
			scopes = append(scopes, Scope{
				Name:         "Closure",
				VariablesRef: closureRef,
				NamedVars:    len(d.variableRefs[closureRef]),
			})
		}
	}
	
	// Scope global
	globalRef := d.createVariableReference(d.getGlobalVariables(vm))
	scopes = append(scopes, Scope{
		Name:         "Global",
		VariablesRef: globalRef,
		NamedVars:    len(d.variableRefs[globalRef]),
	})
	
	return scopes
}

// GetVariables obtiene las variables de un scope por referencia
func (d *Debugger) GetVariables(ref int) []Variable {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if vars, ok := d.variableRefs[ref]; ok {
		return vars
	}
	return []Variable{}
}

// Variables helper methods...
func (d *Debugger) createVariableReference(vars []Variable) int {
	d.nextVarRef++
	d.variableRefs[d.nextVarRef] = vars
	return d.nextVarRef
}

func (d *Debugger) getLocalVariables(vm *vm, frameIndex int) []Variable {
	vars := make([]Variable, 0)
	
	// Obtener variables del stack
	if vm.prg != nil && vm.prg.names != nil {
		for name, ref := range vm.prg.names {
			if ref.isVar || ref.isConst {
				var value Value
				if ref.idx >= 0 && ref.idx < len(vm.stack) {
					value = vm.stack[vm.sb+ref.idx]
				}
				vars = append(vars, Variable{
					Name:  name.String(),
					Value: value,
					Type:  d.getValueType(value),
				})
			}
		}
	}
	
	return vars
}

func (d *Debugger) getClosureVariables(vm *vm) []Variable {
	vars := make([]Variable, 0)
	
	if vm.stash != nil {
		for name, value := range vm.stash.values {
			vars = append(vars, Variable{
				Name:  name.String(),
				Value: value,
				Type:  d.getValueType(value),
			})
		}
	}
	
	return vars
}

func (d *Debugger) getGlobalVariables(vm *vm) []Variable {
	vars := make([]Variable, 0)
	
	global := vm.runtime.globalObject
	for _, prop := range global.propNames() {
		if value := global.getStr(prop, nil); value != nil {
			vars = append(vars, Variable{
				Name:  prop.String(),
				Value: value,
				Type:  d.getValueType(value),
			})
		}
	}
	
	return vars
}

func (d *Debugger) getValueType(v Value) string {
	if v == nil {
		return "undefined"
	}
	
	switch v.(type) {
	case valueNull:
		return "null"
	case valueBool:
		return "boolean"
	case valueInt, valueFloat:
		return "number"
	case valueString:
		return "string"
	case *Object:
		if obj := v.(*Object); obj != nil {
			return obj.className()
		}
		return "object"
	default:
		return "unknown"
	}
}

// executeCommand ejecuta un comando del debugger
func (d *Debugger) executeCommand(cmd DebugCommand) {
	switch cmd {
	case DebugContinue:
		d.Continue()
	case DebugStepOver:
		d.StepOver()
	case DebugStepInto:
		d.StepInto()
	case DebugStepOut:
		d.StepOut()
	case DebugPause:
		d.Pause()
	}
}

// SetLogger establece un logger para debugging del debugger
func (d *Debugger) SetLogger(logger *log.Logger) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger = logger
}

// Métodos de Breakpoint

func (bp *Breakpoint) ID() int {
	return bp.id
}

func (bp *Breakpoint) SourcePos() file.Position {
	return file.Position{
		Filename: bp.file,
		Line:     bp.line,
		Column:   bp.column,
	}
}

func (bp *Breakpoint) Enable() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.enabled = true
}

func (bp *Breakpoint) Disable() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.enabled = false
}

func (bp *Breakpoint) IsEnabled() bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.enabled
}

func (bp *Breakpoint) Hits() int {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.hits
}

// stackFrame implementa StackFrame
type stackFrame struct {
	prg *Program
	pc  int
}

func (f *stackFrame) FuncName() string {
	if f.prg != nil && f.prg.funcName != "" {
		return f.prg.funcName.String()
	}
	return "<anonymous>"
}

func (f *stackFrame) Position() string {
	if f.prg != nil && f.pc < len(f.prg.src) {
		if pos := f.prg.src[f.pc]; pos.srcPos > 0 {
			position := f.prg.Position(pos.srcPos)
			return fmt.Sprintf("%s:%d:%d", position.Filename, position.Line, position.Column)
		}
	}
	return "<unknown>"
}
```

## Explicación de la Implementación

### 1. Estructura Principal
La estructura `Debugger` mantiene todo el estado del debugging:
- `runtime`: Referencia al runtime de Goja
- `breakpoints`: Mapa de breakpoints por ID
- `handler`: Función callback para manejar eventos
- `flags`: Estado del debugger (pausado, stepping, etc.)

### 2. Sistema de Breakpoints
- Los breakpoints se identifican por archivo, línea y columna
- Cada breakpoint tiene un ID único y contador de hits
- Se pueden habilitar/deshabilitar individualmente

### 3. Modos de Stepping
- **StepInto**: Entra en todas las funciones
- **StepOver**: Ejecuta funciones completas sin entrar
- **StepOut**: Sale de la función actual
- **Continue**: Ejecuta hasta el próximo breakpoint

### 4. Inspección de Variables
El debugger puede inspeccionar:
- Variables locales del stack
- Variables en closures (stash)
- Variables globales

### 5. Thread Safety
- Usa mutex para proteger el estado compartido
- El método `checkBreakpoint` está optimizado para rendimiento

## Integración con la VM

El debugger se integra con la VM a través de dos métodos principales:
- `checkBreakpoint`: Llamado en cada instrucción para verificar si debe pausar
- `handlePause`: Llamado cuando la VM se pausa para manejar el evento

## Limitaciones Conocidas

### Step-Into en Closures

El debugger actual no puede hacer step-into directamente en closures (funciones almacenadas en variables). Por ejemplo:

```javascript
let miFunc = crearClosure();
miFunc(); // Step-into no funcionará aquí
```

**Solución recomendada**: Establecer breakpoints dentro de las funciones closure. Ver [08_GUIA_MEJORA_CLOSURES.md](08_GUIA_MEJORA_CLOSURES.md) para más detalles.

## Siguiente Paso

Continúa con [03_GUIA_IMPLEMENTACION_RUNTIME_VM.md](03_GUIA_IMPLEMENTACION_RUNTIME_VM.md) para integrar el debugger con el Runtime y la VM.