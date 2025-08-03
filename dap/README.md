# Goja Debug Adapter Protocol (DAP) Server

Un servidor DAP completo para debugging de scripts JavaScript en el motor Goja, compatible con VS Code.

## Características

✅ **Breakpoints**: Soporte completo para breakpoints en líneas específicas  
✅ **Step Operations**: Step Into, Step Over, Step Out  
✅ **Variable Inspection**: Variables globales y locales  
✅ **Call Stack**: Stack completo con información de contexto  
✅ **REPL/Debug Console**: Evaluación de expresiones en tiempo real  
✅ **Native Function Handling**: Manejo inteligente de funciones nativas  
✅ **Flow Control**: Seguimiento del flujo de código sin paradas innecesarias  

## Construcción

```bash
./build.sh
```

## Uso

### Modo Normal
```bash
./gojs script.js
```

### Modo Debug
```bash
./gojs -d script.js              # Puerto por defecto 5678
./gojs -d -port 9000 script.js   # Puerto personalizado
```

## Configuración para VS Code

### 1. Crear extensión Goja (opcional)

Crea un archivo `package.json` para una extensión VS Code simple:

```json
{
    "name": "goja-debug",
    "displayName": "Goja Debug",
    "version": "0.1.0",
    "engines": {
        "vscode": "^1.50.0"
    },
    "categories": ["Debuggers"],
    "contributes": {
        "debuggers": [{
            "type": "goja",
            "label": "Goja Debug",
            "program": "./gojs",
            "configurationAttributes": {
                "launch": {
                    "required": ["program"],
                    "properties": {
                        "program": {
                            "type": "string",
                            "description": "Absolute path to the JavaScript file.",
                            "default": "${workspaceFolder}/main.js"
                        },
                        "debugServer": {
                            "type": "number",
                            "description": "Port for the debug adapter server.",
                            "default": 5678
                        },
                        "stopOnEntry": {
                            "type": "boolean",
                            "description": "Automatically stop after launch.",
                            "default": false
                        }
                    }
                }
            }
        }]
    }
}
```

### 2. Configuración launch.json

En tu proyecto, crea o actualiza `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "type": "goja",
            "request": "launch",
            "name": "Debug Goja Script",
            "program": "${workspaceFolder}/test-debug.js",
            "debugServer": 5678,
            "stopOnEntry": false
        }
    ]
}
```

### 3. Debugging

1. **Iniciar el servidor DAP:**
   ```bash
   ./gojs -d test-debug.js
   ```

2. **En VS Code:**
   - Abrir el archivo JavaScript que quieres debuggear
   - Poner breakpoints haciendo clic en el margen izquierdo
   - Presionar `F5` o usar "Run and Debug"
   - Seleccionar "Debug Goja Script"

3. **Funcionalidades disponibles:**
   - **Breakpoints**: Clic en margen izquierdo para agregar/quitar
   - **Step Into (F11)**: Entrar en funciones
   - **Step Over (F10)**: Ejecutar línea sin entrar en funciones
   - **Step Out (Shift+F11)**: Salir de función actual
   - **Continue (F5)**: Continuar ejecución
   - **Variables Panel**: Ver variables locales y globales
   - **Call Stack Panel**: Ver el stack de llamadas
   - **Debug Console**: Evaluar expresiones (`x + y`, `typeof variable`, etc.)

## Ejemplos de Uso

### Script de Prueba
```javascript
// test-debug.js
console.log("Starting debug test...");

var globalVar = "I am global";
var globalNumber = 42;

function testFunction(param1, param2) {
    var localVar = "I am local";
    var localNumber = param1 + param2;
    
    debugger; // Pausa automática aquí
    
    console.log("localVar:", localVar);
    console.log("localNumber:", localNumber);
    
    return localNumber;
}

var result = testFunction(10, 20);
console.log("Result:", result);
```

### Debugging Avanzado

1. **Inspección de Variables:**
   - Variables globales aparecen en el scope "Global"
   - Variables locales aparecen en el scope "Local"
   - Objetos complejos se pueden expandir

2. **Debug Console:**
   ```javascript
   // Evaluar expresiones en tiempo real
   localVar + " modified"
   typeof globalNumber
   globalObject.nested.inner
   ```

3. **Breakpoints Condicionales:**
   - Próximamente (no implementado aún)

## Arquitectura

### Componentes

- **`main.go`**: Punto de entrada principal
- **`gojs.go`**: CLI para ejecutar scripts
- **`adapter.go`**: Implementación del protocolo DAP
- **`protocol.go`**: Definiciones de tipos DAP

### Flujo de Funcionamiento

1. **Inicio**: `./gojs -d script.js` inicia el servidor DAP
2. **Conexión**: VS Code se conecta via TCP al puerto especificado
3. **Protocolo**: Comunicación via Debug Adapter Protocol (JSON-RPC sobre TCP)
4. **Ejecución**: El script se ejecuta con debugger habilitado
5. **Interacción**: Breakpoints, steps, y evaluaciones según comandos de VS Code

### Manejo de Funciones Nativas

El servidor maneja inteligentemente las funciones nativas de Go:
- Las funciones nativas se ejecutan sin parar el debugger
- El flujo se mantiene en código JavaScript relevante
- Se evitan paradas innecesarias en código interno

## Troubleshooting

### Error: "Port already in use"
```bash
# Cambiar puerto
./gojs -d -port 9000 script.js
```

### Error: "No debugger connection"
- Verificar que VS Code esté configurado con el puerto correcto
- Comprobar que no haya firewall bloqueando la conexión
- Reiniciar el servidor DAP

### Variables no aparecen
- Verificar que estés pausado en un breakpoint o step
- Las variables locales solo aparecen dentro de funciones
- Usar el Debug Console para evaluación manual

### Breakpoints no funcionan
- Verificar que el archivo en VS Code sea el mismo que se está ejecutando
- Los breakpoints solo funcionan en líneas ejecutables
- Evitar poner breakpoints en líneas vacías o comentarios

## Desarrollo

### Estructura del Proyecto
```
dap/
├── adapter.go          # Implementación DAP principal
├── protocol.go         # Tipos y estructuras DAP
├── main.go            # Punto de entrada
├── gojs.go            # CLI runner
├── build.sh           # Script de construcción
├── test-debug.js      # Script de prueba
└── README.md          # Esta documentación
```

### Extending

Para agregar nuevas funcionalidades:

1. **Nuevos comandos DAP**: Actualizar `protocol.go` y `adapter.go`
2. **Mejores variables**: Modificar `getLocalVariables()` y `getGlobalVariables()`
3. **Breakpoints condicionales**: Extender `handleSetBreakpoints()`

---

¡El servidor DAP está listo para usar! Disfruta debugging tus scripts JavaScript con Goja en VS Code. 🚀