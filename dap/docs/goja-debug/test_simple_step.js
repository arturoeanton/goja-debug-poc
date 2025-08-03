// Simple test for step-into

function calculateSum(a, b) {
    console.log("Inside calculateSum");
    var sum = a + b;
    return sum;
}

console.log("About to call calculateSum");
var result = calculateSum(5, 3); // Put breakpoint here and try step-into
console.log("Result is:", result);