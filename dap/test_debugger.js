// Test script to verify debugger behavior
console.log("Starting test...");

var counter = 0;
for (var i = 0; i < 3; i++) {
    counter = counter + 1;
    console.log("Iteration", i, "counter:", counter);
}

console.log("Final counter:", counter);
console.log("Test complete.");