// Simple test script for Goja debugging
console.log("Hello from Goja!");

let x = 10;
let y = 20;
let sum = x + y;

console.log("The sum of", x, "and", y, "is:", sum);

// Test function
function greet(name) {
    return "Hello, " + name + "!";
}

let message = greet("World");
console.log(message);

console.log("Script completed");