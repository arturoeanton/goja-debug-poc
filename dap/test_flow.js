// Test to show execution flow
console.log("1. Start");

var x = 5;
console.log("2. x =", x);

var y = 10;
console.log("3. y =", y);

var sum = x + y;
console.log("4. sum =", sum);

console.log("5. Before if statement");

if (sum > 10) {
    console.log("6a. Inside if - sum is greater than 10");
} else {
    console.log("6b. Inside else - sum is 10 or less");
}

console.log("7. After if statement");
console.log("8. End of script");