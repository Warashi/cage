#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Temporary test directory
TEST_DIR=$(mktemp -d)
CAGE_BIN="./cage"

# Function to print test results
print_test_result() {
    local test_name="$1"
    local result="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC} $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Function to run a test
run_test() {
    local test_name="$1"
    local expected_result="$2"  # "success" or "failure"
    shift 2
    local command=("$@")
    
    if [ "$expected_result" = "success" ]; then
        if "${command[@]}" >/dev/null 2>&1; then
            print_test_result "$test_name" "PASS"
        else
            print_test_result "$test_name" "FAIL"
        fi
    else
        if "${command[@]}" >/dev/null 2>&1; then
            print_test_result "$test_name" "FAIL"
        else
            print_test_result "$test_name" "PASS"
        fi
    fi
}

# Cleanup function
cleanup() {
    rm -rf "$TEST_DIR"
}

trap cleanup EXIT

# Setup test environment
setup() {
    echo "Setting up test environment..."
    
    # Create test directories and files
    mkdir -p "$TEST_DIR/allowed"
    mkdir -p "$TEST_DIR/restricted"
    mkdir -p "$TEST_DIR/readable"
    
    echo "test content" > "$TEST_DIR/readable/test.txt"
    
    # Build cage if not present
    if [ ! -f "$CAGE_BIN" ]; then
        echo "Building cage..."
        go build -o cage main.go || {
            echo -e "${RED}Failed to build cage${NC}"
            exit 1
        }
    fi
    
    echo "Test environment ready."
    echo
}

# Test 1: Basic command execution (read-only operations should work)
test_basic_execution() {
    echo "Testing basic command execution..."
    
    run_test "Execute simple command (ls)" "success" \
        "$CAGE_BIN" ls -la "$TEST_DIR"
    
    run_test "Execute command with arguments" "success" \
        "$CAGE_BIN" echo "Hello, World!"
    
    run_test "Read file content" "success" \
        "$CAGE_BIN" cat "$TEST_DIR/readable/test.txt"
}

# Test 2: Write restrictions (default behavior)
test_write_restrictions() {
    echo -e "\nTesting write restrictions..."
    
    run_test "Write to file without permission (should fail)" "failure" \
        "$CAGE_BIN" sh -c "echo 'data' > '$TEST_DIR/restricted/file.txt'"
    
    run_test "Create directory without permission (should fail)" "failure" \
        "$CAGE_BIN" mkdir "$TEST_DIR/restricted/newdir"
    
    run_test "Delete file without permission (should fail)" "failure" \
        "$CAGE_BIN" rm "$TEST_DIR/readable/test.txt"
}

# Test 3: Single directory allow
test_single_allow() {
    echo -e "\nTesting single --allow flag..."
    
    run_test "Write to allowed directory" "success" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" sh -c "echo 'allowed' > '$TEST_DIR/allowed/file.txt'"
    
    run_test "Create file in allowed directory" "success" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" touch "$TEST_DIR/allowed/newfile.txt"
    
    run_test "Write to non-allowed directory (should fail)" "failure" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" sh -c "echo 'data' > '$TEST_DIR/restricted/file.txt'"
}

# Test 4: Multiple directory allow
test_multiple_allow() {
    echo -e "\nTesting multiple --allow flags..."
    
    mkdir -p "$TEST_DIR/allowed2"
    
    run_test "Write to first allowed directory" "success" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" --allow "$TEST_DIR/allowed2" \
        sh -c "echo 'data1' > '$TEST_DIR/allowed/multi1.txt'"
    
    run_test "Write to second allowed directory" "success" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" --allow "$TEST_DIR/allowed2" \
        sh -c "echo 'data2' > '$TEST_DIR/allowed2/multi2.txt'"
    
    run_test "Write to non-allowed directory (should fail)" "failure" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" --allow "$TEST_DIR/allowed2" \
        sh -c "echo 'data' > '$TEST_DIR/restricted/file.txt'"
}

# Test 5: Allow all flag
test_allow_all() {
    echo -e "\nTesting --allow-all flag..."
    
    run_test "Write anywhere with --allow-all" "success" \
        "$CAGE_BIN" --allow-all sh -c "echo 'unrestricted' > '$TEST_DIR/restricted/all.txt'"
    
    run_test "Create directory with --allow-all" "success" \
        "$CAGE_BIN" --allow-all mkdir "$TEST_DIR/restricted/alldir"
    
    run_test "Delete file with --allow-all" "success" \
        "$CAGE_BIN" --allow-all rm "$TEST_DIR/restricted/all.txt"
}

# Test 6: Complex commands
test_complex_commands() {
    echo -e "\nTesting complex commands..."
    
    # Test with pipes
    run_test "Command with pipe (read-only)" "success" \
        "$CAGE_BIN" sh -c "echo 'test' | grep 'test'"
    
    # Test with redirection to allowed path
    run_test "Redirection to allowed path" "success" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" sh -c "echo 'redirected' > '$TEST_DIR/allowed/redirect.txt'"
    
    # Test with command separator
    run_test "Multiple commands with separator" "success" \
        "$CAGE_BIN" --allow "$TEST_DIR/allowed" sh -c "cd '$TEST_DIR' && echo 'test' > allowed/cmd.txt"
}

# Test 7: Edge cases
test_edge_cases() {
    echo -e "\nTesting edge cases..."
    
    # Test with non-existent command
    run_test "Non-existent command (should fail)" "failure" \
        "$CAGE_BIN" nonexistentcommand
    
    # Test with empty allow path
    run_test "Empty command (should fail)" "failure" \
        "$CAGE_BIN"
    
    # Test read access to various system directories
    run_test "Read /etc/passwd" "success" \
        "$CAGE_BIN" cat /etc/passwd
}

# Platform-specific tests
test_platform_specific() {
    echo -e "\nTesting platform-specific features..."
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Running Linux-specific tests..."
        # Linux specific tests here
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "Running macOS-specific tests..."
        # macOS specific tests here
    fi
}

# Main test execution
main() {
    echo "=== Cage E2E Test Suite ==="
    echo "Platform: $OSTYPE"
    echo
    
    setup
    
    test_basic_execution
    test_write_restrictions
    test_single_allow
    test_multiple_allow
    test_allow_all
    test_complex_commands
    test_edge_cases
    test_platform_specific
    
    echo
    echo "=== Test Summary ==="
    echo "Tests run: $TESTS_RUN"
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Run the tests
main