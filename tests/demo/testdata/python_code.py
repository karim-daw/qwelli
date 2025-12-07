# Python example code
def calculate_fibonacci(n):
    """Calculate the nth Fibonacci number using recursion."""
    if n <= 1:
        return n
    return calculate_fibonacci(n - 1) + calculate_fibonacci(n - 2)

def main():
    print("Fibonacci sequence generator")
    for i in range(10):
        result = calculate_fibonacci(i)
        print(f"F({i}) = {result}")

if __name__ == "__main__":
    main()

