// Simple demo to showcase debugger features

// Global variables
var name = "Goja Debugger";
var version = 1.0;

// Function with local variables
function greet(userName) {
    var greeting = "Hello, " + userName + "!";
    var timestamp = new Date().toISOString();
    
    console.log(greeting);
    console.log("Timestamp:", timestamp);
    
    return greeting;
}

// Function with nested calls
function processData(data) {
    var processed = data.toUpperCase();
    var length = processed.length;
    
    // Nested function call
    var result = formatResult(processed, length);
    
    return result;
}

function formatResult(text, len) {
    var formatted = "[" + text + "] (length: " + len + ")";
    return formatted;
}

// Main execution
console.log("Starting debugger demo...");
console.log("Version:", version);

var userGreeting = greet("Developer");
console.log("Greeting returned:", userGreeting);

var data = "test data";
var processedData = processData(data);
console.log("Processed:", processedData);

// Array operations (native functions)
var numbers = [1, 2, 3, 4, 5];
var doubled = numbers.map(function(n) { return n * 2; });
console.log("Doubled numbers:", doubled);

console.log("Demo complete!");