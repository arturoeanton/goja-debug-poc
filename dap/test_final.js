// Final test for function stepping
console.log("1. Starting test");

function greet(name) {
    console.log("2. Inside greet function");
    return "Hello, " + name + "!";
}

console.log("3. Before calling greet");

// Set breakpoint on this line
let message = greet("World");  

console.log("4. After calling greet:", message);
console.log("5. Test completed");