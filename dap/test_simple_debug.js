// Simple test script for debugging
console.log("Starting debug test");

let x = 10;
let y = 20;

function add(a, b) {
    console.log("Inside add function");
    let result = a + b;
    return result;
}

console.log("Before calling add");
let sum = add(x, y);
console.log("Sum is:", sum);

// Test loop
for (let i = 0; i < 3; i++) {
    console.log("Loop iteration:", i);
}

console.log("Debug test completed");