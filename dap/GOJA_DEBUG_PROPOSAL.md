# Propuesta de cambios para debug.go en Goja

## Problema actual

El debugger de Goja no expone las variables locales de las funciones. Solo podemos acceder a:
- Variables globales
- Stack frames (pero sin sus scopes)
- Posición en el código

## Cambios necesarios en Goja

### 1. Extender StackFrame para incluir scope

```go
// En debug.go
type StackFrame struct {
    // ... campos existentes ...
    
    // Nuevos campos para debugging
    scope      *object     // El scope actual del frame
    localVars  valueStack  // Variables locales
    arguments  []Value     // Argumentos de la función
}

// Nuevos métodos
func (f *StackFrame) GetLocalScope() *Object {
    if f.scope != nil {
        return f.scope.ToObject(f.vm)
    }
    return nil
}

func (f *StackFrame) GetLocalVariables() map[string]Value {
    vars := make(map[string]Value)
    // Iterar sobre las variables locales en el scope
    if f.scope != nil {
        // Acceder a las propiedades del scope
        for _, key := range f.scope.propNames {
            vars[key] = f.scope.getStr(key, nil)
        }
    }
    return vars
}

func (f *StackFrame) GetArguments() []Value {
    return f.arguments
}

func (f *StackFrame) GetThis() Value {
    return f.funcObj
}
```

### 2. Capturar el scope cuando se crea el StackFrame

```go
// En vm.go o donde se crean los stack frames
func (vm *vm) captureStackFrame() StackFrame {
    frame := StackFrame{
        // ... campos existentes ...
    }
    
    // Capturar el scope actual
    if vm.stash != nil {
        frame.scope = vm.stash
    }
    
    // Capturar argumentos si estamos en una función
    if vm.args > 0 {
        frame.arguments = make([]Value, vm.args)
        for i := 0; i < vm.args; i++ {
            frame.arguments[i] = vm.stack[vm.sp-vm.args+i]
        }
    }
    
    return frame
}
```

### 3. Extender DebuggerState para incluir información del scope actual

```go
type DebuggerState struct {
    PC        int
    SourcePos Position
    Breakpoint *Breakpoint
    
    // Nuevo campo
    CurrentScope map[string]Value  // Variables disponibles en el scope actual
}
```

### 4. En el debugHandler, poblar CurrentScope

```go
func (d *debugger) debugHandler(vm *vm, pc int) {
    state := &DebuggerState{
        PC:        pc,
        SourcePos: vm.prg.sourcePos(pc),
        // ... otros campos ...
    }
    
    // Poblar CurrentScope
    state.CurrentScope = make(map[string]Value)
    
    // Variables locales del stash actual
    if vm.stash != nil {
        for _, name := range vm.stash.names {
            state.CurrentScope[name] = vm.stash.getStr(name, nil)
        }
    }
    
    // Argumentos de la función actual
    if vm.args > 0 {
        for i := 0; i < vm.args; i++ {
            argName := fmt.Sprintf("arg%d", i)
            if vm.prg != nil && vm.curFunc != nil {
                // Intentar obtener el nombre real del parámetro
                // desde la información de la función
            }
            state.CurrentScope[argName] = vm.stack[vm.sp-vm.args+i]
        }
    }
    
    // Llamar al handler del usuario
    cmd := d.handler(state)
    // ...
}
```

## Uso en el DAP adapter

Con estos cambios, podríamos hacer:

```go
func (da *DebugAdapter) getFrameVariables(frameID int) []Variable {
    frame := da.frameMap[frameID]
    
    // Obtener variables locales
    localVars := frame.GetLocalVariables()
    for name, value := range localVars {
        variables = append(variables, Variable{
            Name:  name,
            Value: value.String(),
            Type:  getValueType(value),
        })
    }
    
    // Obtener argumentos
    args := frame.GetArguments()
    for i, arg := range args {
        variables = append(variables, Variable{
            Name:  fmt.Sprintf("argument[%d]", i),
            Value: arg.String(),
            Type:  getValueType(arg),
        })
    }
    
    return variables
}
```

## Alternativa más simple

Si los cambios anteriores son muy complejos, una alternativa más simple sería:

1. Agregar un método a Runtime para evaluar expresiones en el contexto del frame actual:

```go
func (r *Runtime) EvaluateInFrame(frameIndex int, expression string) (Value, error) {
    // Guardar el contexto actual
    oldStash := r.vm.stash
    oldSp := r.vm.sp
    
    // Cambiar al contexto del frame solicitado
    if frameIndex < len(r.vm.callStack) {
        frame := r.vm.callStack[frameIndex]
        r.vm.stash = frame.stash
        r.vm.sp = frame.sp
    }
    
    // Evaluar la expresión
    result, err := r.RunString(expression)
    
    // Restaurar el contexto
    r.vm.stash = oldStash
    r.vm.sp = oldSp
    
    return result, err
}
```

Esto permitiría al DAP adapter evaluar variables específicas cuando el usuario las solicite.

## Conclusión

Para un soporte completo de debugging necesitamos:
1. Acceso al scope/stash de cada frame
2. Lista de variables locales y sus valores
3. Acceso a los argumentos de funciones
4. Posibilidad de evaluar expresiones en el contexto de un frame específico

¿Cuál de estos enfoques prefieres implementar en tu fork de Goja?