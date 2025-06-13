#!/bin/bash

echo "Testing progress updates with carriage returns..."
echo ""

# Test 1: Simple progress bar
echo "Test 1: Progress bar"
for i in 0 25 50 75 100; do
    printf "\rProgress: %3d%%" $i
    sleep 0.5
done
echo ""

# Test 2: Spinner
echo -e "\nTest 2: Spinner"
spinner=( '⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏' )
for i in {1..20}; do
    printf "\r%s Processing..." "${spinner[$((i % 10))]}"
    sleep 0.2
done
printf "\r✓ Done!        \n"

# Test 3: Multi-line updates (like AI agents)
echo -e "\nTest 3: Multi-line status"
for i in {1..5}; do
    printf "\033[2K\rStep $i of 5: Initializing..."
    sleep 0.5
    printf "\033[2K\rStep $i of 5: Processing..."
    sleep 0.5
    printf "\033[2K\rStep $i of 5: Complete ✓"
    echo ""
done

echo -e "\nAll tests complete!"