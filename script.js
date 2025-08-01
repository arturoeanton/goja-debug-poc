var x = 5;
var y = 10;
var sum = x + y;
printMessage("Sum of", x, "and", y, "is:", sum);

if (sum > 10) {
    printMessage("The sum is greater than 10.");
} else {
    printMessage("The sum is 10 or less.");
}

printMessage("End of script.");


function printMessage(message) {
    console.log(message);
}