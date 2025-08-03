// Test simple para investigar step-into en closures
console.log("=== Test Step-Into Closures ===");

// Caso 1: Closure simple
function crearFunc() {
    console.log("Dentro de crearFunc");
    return function() {
        console.log("Dentro del closure");  // Línea 8 - breakpoint aquí
        return 42;
    };
}

console.log("Creando closure...");
let miFuncion = crearFunc();

console.log("Llamando closure..."); // Línea 16
miFuncion();  // Línea 17 - step-into debería entrar aquí

// Caso 2: Función directa (para comparar)
function funcionDirecta() {
    console.log("Dentro de función directa");
    return 99;
}

console.log("Llamando función directa..."); // Línea 25
funcionDirecta(); // Línea 26 - step-into SÍ funciona aquí

console.log("=== Fin del test ===");