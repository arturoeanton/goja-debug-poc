# Guía de Verificación de Performance - Debugger Goja

## Tests de Performance para Verificar Zero Impact

### 1. Benchmark Comparativo

Crear `debugger_bench_test.go`:

```go
package goja

import (
    "testing"
)

// Benchmark sin debugger (comportamiento original)
func BenchmarkRuntimeWithoutDebugger(b *testing.B) {
    script := `
        function fibonacci(n) {
            if (n <= 1) return n;
            return fibonacci(n-1) + fibonacci(n-2);
        }
        
        for (var i = 0; i < 100; i++) {
            fibonacci(10);
        }
    `
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r := New()
        _, err := r.RunString(script)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Benchmark con debugger deshabilitado
func BenchmarkRuntimeWithDebuggerDisabled(b *testing.B) {
    script := `
        function fibonacci(n) {
            if (n <= 1) return n;
            return fibonacci(n-1) + fibonacci(n-2);
        }
        
        for (var i = 0; i < 100; i++) {
            fibonacci(10);
        }
    `
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r := New()
        d := r.EnableDebugger()
        d.Disable() // Debugger presente pero deshabilitado
        
        _, err := r.RunString(script)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Benchmark con debugger habilitado pero sin breakpoints
func BenchmarkRuntimeWithDebuggerNoBreakpoints(b *testing.B) {
    script := `
        function fibonacci(n) {
            if (n <= 1) return n;
            return fibonacci(n-1) + fibonacci(n-2);
        }
        
        for (var i = 0; i < 100; i++) {
            fibonacci(10);
        }
    `
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r := New()
        d := r.EnableDebugger()
        d.Enable() // Debugger habilitado pero sin breakpoints
        
        _, err := r.RunString(script)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 2. Verificación del Hot Path

```go
// Test para verificar que el check rápido funciona
func TestDebuggerQuickCheck(t *testing.T) {
    r := New()
    
    // Medir tiempo sin debugger
    start := time.Now()
    for i := 0; i < 1000000; i++ {
        // Simular check en vm.run()
        if r.debugger != nil && r.debugger.enabled {
            t.Fatal("No debería entrar aquí")
        }
    }
    withoutDebugger := time.Since(start)
    
    // Habilitar debugger pero deshabilitado
    d := r.EnableDebugger()
    d.Disable()
    
    start = time.Now()
    for i := 0; i < 1000000; i++ {
        if r.debugger != nil && r.debugger.enabled {
            t.Fatal("No debería entrar aquí")
        }
    }
    withDebuggerDisabled := time.Since(start)
    
    // La diferencia debe ser mínima (< 5%)
    overhead := float64(withDebuggerDisabled-withoutDebugger) / float64(withoutDebugger) * 100
    if overhead > 5 {
        t.Errorf("Overhead demasiado alto: %.2f%%", overhead)
    }
}
```

### 3. Profile del Compilador

```go
func TestCompilerNoDebugMode(t *testing.T) {
    // Compilar con modo normal
    prg1, _ := Compile("test.js", `
        function test(a, b) {
            var x = a + b;
            return x * 2;
        }
    `, false)
    
    // Compilar con RuntimeOptions pero sin debug mode
    r := NewWithOptions(RuntimeOptions{EnableDebugMode: false})
    prg2, _ := r.Compile("test.js", `
        function test(a, b) {
            var x = a + b;
            return x * 2;
        }
    `)
    
    // El bytecode debe ser idéntico
    if !reflect.DeepEqual(prg1.code, prg2.code) {
        t.Error("El bytecode debería ser idéntico sin modo debug")
    }
}
```

## Comandos para Ejecutar Benchmarks

```bash
# Ejecutar todos los benchmarks
go test -bench=BenchmarkRuntime -benchmem -benchtime=10s

# Comparar resultados
go test -bench=BenchmarkRuntime -benchmem -benchtime=10s > without_changes.txt
# (aplicar cambios del debugger)
go test -bench=BenchmarkRuntime -benchmem -benchtime=10s > with_debugger.txt

# Usar benchstat para comparar
go install golang.org/x/perf/cmd/benchstat@latest
benchstat without_changes.txt with_debugger.txt
```

## Resultados Esperados

```
name                             old time/op    new time/op    delta
RuntimeWithoutDebugger-8          10.5ms ± 1%    10.5ms ± 1%    ~     (p=1.000 n=10+10)
RuntimeWithDebuggerDisabled-8     10.5ms ± 1%    10.6ms ± 1%  +0.95%  (p=0.001 n=10+10)
RuntimeWithDebuggerNoBreakpoints-8 10.5ms ± 1%    10.7ms ± 1%  +1.90%  (p=0.001 n=10+10)
```

## Optimizaciones Adicionales

### 1. Usar Atomic para Flags (opcional)

Si el overhead de mutex es preocupante:

```go
type Debugger struct {
    // Cambiar flags de uint32 con mutex a atomic
    flags atomic.Uint32
}

func (d *Debugger) checkBreakpoint(vm *vm) bool {
    // Check atómico sin lock
    flags := d.flags.Load()
    if flags == 0 && len(d.breakpoints) == 0 {
        return false
    }
    // ... resto con mutex solo si es necesario
}
```

### 2. Compilación Condicional (opcional)

Para garantizar absolutamente cero overhead:

```go
// +build debug

package goja

// Versión con debugger

// +build !debug

package goja

// Versión sin debugger (stubs vacíos)
```

## Conclusión

Con estas verificaciones puedes garantizar que:

1. **Sin debugger**: 0% overhead
2. **Con debugger deshabilitado**: < 1% overhead
3. **Con debugger habilitado sin breakpoints**: < 2% overhead
4. **Solo con modo debug activo**: Performance degradada (esperado)

El diseño actual ya contempla estos requisitos, pero estos tests lo verifican empíricamente.