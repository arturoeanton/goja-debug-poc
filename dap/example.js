// Example JavaScript file for debugging with Goja
console.log("Starting debug example");

// Function example
function factorial(n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

// Variables
let x = 5;
let result = factorial(x);
console.log(`Factorial of ${x} is: ${result}`);

// Object and array examples
let person = {
    name: "John Doe",
    age: 30,
    city: "New York"
};

let numbers = [1, 2, 3, 4, 5];

console.log("Person:", person);
console.log("Numbers:", numbers);

// Loop example
for (let i = 0; i < 3; i++) {
    console.log(`Loop iteration: ${i}`);
}

// Conditional example
if (result > 100) {
    console.log("Result is greater than 100");
} else {
    console.log("Result is less than or equal to 100");
}

console.log("Debug example completed");