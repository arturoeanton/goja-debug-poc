console.log("Starting test...");

var x = 5;
var y = 10;
var sum = x + y;

console.log("Before if statement");

if (sum > 10) {
    console.log("Sum is greater than 10");
    console.log("This should execute");
} else {
    console.log("Sum is 10 or less");
    console.log("This should NOT execute");
}

console.log("After if statement");

// Another test
var test = true;
if (test) {
    console.log("Test is true");
} else {
    console.log("Test is false");
}

console.log("Done!");